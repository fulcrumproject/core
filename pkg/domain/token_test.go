package domain

import (
	"testing"
	"time"

	"context"
	"strings"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestToken_TableName(t *testing.T) {
	token := Token{}
	assert.Equal(t, "tokens", token.TableName())
}

func TestToken_Validate(t *testing.T) {
	validID := uuid.New()
	now := time.Now().Add(24 * time.Hour) // 1 day in the future

	tests := []struct {
		name       string
		token      *Token
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid fulcrum admin token",
			token: &Token{
				Name:        "Admin Token",
				Role:        auth.RoleAdmin,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
			},
			wantErr: false,
		},
		{
			name: "Valid participant token",
			token: &Token{
				Name:          "Provider Admin Token",
				Role:          auth.RoleParticipant,
				HashedValue:   "hashedvalue",
				ExpireAt:      now,
				ParticipantID: &validID,
			},
			wantErr: false,
		},
		{
			name: "Valid agent token",
			token: &Token{
				Name:          "Agent Token",
				Role:          auth.RoleAgent,
				HashedValue:   "hashedvalue",
				ExpireAt:      now,
				AgentID:       &validID,
				ParticipantID: &validID,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			token: &Token{
				Name:        "",
				Role:        auth.RoleAdmin,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
			},
			wantErr:    true,
			errMessage: "token name cannot be empty",
		},
		{
			name: "Empty hashed value",
			token: &Token{
				Name:        "Admin Token",
				Role:        auth.RoleAdmin,
				HashedValue: "",
				ExpireAt:    now,
			},
			wantErr:    true,
			errMessage: "token hashed value cannot be empty",
		},
		{
			name: "Invalid role",
			token: &Token{
				Name:        "Admin Token",
				Role:        "invalid-role",
				HashedValue: "hashedvalue",
				ExpireAt:    now,
			},
			wantErr:    true,
			errMessage: "invalid auth role",
		},
		{
			name: "Zero expire time",
			token: &Token{
				Name:        "Admin Token",
				Role:        auth.RoleAdmin,
				HashedValue: "hashedvalue",
				ExpireAt:    time.Time{},
			},
			wantErr:    true,
			errMessage: "token expire at cannot be empty",
		},
		{
			name: "Fulcrum admin with participant ID",
			token: &Token{
				Name:          "Admin Token",
				Role:          auth.RoleAdmin,
				HashedValue:   "hashedvalue",
				ExpireAt:      now,
				ParticipantID: &validID,
			},
			wantErr:    true,
			errMessage: "fulcrum admin tokens should not have any scope IDs",
		},
		{
			name: "Provider admin without participant ID",
			token: &Token{
				Name:        "Provider Admin Token",
				Role:        auth.RoleParticipant,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
			},
			wantErr:    true,
			errMessage: "participant ID is required for participant role",
		},
		{
			name: "Agent without agent ID",
			token: &Token{
				Name:          "Agent Token",
				Role:          auth.RoleAgent,
				HashedValue:   "hashedvalue",
				ExpireAt:      now,
				ParticipantID: &validID,
			},
			wantErr:    true,
			errMessage: "agent ID is required for agent role",
		},
		{
			name: "Agent without participant ID",
			token: &Token{
				Name:        "Agent Token",
				Role:        auth.RoleAgent,
				HashedValue: "hashedvalue",
				ExpireAt:    now,
				AgentID:     &validID,
			},
			wantErr:    true,
			errMessage: "participant ID is required for agent role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.token.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToken_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		expireAt time.Time
		want     bool
	}{
		{
			name:     "Not expired",
			expireAt: time.Now().Add(24 * time.Hour), // 1 day in the future
			want:     false,
		},
		{
			name:     "Expired",
			expireAt: time.Now().Add(-24 * time.Hour), // 1 day in the past
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{
				ExpireAt: tt.expireAt,
			}
			got := token.IsExpired()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToken_GenerateTokenValue(t *testing.T) {
	token := &Token{}
	err := token.GenerateTokenValue()
	assert.NoError(t, err)
	assert.NotEmpty(t, token.PlainValue)
	assert.NotEmpty(t, token.HashedValue)
	assert.NotEqual(t, token.PlainValue, token.HashedValue)
}

func TestToken_VerifyTokenValue(t *testing.T) {
	token := &Token{}
	err := token.GenerateTokenValue()
	assert.NoError(t, err)

	validValue := token.PlainValue
	invalidValue := "invalid-token-value"

	assert.True(t, token.VerifyTokenValue(validValue))
	assert.False(t, token.VerifyTokenValue(invalidValue))
}

func TestHashTokenValue(t *testing.T) {
	value1 := "token1"
	value2 := "token2"

	hash1 := HashTokenValue(value1)
	hash2 := HashTokenValue(value2)
	hash1Again := HashTokenValue(value1)

	assert.NotEmpty(t, hash1)
	assert.NotEmpty(t, hash2)
	assert.NotEqual(t, hash1, hash2)
	assert.Equal(t, hash1, hash1Again)
}


func TestTokenCommander_Create_Authorization(t *testing.T) {
	provider1ID := properties.UUID(uuid.New())
	provider2ID := properties.UUID(uuid.New())
	agent1ID := properties.UUID(uuid.New())
	agent2ID := properties.UUID(uuid.New())

	// Setup mock store
	var ms *mockTokenStore
	ms = &mockTokenStore{
		participantRepo: &mockParticipantRepository{
			existsFunc: func(ctx context.Context, id properties.UUID) (bool, error) {
				// Both participants exist
				return id == provider1ID || id == provider2ID, nil
			},
		},
		agentRepo: &mockAgentRepository{
			getFunc: func(ctx context.Context, id properties.UUID) (*Agent, error) {
				if id == agent1ID {
					return &Agent{
						BaseEntity: BaseEntity{ID: agent1ID},
						Name:       "Agent 1",
						ProviderID: provider1ID, // Agent 1 belongs to Provider 1
					}, nil
				}
				if id == agent2ID {
					return &Agent{
						BaseEntity: BaseEntity{ID: agent2ID},
						Name:       "Agent 2",
						ProviderID: provider2ID, // Agent 2 belongs to Provider 2
					}, nil
				}
				return nil, NewNotFoundErrorf("agent not found")
			},
		},
		tokenRepo: &mockTokenRepository{
			createFunc: func(ctx context.Context, token *Token) error {
				return nil
			},
		},
		eventRepo: &mockEventRepository{
			createFunc: func(ctx context.Context, event *Event) error {
				return nil
			},
		},
		atomicFunc: func(ctx context.Context, fn func(Store) error) error {
			return fn(ms)
		},
	}

	commander := NewTokenCommander(ms)
	expireAt := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name        string
		identity    *auth.Identity
		params      CreateTokenParams
		wantErr     bool
		errContains string
	}{
		{
			name: "admin can create admin token",
			identity: &auth.Identity{
				Role: auth.RoleAdmin,
				ID:   properties.UUID(uuid.New()),
				Name: "Admin User",
				Scope: auth.IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			params: CreateTokenParams{
				Name:     "Admin Token",
				Role:     auth.RoleAdmin,
				ExpireAt: &expireAt,
			},
			wantErr: false,
		},
		{
			name: "admin can create token for any participant",
			identity: &auth.Identity{
				Role: auth.RoleAdmin,
				ID:   properties.UUID(uuid.New()),
				Name: "Admin User",
			},
			params: CreateTokenParams{
				Name:     "Participant Token",
				Role:     auth.RoleParticipant,
				ScopeID:  &provider1ID,
				ExpireAt: &expireAt,
			},
			wantErr: false,
		},
		{
			name: "admin can create token for any agent",
			identity: &auth.Identity{
				Role: auth.RoleAdmin,
				ID:   properties.UUID(uuid.New()),
				Name: "Admin User",
			},
			params: CreateTokenParams{
				Name:     "Agent Token",
				Role:     auth.RoleAgent,
				ScopeID:  &agent2ID,
				ExpireAt: &expireAt,
			},
			wantErr: false,
		},
		{
			name: "participant CANNOT create admin token",
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				ID:   properties.UUID(uuid.New()),
				Name: "Participant User",
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID,
				},
			},
			params: CreateTokenParams{
				Name:     "Admin Token Attempt",
				Role:     auth.RoleAdmin,
				ExpireAt: &expireAt,
			},
			wantErr:     true,
			errContains: "role admin not allowed",
		},
		{
			name: "participant can create token for itself",
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				ID:   properties.UUID(uuid.New()),
				Name: "Participant User",
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID,
				},
			},
			params: CreateTokenParams{
				Name:     "Own Participant Token",
				Role:     auth.RoleParticipant,
				ScopeID:  &provider1ID,
				ExpireAt: &expireAt,
			},
			wantErr: false,
		},
		{
			name: "participant CANNOT create token for another participant",
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				ID:   properties.UUID(uuid.New()),
				Name: "Participant 1 User",
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID,
				},
			},
			params: CreateTokenParams{
				Name:     "Other Participant Token",
				Role:     auth.RoleParticipant,
				ScopeID:  &provider2ID, // Trying to create for Provider 2
				ExpireAt: &expireAt,
			},
			wantErr:     true,
			errContains: "cannot create token for another participant",
		},
		{
			name: "participant can create token for its own agent",
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				ID:   properties.UUID(uuid.New()),
				Name: "Participant 1 User",
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID,
				},
			},
			params: CreateTokenParams{
				Name:     "Own Agent Token",
				Role:     auth.RoleAgent,
				ScopeID:  &agent1ID, // Agent 1 belongs to Provider 1
				ExpireAt: &expireAt,
			},
			wantErr: false,
		},
		{
			name: "participant CANNOT create token for another participant's agent",
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				ID:   properties.UUID(uuid.New()),
				Name: "Participant 1 User",
				Scope: auth.IdentityScope{
					ParticipantID: &provider1ID,
				},
			},
			params: CreateTokenParams{
				Name:     "Other Agent Token",
				Role:     auth.RoleAgent,
				ScopeID:  &agent2ID, // Agent 2 belongs to Provider 2
				ExpireAt: &expireAt,
			},
			wantErr:     true,
			errContains: "cannot create token for agent belonging to another participant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := auth.WithIdentity(context.Background(), tt.identity)
			token, err := commander.Create(ctx, tt.params)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && (err == nil || !strings.Contains(err.Error(), tt.errContains)) {
					t.Errorf("Create() error should contain '%s', got: %v", tt.errContains, err)
				}
			} else {
				if token == nil {
					t.Error("Expected token to be created")
					return
				}
				if token.Role != tt.params.Role {
					t.Errorf("Expected role %s, got %s", tt.params.Role, token.Role)
				}
			}
		})
	}
}

