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

func TestAgentInstallToken_TableName(t *testing.T) {
	assert.Equal(t, "agent_install_tokens", (AgentInstallToken{}).TableName())
}

func TestAgentInstallToken_IsExpired(t *testing.T) {
	past := &AgentInstallToken{ExpiresAt: time.Now().UTC().Add(-time.Hour)}
	future := &AgentInstallToken{ExpiresAt: time.Now().UTC().Add(time.Hour)}
	assert.True(t, past.IsExpired())
	assert.False(t, future.IsExpired())
}

// setupInstallTokenTest wires up a MockStore whose Atomic delegates to the
// same store, an agent with a populated AgentType (so template validation
// passes), and the per-test install-token repo + event repo. The returned
// installRepo is the seam tests use to control GetByAgentID semantics
// (exists / not-found / error).
func setupInstallTokenTest(t *testing.T) (*MockStore, *MockAgentInstallTokenRepository, *MockTokenRepository, properties.UUID, context.Context) {
	t.Helper()
	ms := setupMockStore(t)

	agentID := properties.UUID(uuid.New())
	agent := &Agent{
		BaseEntity: BaseEntity{ID: agentID},
		ProviderID: properties.UUID(uuid.New()),
		AgentType: &AgentType{
			BaseEntity:     BaseEntity{ID: properties.UUID(uuid.New())},
			CmdTemplate:    "curl {{.configUrl}} -H 'Authorization: Bearer {{.authToken}}'",
			ConfigTemplate: "agent: {{.name}}",
		},
	}

	agentRepo := NewMockAgentRepository(t)
	agentRepo.EXPECT().Get(mock.Anything, agentID).Return(agent, nil).Maybe()
	ms.EXPECT().AgentRepo().Return(agentRepo).Maybe()

	installRepo := NewMockAgentInstallTokenRepository(t)
	ms.EXPECT().AgentInstallTokenRepo().Return(installRepo).Maybe()

	tokenRepo := NewMockTokenRepository(t)
	ms.EXPECT().TokenRepo().Return(tokenRepo).Maybe()

	eventRepo := NewMockEventRepository(t)
	eventRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Maybe()
	ms.EXPECT().EventRepo().Return(eventRepo).Maybe()

	ctx := auth.WithIdentity(context.Background(), &auth.Identity{
		Role: auth.RoleAdmin,
		ID:   properties.UUID(uuid.New()),
		Name: "Test Admin",
	})
	return ms, installRepo, tokenRepo, agentID, ctx
}

func TestAgentInstallTokenCommander_Create(t *testing.T) {
	t.Run("happy path returns entity with PlainToken, PlainBootstrapToken, BootstrapTokenID", func(t *testing.T) {
		ms, installRepo, tokenRepo, agentID, ctx := setupInstallTokenTest(t)

		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(nil, NotFoundError{}).Once()

		var createdToken *Token
		tokenRepo.EXPECT().Create(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, tok *Token) error {
				tok.ID = properties.UUID(uuid.New())
				createdToken = tok
				return nil
			}).Once()

		var created *AgentInstallToken
		installRepo.EXPECT().Create(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, c *AgentInstallToken) error {
				created = c
				return nil
			}).Once()

		tok, err := NewAgentInstallTokenCommander(ms).Create(ctx, agentID)
		assert.NoError(t, err)
		assert.NotNil(t, tok)
		assert.NotEmpty(t, tok.PlainToken, "PlainToken should be populated on the returned entity")
		assert.Equal(t, HashTokenValue(tok.PlainToken), tok.TokenHashed, "hash must match plaintext")
		assert.WithinDuration(t, time.Now().UTC().Add(5*time.Minute), tok.ExpiresAt, 5*time.Second)
		assert.NotEmpty(t, tok.PlainBootstrapToken, "PlainBootstrapToken should be populated")
		assert.NotNil(t, tok.BootstrapTokenID, "BootstrapTokenID should be set")
		assert.Equal(t, createdToken.ID, *tok.BootstrapTokenID)
		assert.Equal(t, auth.RoleAgent, createdToken.Role)
		assert.Equal(t, HashTokenValue(tok.PlainBootstrapToken), createdToken.HashedValue)
		assert.WithinDuration(t, tok.ExpiresAt, createdToken.ExpireAt, time.Second)
		// The persisted entity must match the returned one — same hash, same ID.
		assert.Equal(t, tok.TokenHashed, created.TokenHashed)
	})

	t.Run("conflict when one already exists", func(t *testing.T) {
		ms, installRepo, _, agentID, ctx := setupInstallTokenTest(t)
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(&AgentInstallToken{AgentID: agentID}, nil).Once()

		_, err := NewAgentInstallTokenCommander(ms).Create(ctx, agentID)
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
		_, err := NewAgentInstallTokenCommander(ms).Create(ctx, agentID)
		assert.ErrorAs(t, err, &InvalidInputError{})
	})
}

