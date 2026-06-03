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

func TestInstallToken_TableName(t *testing.T) {
	assert.Equal(t, "install_tokens", (InstallToken{}).TableName())
}

func TestInstallToken_IsExpired(t *testing.T) {
	past := &InstallToken{ExpiresAt: time.Now().UTC().Add(-time.Hour)}
	future := &InstallToken{ExpiresAt: time.Now().UTC().Add(time.Hour)}
	assert.True(t, past.IsExpired())
	assert.False(t, future.IsExpired())
}

// installTokenFixture bundles the per-entity-type setup the commander tests
// need: a MockStore whose Atomic delegates to itself, an entity owning a
// non-empty TemplateValidation so HasInstallTemplates() passes, and the two
// always-needed sub-repos (install-token + token + event).
type installTokenFixture struct {
	store       *MockStore
	installRepo *MockInstallTokenRepository
	tokenRepo   *MockTokenRepository
	entityID    properties.UUID
	ctx         context.Context
}

func setupInstallTokenFixture(t *testing.T, entityType InstallTokenEntityType) installTokenFixture {
	t.Helper()
	ms := setupMockStore(t)

	tv := TemplateValidation{
		CmdTemplate:    "curl {{.configUrl}} -H 'Authorization: Bearer {{.authToken}}'",
		ConfigTemplate: "{{.name}}",
	}
	entityID := properties.UUID(uuid.New())

	switch entityType {
	case InstallTokenEntityTypeAgent:
		agent := &Agent{
			BaseEntity: BaseEntity{ID: entityID},
			ProviderID: properties.UUID(uuid.New()),
			AgentType: &AgentType{
				BaseEntity:         BaseEntity{ID: properties.UUID(uuid.New())},
				TemplateValidation: tv,
			},
		}
		agentRepo := NewMockAgentRepository(t)
		agentRepo.EXPECT().Get(mock.Anything, entityID).Return(agent, nil).Maybe()
		ms.EXPECT().AgentRepo().Return(agentRepo).Maybe()
	case InstallTokenEntityTypeInfrastructure:
		infra := &Infrastructure{
			BaseEntity: BaseEntity{ID: entityID},
			ProviderID: properties.UUID(uuid.New()),
			InfrastructureType: &InfrastructureType{
				BaseEntity:         BaseEntity{ID: properties.UUID(uuid.New())},
				TemplateValidation: tv,
			},
		}
		infraRepo := NewMockInfrastructureRepository(t)
		infraRepo.EXPECT().Get(mock.Anything, entityID).Return(infra, nil).Maybe()
		ms.EXPECT().InfrastructureRepo().Return(infraRepo).Maybe()
	}

	installRepo := NewMockInstallTokenRepository(t)
	ms.EXPECT().InstallTokenRepo().Return(installRepo).Maybe()

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
	return installTokenFixture{
		store:       ms,
		installRepo: installRepo,
		tokenRepo:   tokenRepo,
		entityID:    entityID,
		ctx:         ctx,
	}
}

