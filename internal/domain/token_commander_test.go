package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTokenCommander_Create(t *testing.T) {
	validID := uuid.New()
	validName := "Test Token"

	tests := []struct {
		name       string
		ctx        context.Context
		role       AuthRole
		scopeID    *UUID
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Create fulcrum admin token",
			ctx:     ContextWithMockAuth(context.Background(), NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)),
			role:    RoleFulcrumAdmin,
			scopeID: nil,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock token Create
				tokenRepo.createFunc = func(ctx context.Context, token *Token) error {
					token.ID = validID
					assert.Equal(t, validName, token.Name)
					assert.Equal(t, RoleFulcrumAdmin, token.Role)
					assert.NotEmpty(t, token.HashedValue)
					assert.True(t, token.ExpireAt.After(time.Now()))
					assert.Nil(t, token.ParticipantID)
					assert.Nil(t, token.AgentID)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeTokenCreated, eventType)
					assert.NotNil(t, properties)
					assert.Equal(t, &validID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name:    "Create participant token (formerly provider admin)",
			ctx:     ContextWithMockAuth(context.Background(), NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)),
			role:    RoleParticipant,
			scopeID: &validID,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				participantRepo := &MockParticipantRepository{}
				store.WithParticipantRepo(participantRepo)

				// Mock participant exists
				participantRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					assert.Equal(t, validID, id)
					return true, nil
				}

				// Mock token Create
				tokenRepo.createFunc = func(ctx context.Context, token *Token) error {
					token.ID = validID
					assert.Equal(t, validName, token.Name)
					assert.Equal(t, RoleParticipant, token.Role)
					assert.NotEmpty(t, token.HashedValue)
					assert.Equal(t, &validID, token.ParticipantID)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeTokenCreated, eventType)
					// providerID in audit might become participantID, or be nil if entityID covers it.
					// For now, let's assume entityID is sufficient for participant tokens.
					// The audit log structure might need further review based on unification.
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name:    "Create participant token (formerly broker)",
			ctx:     ContextWithMockAuth(context.Background(), NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)),
			role:    RoleParticipant, // Updated role
			scopeID: &validID,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				participantRepo := &MockParticipantRepository{} // Updated repository
				store.WithParticipantRepo(participantRepo)      // Updated method

				// Mock participant exists
				participantRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					assert.Equal(t, validID, id)
					return true, nil
				}

				// Mock token Create
				tokenRepo.createFunc = func(ctx context.Context, token *Token) error {
					token.ID = validID
					assert.Equal(t, validName, token.Name)
					assert.Equal(t, RoleParticipant, token.Role) // Updated role
					assert.NotEmpty(t, token.HashedValue)
					assert.Equal(t, &validID, token.ParticipantID) // Updated field
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeTokenCreated, eventType)
					// Similar to provider token, audit details might change.
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name:    "Create agent token",
			ctx:     ContextWithMockAuth(context.Background(), NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)),
			role:    RoleAgent,
			scopeID: &validID,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				// Mock agent FindByID
				agentProviderID := uuid.New()
				agent := &Agent{
					BaseEntity: BaseEntity{
						ID: validID,
					},
					ProviderID: agentProviderID, // Changed from ProviderID
				}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					assert.Equal(t, validID, id)
					return agent, nil
				}

				// Mock token Create
				tokenRepo.createFunc = func(ctx context.Context, token *Token) error {
					token.ID = validID
					assert.Equal(t, validName, token.Name)
					assert.Equal(t, RoleAgent, token.Role)
					assert.NotEmpty(t, token.HashedValue)
					assert.Equal(t, &validID, token.AgentID)
					assert.Equal(t, &agentProviderID, token.ParticipantID) // Changed from ProviderID
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeTokenCreated, eventType)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name:    "Non-admin creating different role token",
			ctx:     ContextWithMockAuth(context.Background(), NewMockAuthIdentity(uuid.New(), RoleParticipant)), // Changed from RoleProviderAdmin
			role:    RoleAgent,
			scopeID: &validID,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				// No need to set up mocks as we expect early failure
			},
			wantErr:    true,
			errMessage: "role agent not allowed",
		},
		{
			name:    "Participant ID not found (formerly Provider ID not found)",
			ctx:     ContextWithMockAuth(context.Background(), NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)),
			role:    RoleParticipant,
			scopeID: &validID,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				store.WithParticipantRepo(participantRepo)

				// Mock participant exists as false
				participantRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return false, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "invalid participant ID",
		},
		{
			name:    "Participant ID not found (formerly Broker ID not found)",
			ctx:     ContextWithMockAuth(context.Background(), NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)),
			role:    RoleParticipant, // Updated role
			scopeID: &validID,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{} // Updated repository
				store.WithParticipantRepo(participantRepo)      // Updated method

				// Mock participant exists as false
				participantRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return false, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "invalid participant ID", // Updated message
		},
		{
			name:    "Agent ID not found",
			ctx:     ContextWithMockAuth(context.Background(), NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)),
			role:    RoleAgent,
			scopeID: &validID,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				// Mock agent FindByID with error
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return nil, errors.New("agent not found")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "invalid agent ID",
		},
		{
			name:    "Token creation error",
			ctx:     ContextWithMockAuth(context.Background(), NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)),
			role:    RoleFulcrumAdmin,
			scopeID: nil,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock token Create with error
				tokenRepo.createFunc = func(ctx context.Context, token *Token) error {
					return errors.New("database error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "database error",
		},
		{
			name:    "Audit entry error",
			ctx:     ContextWithMockAuth(context.Background(), NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)),
			role:    RoleFulcrumAdmin,
			scopeID: nil,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock token Create
				tokenRepo.createFunc = func(ctx context.Context, token *Token) error {
					token.ID = validID
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					return nil, errors.New("audit entry error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "audit entry error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			now := time.Now().Add(24 * time.Hour) // 1 day in the future

			tt.setupMocks(store, audit)
			commander := NewTokenCommander(store, audit)
			token, err := commander.Create(tt.ctx, validName, tt.role, now, tt.scopeID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, validID, token.ID)
				assert.Equal(t, validName, token.Name)
				assert.Equal(t, tt.role, token.Role)
				assert.NotEmpty(t, token.PlainValue) // Plain value should be available after creation
				assert.NotEmpty(t, token.HashedValue)
			}
		})
	}
}