// Mock store implementation for token tests
type mockTokenStore struct {
	participantRepo ParticipantRepository
	agentRepo       AgentRepository
	tokenRepo       TokenRepository
	eventRepo       EventRepository
	atomicFunc      func(context.Context, func(Store) error) error
}

func (m *mockTokenStore) ParticipantRepo() ParticipantRepository             { return m.participantRepo }
func (m *mockTokenStore) AgentRepo() AgentRepository                         { return m.agentRepo }
func (m *mockTokenStore) TokenRepo() TokenRepository                         { return m.tokenRepo }
func (m *mockTokenStore) EventRepo() EventRepository                         { return m.eventRepo }
func (m *mockTokenStore) AgentTypeRepo() AgentTypeRepository                 { return nil }
func (m *mockTokenStore) ServiceTypeRepo() ServiceTypeRepository             { return nil }
func (m *mockTokenStore) ServiceRepo() ServiceRepository                     { return nil }
func (m *mockTokenStore) ServiceGroupRepo() ServiceGroupRepository           { return nil }
func (m *mockTokenStore) ServiceOptionTypeRepo() ServiceOptionTypeRepository { return nil }
func (m *mockTokenStore) ServiceOptionRepo() ServiceOptionRepository         { return nil }
func (m *mockTokenStore) ServicePoolSetRepo() ServicePoolSetRepository       { return nil }
func (m *mockTokenStore) ServicePoolRepo() ServicePoolRepository             { return nil }
func (m *mockTokenStore) ServicePoolValueRepo() ServicePoolValueRepository   { return nil }
func (m *mockTokenStore) JobRepo() JobRepository                             { return nil }
func (m *mockTokenStore) MetricTypeRepo() MetricTypeRepository               { return nil }
func (m *mockTokenStore) MetricEntryRepo() MetricEntryRepository             { return nil }
func (m *mockTokenStore) EventSubscriptionRepo() EventSubscriptionRepository { return nil }
func (m *mockTokenStore) Atomic(ctx context.Context, fn func(Store) error) error {
	if m.atomicFunc != nil {
		return m.atomicFunc(ctx, fn)
	}
	return fn(m)
}