func TestInstallTokenCommander_Create(t *testing.T) {
	entityCases := []struct {
		name       string
		entityType InstallTokenEntityType
	}{
		{"agent", InstallTokenEntityTypeAgent},
		{"infrastructure", InstallTokenEntityTypeInfrastructure},
	}
	for _, tc := range entityCases {
		t.Run(tc.name+" happy path returns entity with PlainToken, PlainBootstrapToken, BootstrapTokenID", func(t *testing.T) {
			f := setupInstallTokenFixture(t, tc.entityType)

			f.installRepo.EXPECT().GetByEntity(mock.Anything, tc.entityType, f.entityID).
				Return(nil, NotFoundError{}).Once()

			var createdToken *Token
			f.tokenRepo.EXPECT().Create(mock.Anything, mock.Anything).
				RunAndReturn(func(_ context.Context, tok *Token) error {
					tok.ID = properties.UUID(uuid.New())
					createdToken = tok
					return nil
				}).Once()

			var persisted *InstallToken
			f.installRepo.EXPECT().Create(mock.Anything, mock.Anything).
				RunAndReturn(func(_ context.Context, c *InstallToken) error {
					persisted = c
					return nil
				}).Once()

			tok, err := NewInstallTokenCommander(f.store).Create(f.ctx, tc.entityType, f.entityID)
			assert.NoError(t, err)
			assert.NotNil(t, tok)
			assert.Equal(t, tc.entityType, tok.EntityType)
			assert.Equal(t, f.entityID, tok.EntityID)
			assert.NotEmpty(t, tok.PlainToken)
			assert.Equal(t, HashTokenValue(tok.PlainToken), tok.TokenHashed)
			assert.WithinDuration(t, time.Now().UTC().Add(installTokenTTL), tok.ExpiresAt, 5*time.Second)
			assert.NotEmpty(t, tok.PlainBootstrapToken)
			assert.NotNil(t, tok.BootstrapTokenID)
			assert.Equal(t, createdToken.ID, *tok.BootstrapTokenID)
			assert.Equal(t, auth.RoleAgent, createdToken.Role)
			assert.Equal(t, HashTokenValue(tok.PlainBootstrapToken), createdToken.HashedValue)
			assert.WithinDuration(t, tok.ExpiresAt, createdToken.ExpireAt, time.Second)
			// Persisted row matches the returned token.
			assert.Equal(t, tok.TokenHashed, persisted.TokenHashed)
			assert.Equal(t, tok.EntityType, persisted.EntityType)
			assert.Equal(t, tok.EntityID, persisted.EntityID)
		})
	}

	t.Run("conflict when one already exists", func(t *testing.T) {
		f := setupInstallTokenFixture(t, InstallTokenEntityTypeAgent)
		f.installRepo.EXPECT().GetByEntity(mock.Anything, InstallTokenEntityTypeAgent, f.entityID).
			Return(&InstallToken{EntityType: InstallTokenEntityTypeAgent, EntityID: f.entityID}, nil).Once()

		_, err := NewInstallTokenCommander(f.store).Create(f.ctx, InstallTokenEntityTypeAgent, f.entityID)
		assert.ErrorAs(t, err, &ConflictError{})
	})

	t.Run("invalid input when agent type has no install templates", func(t *testing.T) {
		ms := setupMockStore(t)
		entityID := properties.UUID(uuid.New())
		agent := &Agent{
			BaseEntity: BaseEntity{ID: entityID},
			AgentType:  &AgentType{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}},
		}
		agentRepo := NewMockAgentRepository(t)
		agentRepo.EXPECT().Get(mock.Anything, entityID).Return(agent, nil).Once()
		ms.EXPECT().AgentRepo().Return(agentRepo).Once()

		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})
		_, err := NewInstallTokenCommander(ms).Create(ctx, InstallTokenEntityTypeAgent, entityID)
		assert.ErrorAs(t, err, &InvalidInputError{})
	})

	t.Run("invalid input when infrastructure type has no install templates", func(t *testing.T) {
		ms := setupMockStore(t)
		entityID := properties.UUID(uuid.New())
		infra := &Infrastructure{
			BaseEntity:         BaseEntity{ID: entityID},
			InfrastructureType: &InfrastructureType{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}},
		}
		infraRepo := NewMockInfrastructureRepository(t)
		infraRepo.EXPECT().Get(mock.Anything, entityID).Return(infra, nil).Once()
		ms.EXPECT().InfrastructureRepo().Return(infraRepo).Once()

		ctx := auth.WithIdentity(context.Background(), &auth.Identity{Role: auth.RoleAdmin, ID: properties.UUID(uuid.New())})
		_, err := NewInstallTokenCommander(ms).Create(ctx, InstallTokenEntityTypeInfrastructure, entityID)
		assert.ErrorAs(t, err, &InvalidInputError{})
	})
}

