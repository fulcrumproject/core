package authz

import (
	"context"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTokenCreationScope_Matches(t *testing.T) {
	provider1ID := properties.UUID(uuid.New())
	provider2ID := properties.UUID(uuid.New())
	agent1ID := properties.UUID(uuid.New())
	agent2ID := properties.UUID(uuid.New())

	// Setup mock agent querier
	mockAgentQuerier := &mockAgentRepo{
		agents: map[properties.UUID]*domain.Agent{
			agent1ID: {
				BaseEntity: domain.BaseEntity{ID: agent1ID},
				Name:       "Agent 1",
				ProviderID: provider1ID, // Agent 1 belongs to Provider 1
			},
			agent2ID: {
				BaseEntity: domain.BaseEntity{ID: agent2ID},
				Name:       "Agent 2",
				ProviderID: provider2ID, // Agent 2 belongs to Provider 2
			},
		},
	}

	tests := []struct {
		name     string
		scope    *TokenCreationScope
		identity *auth.Identity
		want     bool
	}{
		// Admin tests
		{
			name: "admin can create admin token",
			scope: NewTokenCreationScope(
				context.Background(),
				mockAgentQuerier,
				auth.RoleAdmin,
				nil,
			),
			identity: &auth.Identity{
				Role: auth.RoleAdmin,
				Scope: auth.IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			want: true,
		},
		{
			name: "admin can create participant token",
			scope: NewTokenCreationScope(
				context.Background(),
				mockAgentQuerier,
				auth.RoleParticipant,
				&provider1ID,
			),
			identity: &auth.Identity{
				Role: auth.RoleAdmin,
				Scope: auth.IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			want: true,
		},
		{
			name: "admin can create agent token for any agent",
			scope: NewTokenCreationScope(
				context.Background(),
				mockAgentQuerier,
				auth.RoleAgent,
				&agent2ID,
			),
			identity: &auth.Identity{
				Role: auth.RoleAdmin,
				Scope: auth.IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			want: true,
		},

		// Participant creating admin tokens (should fail)
		{
			name: "participant CANNOT create admin token",
			scope: NewTokenCreationScope(
				context.Background(),
				mockAgentQuerier,
				auth.RoleAdmin,
				nil,
			),
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID,
				},
			},
			want: false,
		},

		// Participant creating participant tokens
		{
			name: "participant can create token for itself",
			scope: NewTokenCreationScope(
				context.Background(),
				mockAgentQuerier,
				auth.RoleParticipant,
				&provider1ID,
			),
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID,
				},
			},
			want: true,
		},
		{
			name: "participant CANNOT create token for another participant",
			scope: NewTokenCreationScope(
				context.Background(),
				mockAgentQuerier,
				auth.RoleParticipant,
				&provider2ID,
			),
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID,
				},
			},
			want: false,
		},

		// Participant creating agent tokens
		{
			name: "participant can create token for its own agent",
			scope: NewTokenCreationScope(
				context.Background(),
				mockAgentQuerier,
				auth.RoleAgent,
				&agent1ID,
			),
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID,
				},
			},
			want: true,
		},
		{
			name: "participant CANNOT create token for another participant's agent",
			scope: NewTokenCreationScope(
				context.Background(),
				mockAgentQuerier,
				auth.RoleAgent,
				&agent2ID, // Agent 2 belongs to Provider 2
			),
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID, // Provider 1
				},
			},
			want: false,
		},
		{
			name: "participant creating agent token with invalid agent ID",
			scope: NewTokenCreationScope(
				context.Background(),
				mockAgentQuerier,
				auth.RoleAgent,
				func() *properties.UUID { id := properties.UUID(uuid.New()); return &id }(), // Non-existent agent
			),
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID,
				},
			},
			want: false,
		},

		// Edge cases
		{
			name: "nil identity",
			scope: NewTokenCreationScope(
				context.Background(),
				mockAgentQuerier,
				auth.RoleParticipant,
				&provider1ID,
			),
			identity: nil,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scope.Matches(tt.identity)
			assert.Equal(t, tt.want, got, "TokenCreationScope.Matches() result")
		})
	}
}

// Mock implementations for testing

type mockAgentRepo struct {
	agents map[properties.UUID]*domain.Agent
}

func (m *mockAgentRepo) Get(ctx context.Context, id properties.UUID) (*domain.Agent, error) {
	if agent, ok := m.agents[id]; ok {
		return agent, nil
	}
	return nil, domain.NewNotFoundErrorf("agent not found")
}

func (m *mockAgentRepo) CountByAgentType(context.Context, properties.UUID) (int64, error) {
	return 0, nil
}

// Implement other AgentRepository methods (not used in these tests)
func (m *mockAgentRepo) Create(context.Context, *domain.Agent) error                                    { return nil }
func (m *mockAgentRepo) Save(context.Context, *domain.Agent) error                                      { return nil }
func (m *mockAgentRepo) Delete(context.Context, properties.UUID) error                                  { return nil }
func (m *mockAgentRepo) Exists(context.Context, properties.UUID) (bool, error)                          { return false, nil }
func (m *mockAgentRepo) AuthScope(context.Context, properties.UUID) (auth.ObjectScope, error)           { return nil, nil }
func (m *mockAgentRepo) Count(context.Context) (int64, error)                                           { return 0, nil }
func (m *mockAgentRepo) List(context.Context, *auth.IdentityScope, *domain.PageReq) (*domain.PageRes[domain.Agent], error) {
	return nil, nil
}
func (m *mockAgentRepo) GetByProviderID(context.Context, properties.UUID) ([]*domain.Agent, error) {
	return nil, nil
}
func (m *mockAgentRepo) DeleteByProviderID(context.Context, properties.UUID) error { return nil }

func (m *mockAgentRepo) CountByProvider(context.Context, properties.UUID) (int64, error) {
	return 0, nil
}

func (m *mockAgentRepo) FindByServiceTypeAndTags(context.Context, properties.UUID, []string) ([]*domain.Agent, error) {
	return nil, nil
}

func (m *mockAgentRepo) MarkInactiveAgentsAsDisconnected(context.Context, time.Duration) (int64, error) {
	return 0, nil
}