func TestAgentInstallTokenCommander_Regenerate(t *testing.T) {
	t.Run("rotates token, mints fresh bootstrap, and deletes prior bootstrap", func(t *testing.T) {
		ms, installRepo, tokenRepo, agentID, ctx := setupInstallTokenTest(t)

		priorBootstrapID := properties.UUID(uuid.New())
		existing := &AgentInstallToken{
			BaseEntity:       BaseEntity{ID: properties.UUID(uuid.New())},
			AgentID:          agentID,
			TokenHashed:      HashTokenValue("old-token-value"),
			ExpiresAt:        time.Now().UTC().Add(-time.Minute),
			BootstrapTokenID: &priorBootstrapID,
		}
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).Return(existing, nil).Once()
		tokenRepo.EXPECT().Delete(mock.Anything, priorBootstrapID).Return(nil).Once()
		var createdToken *Token
		tokenRepo.EXPECT().Create(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, tok *Token) error {
				tok.ID = properties.UUID(uuid.New())
				createdToken = tok
				return nil
			}).Once()
		installRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

		tok, err := NewAgentInstallTokenCommander(ms).Regenerate(ctx, agentID)
		assert.NoError(t, err)
		assert.NotEmpty(t, tok.PlainToken)
		assert.NotEqual(t, HashTokenValue("old-token-value"), tok.TokenHashed)
		assert.Equal(t, HashTokenValue(tok.PlainToken), tok.TokenHashed)
		assert.True(t, tok.ExpiresAt.After(time.Now().UTC()))
		assert.NotEmpty(t, tok.PlainBootstrapToken)
		assert.NotNil(t, tok.BootstrapTokenID)
		assert.Equal(t, createdToken.ID, *tok.BootstrapTokenID)
		assert.NotEqual(t, priorBootstrapID, *tok.BootstrapTokenID)
	})

	t.Run("not found when none exists", func(t *testing.T) {
		ms, installRepo, _, agentID, ctx := setupInstallTokenTest(t)
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(nil, NotFoundError{}).Once()

		_, err := NewAgentInstallTokenCommander(ms).Regenerate(ctx, agentID)
		assert.ErrorAs(t, err, &NotFoundError{})
	})
}

func TestAgentInstallTokenCommander_Revoke(t *testing.T) {
	t.Run("deletes existing row and its bootstrap token", func(t *testing.T) {
		ms, installRepo, tokenRepo, agentID, ctx := setupInstallTokenTest(t)
		bootstrapID := properties.UUID(uuid.New())
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(&AgentInstallToken{AgentID: agentID, BootstrapTokenID: &bootstrapID}, nil).Once()
		tokenRepo.EXPECT().Delete(mock.Anything, bootstrapID).Return(nil).Once()
		installRepo.EXPECT().DeleteByAgentID(mock.Anything, agentID).Return(nil).Once()

		err := NewAgentInstallTokenCommander(ms).Revoke(ctx, agentID)
		assert.NoError(t, err)
	})

	t.Run("deletes row without bootstrap when BootstrapTokenID is nil", func(t *testing.T) {
		ms, installRepo, _, agentID, ctx := setupInstallTokenTest(t)
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(&AgentInstallToken{AgentID: agentID}, nil).Once()
		installRepo.EXPECT().DeleteByAgentID(mock.Anything, agentID).Return(nil).Once()

		err := NewAgentInstallTokenCommander(ms).Revoke(ctx, agentID)
		assert.NoError(t, err)
	})

	t.Run("not found when nothing to revoke", func(t *testing.T) {
		ms, installRepo, _, agentID, ctx := setupInstallTokenTest(t)
		installRepo.EXPECT().GetByAgentID(mock.Anything, agentID).
			Return(nil, NotFoundError{}).Once()

		err := NewAgentInstallTokenCommander(ms).Revoke(ctx, agentID)
		assert.ErrorAs(t, err, &NotFoundError{})
	})
}