func TestTokenCommander_Update(t *testing.T) {
	ctx := context.Background()
	tokenID := uuid.New()
	existingName := "Existing Token"
	newName := "Updated Token"
	existingExpiry := time.Now().Add(24 * time.Hour)
	newExpiry := time.Now().Add(48 * time.Hour)

	tests := []struct {
		name           string
		updateName     *string
		updateExpireAt *time.Time
		setupMocks     func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr        bool
		errMessage     string
	}{
		{
			name:           "Update both fields",
			updateName:     &newName,
			updateExpireAt: &newExpiry,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Existing token
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        existingName,
					Role:        RoleFulcrumAdmin,
					HashedValue: "hashedvalue",
					ExpireAt:    existingExpiry,
				}

				// Mock FindByID
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					assert.Equal(t, tokenID, id)
					return existingToken, nil
				}

				// Mock Save
				tokenRepo.updateFunc = func(ctx context.Context, token *Token) error {
					assert.Equal(t, tokenID, token.ID)
					assert.Equal(t, newName, token.Name)
					assert.Equal(t, newExpiry, token.ExpireAt)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeTokenUpdated, eventType)

					// Verify before object
					beforeToken, ok := before.(*Token)
					assert.True(t, ok)
					assert.Equal(t, existingName, beforeToken.Name)
					assert.Equal(t, existingExpiry, beforeToken.ExpireAt)

					// Verify after object
					afterToken, ok := after.(*Token)
					assert.True(t, ok)
					assert.Equal(t, newName, afterToken.Name)
					assert.Equal(t, newExpiry, afterToken.ExpireAt)

					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name:           "Update only name",
			updateName:     &newName,
			updateExpireAt: nil,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Existing token
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        existingName,
					Role:        RoleFulcrumAdmin,
					HashedValue: "hashedvalue",
					ExpireAt:    existingExpiry,
				}

				// Mock FindByID
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return existingToken, nil
				}

				// Mock Save
				tokenRepo.updateFunc = func(ctx context.Context, token *Token) error {
					assert.Equal(t, newName, token.Name)
					assert.Equal(t, existingExpiry, token.ExpireAt) // unchanged
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name:           "Update only expiry",
			updateName:     nil,
			updateExpireAt: &newExpiry,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Existing token
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        existingName,
					Role:        RoleFulcrumAdmin,
					HashedValue: "hashedvalue",
					ExpireAt:    existingExpiry,
				}

				// Mock FindByID
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return existingToken, nil
				}

				// Mock Save
				tokenRepo.updateFunc = func(ctx context.Context, token *Token) error {
					assert.Equal(t, existingName, token.Name) // unchanged
					assert.Equal(t, newExpiry, token.ExpireAt)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name:       "Token not found",
			updateName: &newName,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID to return not found error
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return nil, NewNotFoundErrorf("token not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name:       "Validation error - empty name",
			updateName: stringPtr(""),
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Existing token
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        existingName,
					Role:        RoleFulcrumAdmin,
					HashedValue: "hashedvalue",
					ExpireAt:    existingExpiry,
				}

				// Mock FindByID
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return existingToken, nil
				}
			},
			wantErr:    true,
			errMessage: "token name cannot be empty",
		},
		{
			name:       "Save error",
			updateName: &newName,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Existing token
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        existingName,
					Role:        RoleFulcrumAdmin,
					HashedValue: "hashedvalue",
					ExpireAt:    existingExpiry,
				}

				// Mock FindByID
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return existingToken, nil
				}

				// Mock Save with error
				tokenRepo.updateFunc = func(ctx context.Context, token *Token) error {
					return errors.New("database error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "database error",
		},
		{
			name:       "Audit entry error",
			updateName: &newName,
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Existing token
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        existingName,
					Role:        RoleFulcrumAdmin,
					HashedValue: "hashedvalue",
					ExpireAt:    existingExpiry,
				}

				// Mock FindByID
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return existingToken, nil
				}

				// Mock Save
				tokenRepo.updateFunc = func(ctx context.Context, token *Token) error {
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					return nil, errors.New("audit entry error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "audit entry error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewTokenCommander(store, audit)
			token, err := commander.Update(ctx, tokenID, tt.updateName, tt.updateExpireAt)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)

				// Check that the fields were updated
				if tt.updateName != nil {
					assert.Equal(t, *tt.updateName, token.Name)
				}
				if tt.updateExpireAt != nil {
					assert.Equal(t, *tt.updateExpireAt, token.ExpireAt)
				}
			}
		})
	}
}