func TestInstallTokenCommander_Regenerate(t *testing.T) {
	entityCases := []struct {
		name       string
		entityType InstallTokenEntityType
	}{
		{"agent", InstallTokenEntityTypeAgent},
		{"infrastructure", InstallTokenEntityTypeInfrastructure},
	}
	for _, tc := range entityCases {
		t.Run(tc.name+" rotates token, mints fresh bootstrap, and deletes prior bootstrap", func(t *testing.T) {
			f := setupInstallTokenFixture(t, tc.entityType)

			priorBootstrapID := properties.UUID(uuid.New())
			existing := &InstallToken{
				BaseEntity:       BaseEntity{ID: properties.UUID(uuid.New())},
				EntityType:       tc.entityType,
				EntityID:         f.entityID,
				TokenHashed:      HashTokenValue("old-token-value"),
				ExpiresAt:        time.Now().UTC().Add(-time.Minute),
				BootstrapTokenID: &priorBootstrapID,
			}
			f.installRepo.EXPECT().GetByEntity(mock.Anything, tc.entityType, f.entityID).Return(existing, nil).Once()
			f.tokenRepo.EXPECT().Delete(mock.Anything, priorBootstrapID).Return(nil).Once()
			var createdToken *Token
			f.tokenRepo.EXPECT().Create(mock.Anything, mock.Anything).
				RunAndReturn(func(_ context.Context, tok *Token) error {
					tok.ID = properties.UUID(uuid.New())
					createdToken = tok
					return nil
				}).Once()
			f.installRepo.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

			tok, err := NewInstallTokenCommander(f.store).Regenerate(f.ctx, tc.entityType, f.entityID)
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
	}

	t.Run("not found when none exists", func(t *testing.T) {
		f := setupInstallTokenFixture(t, InstallTokenEntityTypeAgent)
		f.installRepo.EXPECT().GetByEntity(mock.Anything, InstallTokenEntityTypeAgent, f.entityID).
			Return(nil, NotFoundError{}).Once()

		_, err := NewInstallTokenCommander(f.store).Regenerate(f.ctx, InstallTokenEntityTypeAgent, f.entityID)
		assert.ErrorAs(t, err, &NotFoundError{})
	})
}

func TestInstallTokenCommander_Revoke(t *testing.T) {
	entityCases := []struct {
		name       string
		entityType InstallTokenEntityType
	}{
		{"agent", InstallTokenEntityTypeAgent},
		{"infrastructure", InstallTokenEntityTypeInfrastructure},
	}
	for _, tc := range entityCases {
		t.Run(tc.name+" deletes existing row and its bootstrap token", func(t *testing.T) {
			f := setupInstallTokenFixture(t, tc.entityType)
			bootstrapID := properties.UUID(uuid.New())
			f.installRepo.EXPECT().GetByEntity(mock.Anything, tc.entityType, f.entityID).
				Return(&InstallToken{EntityType: tc.entityType, EntityID: f.entityID, BootstrapTokenID: &bootstrapID}, nil).Once()
			f.tokenRepo.EXPECT().Delete(mock.Anything, bootstrapID).Return(nil).Once()
			f.installRepo.EXPECT().DeleteByEntity(mock.Anything, tc.entityType, f.entityID).Return(nil).Once()

			err := NewInstallTokenCommander(f.store).Revoke(f.ctx, tc.entityType, f.entityID)
			assert.NoError(t, err)
		})
	}

	t.Run("deletes row without bootstrap when BootstrapTokenID is nil", func(t *testing.T) {
		f := setupInstallTokenFixture(t, InstallTokenEntityTypeAgent)
		f.installRepo.EXPECT().GetByEntity(mock.Anything, InstallTokenEntityTypeAgent, f.entityID).
			Return(&InstallToken{EntityType: InstallTokenEntityTypeAgent, EntityID: f.entityID}, nil).Once()
		f.installRepo.EXPECT().DeleteByEntity(mock.Anything, InstallTokenEntityTypeAgent, f.entityID).Return(nil).Once()

		err := NewInstallTokenCommander(f.store).Revoke(f.ctx, InstallTokenEntityTypeAgent, f.entityID)
		assert.NoError(t, err)
	})

	t.Run("not found when nothing to revoke", func(t *testing.T) {
		f := setupInstallTokenFixture(t, InstallTokenEntityTypeAgent)
		f.installRepo.EXPECT().GetByEntity(mock.Anything, InstallTokenEntityTypeAgent, f.entityID).
			Return(nil, NotFoundError{}).Once()

		err := NewInstallTokenCommander(f.store).Revoke(f.ctx, InstallTokenEntityTypeAgent, f.entityID)
		assert.ErrorAs(t, err, &NotFoundError{})
	})
}
