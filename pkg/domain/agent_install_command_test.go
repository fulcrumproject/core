package domain

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAgentInstallCommand_TableName(t *testing.T) {
	assert.Equal(t, "agent_install_commands", (AgentInstallCommand{}).TableName())
}

func TestAgentInstallCommand_IsExpired(t *testing.T) {
	past := &AgentInstallCommand{ExpiresAt: time.Now().UTC().Add(-time.Hour)}
	future := &AgentInstallCommand{ExpiresAt: time.Now().UTC().Add(time.Hour)}
	assert.True(t, past.IsExpired())
	assert.False(t, future.IsExpired())
}

// setupInstallCommandTest wires up a MockStore whose Atomic delegates to the
// same store, an agent with a populated AgentType (so template validation
// passes), and the per-test install-command repo + event repo. The returned
// installRepo is the seam tests use to control GetByAgentID semantics
// (exists / not-found / error).
func setupInstallCommandTest(t *testing.T) (*MockStore, *MockAgentInstallCommandRepository, properties.UUID, context.Context) {
	t.Helper()
	ms := setupMockStore(t)

	agentID := properties.UUID(uuid.New())
	agent := &Agent{
		BaseEntity: BaseEntity{ID: agentID},
		ProviderID: properties.UUID(uuid.New()),
		AgentType: &AgentType{
			BaseEntity:  BaseEntity{ID: properties.UUID(uuid.New())},
			CmdTemplate: "curl {{.configUrl}}",
		},
	}

	agentRepo := NewMockAgentRepository(t)
	agentRepo.EXPECT().Get(mock.Anything, agentID).Return(agent, nil).Maybe()
	ms.EXPECT().AgentRepo().Return(agentRepo).Maybe()

	installRepo := NewMockAgentInstallCommandRepository(t)
	ms.EXPECT().AgentInstallCommandRepo().Return(installRepo).Maybe()

	eventRepo := NewMockEventRepository(t)
	eventRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Maybe()
	ms.EXPECT().EventRepo().Return(eventRepo).Maybe()

	ctx := auth.WithIdentity(context.Background(), &auth.Identity{
		Role: auth.RoleAdmin,
		ID:   properties.UUID(uuid.New()),
		Name: "Test Admin",
	})
	return ms, installRepo, agentID, ctx
}

func TestAgentInstallCommandCommander_Create(t *testing.T) {
	t.Run("happy path returns entity with PlainToken and stores hash", func(t *testing.T) {
		ms, installRepo, agentID, ctx := setupInstallCommandTest(t)

		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(nil, NotFoundError{}).Once()

		var created *AgentInstallCommand
		installRepo.EXPECT().Create(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, c *AgentInstallCommand) error {
				created = c
				return nil
			}).Once()

		cmd, err := NewAgentInstallCommandCommander(ms, time.Hour).Create(ctx, agentID)
		assert.NoError(t, err)
		assert.NotNil(t, cmd)
		assert.NotEmpty(t, cmd.PlainToken, "PlainToken should be populated on the returned entity")
		assert.Equal(t, HashTokenValue(cmd.PlainToken), cmd.TokenHashed, "hash must match plaintext")
		assert.WithinDuration(t, time.Now().UTC().Add(time.Hour), cmd.ExpiresAt, 5*time.Second)
		// The persisted entity must match the returned one — same hash, same ID.
		assert.Equal(t, cmd.TokenHashed, created.TokenHashed)
	})

	t.Run("conflict when one already exists", func(t *testing.T) {
		ms, installRepo, agentID, ctx := setupInstallCommandTest(t)
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(&AgentInstallCommand{AgentID: agentID}, nil).Once()

		_, err := NewAgentInstallCommandCommander(ms, time.Hour).Create(ctx, agentID)
		assert.ErrorAs(t, err, &ConflictError{})
	})

	t.Run("invalid input when agent type has no cmd template", func(t *testing.T) {
		ms := setupMockStore(t)
		agentID := properties.UUID(uuid.New())
		agent := &Agent{
			BaseEntity: BaseEntity{ID: agentID},
			AgentType:  &AgentType{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}},
		}
		agentRepo := NewMockAgentRepository(t)
		agentRepo.EXPECT().Get(mock.Anything, agentID).Return(agent, nil).Once()
		ms.EXPECT().AgentRepo().Return(agentRepo).Once()

		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})
		_, err := NewAgentInstallCommandCommander(ms, time.Hour).Create(ctx, agentID)
		assert.ErrorAs(t, err, &InvalidInputError{})
	})
}

func TestAgentInstallCommandCommander_Regenerate(t *testing.T) {
	t.Run("rotates token and populates new PlainToken", func(t *testing.T) {
		ms, installRepo, agentID, ctx := setupInstallCommandTest(t)

		existing := &AgentInstallCommand{
			BaseEntity:  BaseEntity{ID: properties.UUID(uuid.New())},
			AgentID:     agentID,
			TokenHashed: HashTokenValue("old-token-value"),
			ExpiresAt:   time.Now().UTC().Add(-time.Minute),
		}
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).Return(existing, nil).Once()
		installRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

		cmd, err := NewAgentInstallCommandCommander(ms, time.Hour).Regenerate(ctx, agentID)
		assert.NoError(t, err)
		assert.NotEmpty(t, cmd.PlainToken)
		assert.NotEqual(t, HashTokenValue("old-token-value"), cmd.TokenHashed)
		assert.Equal(t, HashTokenValue(cmd.PlainToken), cmd.TokenHashed)
		assert.True(t, cmd.ExpiresAt.After(time.Now().UTC()))
	})

	t.Run("not found when none exists", func(t *testing.T) {
		ms, installRepo, agentID, ctx := setupInstallCommandTest(t)
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(nil, NotFoundError{}).Once()

		_, err := NewAgentInstallCommandCommander(ms, time.Hour).Regenerate(ctx, agentID)
		assert.ErrorAs(t, err, &NotFoundError{})
	})
}

func TestAgentInstallCommandCommander_Revoke(t *testing.T) {
	t.Run("deletes existing row", func(t *testing.T) {
		ms, installRepo, agentID, ctx := setupInstallCommandTest(t)
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(&AgentInstallCommand{AgentID: agentID}, nil).Once()
		installRepo.EXPECT().DeleteByAgentID(mock.Anything, agentID).Return(nil).Once()

		err := NewAgentInstallCommandCommander(ms, time.Hour).Revoke(ctx, agentID)
		assert.NoError(t, err)
	})

	t.Run("not found when nothing to revoke", func(t *testing.T) {
		ms, installRepo, agentID, ctx := setupInstallCommandTest(t)
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(nil, NotFoundError{}).Once()

		err := NewAgentInstallCommandCommander(ms, time.Hour).Revoke(ctx, agentID)
		assert.ErrorAs(t, err, &NotFoundError{})
	})
}