type mockTokenRepository struct {
	createFunc func(context.Context, *Token) error
}

func (m *mockTokenRepository) Create(ctx context.Context, token *Token) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, token)
	}
	return nil
}
func (m *mockTokenRepository) Get(context.Context, properties.UUID) (*Token, error) { return nil, nil }
func (m *mockTokenRepository) Save(context.Context, *Token) error                   { return nil }
func (m *mockTokenRepository) Delete(context.Context, properties.UUID) error        { return nil }
func (m *mockTokenRepository) Exists(context.Context, properties.UUID) (bool, error) {
	return false, nil
}
func (m *mockTokenRepository) AuthScope(context.Context, properties.UUID) (auth.ObjectScope, error) {
	return nil, nil
}
func (m *mockTokenRepository) Count(context.Context) (int64, error) { return 0, nil }
func (m *mockTokenRepository) List(context.Context, *auth.IdentityScope, *PageReq) (*PageRes[Token], error) {
	return nil, nil
}
func (m *mockTokenRepository) FindByHashedValue(context.Context, string) (*Token, error) {
	return nil, nil
}
func (m *mockTokenRepository) DeleteByParticipantID(context.Context, properties.UUID) error {
	return nil
}
func (m *mockTokenRepository) DeleteByAgentID(context.Context, properties.UUID) error {
	return nil
}