func TestTokenCommander_Delete(t *testing.T) {
	ctx := context.Background()
	tokenID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Delete success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        "Test Token",
					Role:        RoleFulcrumAdmin,
					HashedValue: "hashedvalue",
					ExpireAt:    time.Now().Add(24 * time.Hour),
				}
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					assert.Equal(t, tokenID, id)
					return existingToken, nil
				}

				// Mock Delete
				tokenRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					assert.Equal(t, tokenID, id)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeTokenDeleted, eventType)
					assert.NotNil(t, properties)
					assert.Equal(t, &tokenID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name: "Token not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID to return not found error
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return nil, NewNotFoundErrorf("token not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Delete error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        "Test Token",
					Role:        RoleFulcrumAdmin,
					HashedValue: "hashedvalue",
					ExpireAt:    time.Now().Add(24 * time.Hour),
				}
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return existingToken, nil
				}

				// Mock Delete with error
				tokenRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					return errors.New("delete error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "delete error",
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        "Test Token",
					Role:        RoleFulcrumAdmin,
					HashedValue: "hashedvalue",
					ExpireAt:    time.Now().Add(24 * time.Hour),
				}
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return existingToken, nil
				}

				// Mock Delete
				tokenRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					return nil, errors.New("audit entry error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "audit entry error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewTokenCommander(store, audit)
			err := commander.Delete(ctx, tokenID)

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

func TestTokenCommander_Regenerate(t *testing.T) {
	ctx := context.Background()
	tokenID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Regenerate success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        "Test Token",
					Role:        RoleFulcrumAdmin,
					HashedValue: "oldhash",
					ExpireAt:    time.Now().Add(24 * time.Hour),
				}
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					assert.Equal(t, tokenID, id)
					return existingToken, nil
				}

				// Mock Save
				tokenRepo.updateFunc = func(ctx context.Context, token *Token) error {
					assert.Equal(t, tokenID, token.ID)
					assert.NotEqual(t, "oldhash", token.HashedValue)
					assert.NotEmpty(t, token.PlainValue)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeTokenRegenerated, eventType)
					assert.NotNil(t, properties)
					assert.Equal(t, &tokenID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name: "Token not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID to return not found error
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return nil, NewNotFoundErrorf("token not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        "Test Token",
					Role:        RoleFulcrumAdmin,
					HashedValue: "oldhash",
					ExpireAt:    time.Now().Add(24 * time.Hour),
				}
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return existingToken, nil
				}

				// Mock Save with error
				tokenRepo.updateFunc = func(ctx context.Context, token *Token) error {
					return errors.New("database error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "database error",
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID
				existingToken := &Token{
					BaseEntity: BaseEntity{
						ID: tokenID,
					},
					Name:        "Test Token",
					Role:        RoleFulcrumAdmin,
					HashedValue: "oldhash",
					ExpireAt:    time.Now().Add(24 * time.Hour),
				}
				tokenRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Token, error) {
					return existingToken, nil
				}

				// Mock Save
				tokenRepo.updateFunc = func(ctx context.Context, token *Token) error {
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					return nil, errors.New("audit entry error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "audit entry error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewTokenCommander(store, audit)
			token, err := commander.Regenerate(ctx, tokenID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.NotEmpty(t, token.PlainValue)
				assert.NotEmpty(t, token.HashedValue)
			}
		})
	}
}
