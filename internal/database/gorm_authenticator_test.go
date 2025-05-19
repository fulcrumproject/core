package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/stretchr/testify/assert"
)

// MockTokenRepo is a simple mock for token repository
type MockTokenRepo struct {
	FindByHashedValueFunc func(ctx context.Context, hashedValue string) (*domain.Token, error)
}

func (m *MockTokenRepo) FindByHashedValue(ctx context.Context, hashedValue string) (*domain.Token, error) {
	return m.FindByHashedValueFunc(ctx, hashedValue)
}

// We don't need to implement these methods for our tests
func (m *MockTokenRepo) FindByID(ctx context.Context, id domain.UUID) (*domain.Token, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockTokenRepo) List(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Token], error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockTokenRepo) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockTokenRepo) Create(ctx context.Context, entity *domain.Token) error {
	return fmt.Errorf("not implemented")
}
func (m *MockTokenRepo) Save(ctx context.Context, entity *domain.Token) error {
	return fmt.Errorf("not implemented")
}
func (m *MockTokenRepo) Delete(ctx context.Context, id domain.UUID) error {
	return fmt.Errorf("not implemented")
}
func (m *MockTokenRepo) DeleteByProviderID(ctx context.Context, providerID domain.UUID) error {
	return fmt.Errorf("not implemented")
}
func (m *MockTokenRepo) DeleteByBrokerID(ctx context.Context, brokerID domain.UUID) error {
	return fmt.Errorf("not implemented")
}
func (m *MockTokenRepo) DeleteByAgentID(ctx context.Context, agentID domain.UUID) error {
	return fmt.Errorf("not implemented")
}

// MockStore is a simple mock for store
type MockStore struct {
	TokenRepoFunc func() domain.TokenRepository
}

func (m *MockStore) TokenRepo() domain.TokenRepository {
	return m.TokenRepoFunc()
}

// We don't need to implement these methods for our tests
func (m *MockStore) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	return fmt.Errorf("not implemented")
}
func (m *MockStore) AgentTypeRepo() domain.AgentTypeRepository {
	return nil
}
func (m *MockStore) AgentRepo() domain.AgentRepository {
	return nil
}
func (m *MockStore) BrokerRepo() domain.BrokerRepository {
	return nil
}
func (m *MockStore) ProviderRepo() domain.ProviderRepository {
	return nil
}
func (m *MockStore) ServiceTypeRepo() domain.ServiceTypeRepository {
	return nil
}
func (m *MockStore) ServiceGroupRepo() domain.ServiceGroupRepository {
	return nil
}
func (m *MockStore) ServiceRepo() domain.ServiceRepository {
	return nil
}
func (m *MockStore) JobRepo() domain.JobRepository {
	return nil
}
func (m *MockStore) AuditEntryRepo() domain.AuditEntryRepository {
	return nil
}
func (m *MockStore) MetricTypeRepo() domain.MetricTypeRepository {
	return nil
}
func (m *MockStore) MetricEntryRepo() domain.MetricEntryRepository {
	return nil
}

// TestGormTokenIdentity_ID tests the ID method
func TestGormTokenIdentity_ID(t *testing.T) {
	// Arrange
	id := domain.NewUUID()
	identity := GormTokenIdentity{
		id:         id,
		name:       "test-token",
		role:       domain.RoleFulcrumAdmin,
		providerID: nil,
		brokerID:   nil,
		agentID:    nil,
	}

	// Act
	result := identity.ID()

	// Assert
	assert.Equal(t, id, result)
}

// TestGormTokenIdentity_Name tests the Name method
func TestGormTokenIdentity_Name(t *testing.T) {
	// Arrange
	name := "test-token"
	identity := GormTokenIdentity{
		id:         domain.NewUUID(),
		name:       name,
		role:       domain.RoleFulcrumAdmin,
		providerID: nil,
		brokerID:   nil,
		agentID:    nil,
	}

	// Act
	result := identity.Name()

	// Assert
	assert.Equal(t, name, result)
}

// TestGormTokenIdentity_Role tests the Role method
func TestGormTokenIdentity_Role(t *testing.T) {
	// Arrange
	role := domain.RoleProviderAdmin
	identity := GormTokenIdentity{
		id:         domain.NewUUID(),
		name:       "test-token",
		role:       role,
		providerID: nil,
		brokerID:   nil,
		agentID:    nil,
	}

	// Act
	result := identity.Role()

	// Assert
	assert.Equal(t, role, result)
}

// TestGormTokenIdentity_Scope tests the Scope method
func TestGormTokenIdentity_Scope(t *testing.T) {
	// Arrange
	providerID := domain.NewUUID()
	brokerID := domain.NewUUID()
	agentID := domain.NewUUID()

	identity := GormTokenIdentity{
		id:         domain.NewUUID(),
		name:       "test-token",
		role:       domain.RoleAgent,
		providerID: &providerID,
		brokerID:   &brokerID,
		agentID:    &agentID,
	}

	// Act
	scope := identity.Scope()

	// Assert
	assert.NotNil(t, scope)
	assert.Equal(t, providerID, *scope.ParticipantID)
	assert.Equal(t, brokerID, *scope.BrokerID)
	assert.Equal(t, agentID, *scope.AgentID)
}

// TestGormTokenIdentity_IsRole tests the IsRole method
func TestGormTokenIdentity_IsRole(t *testing.T) {
	// Arrange
	role := domain.RoleBroker
	identity := GormTokenIdentity{
		id:         domain.NewUUID(),
		name:       "test-token",
		role:       role,
		providerID: nil,
		brokerID:   nil,
		agentID:    nil,
	}

	// Act & Assert
	assert.True(t, identity.IsRole(domain.RoleBroker))
	assert.False(t, identity.IsRole(domain.RoleAgent))
	assert.False(t, identity.IsRole(domain.RoleProviderAdmin))
	assert.False(t, identity.IsRole(domain.RoleFulcrumAdmin))
}

// TestNewTokenAuthenticator tests the NewTokenAuthenticator function
func TestNewTokenAuthenticator(t *testing.T) {
	// Arrange
	mockStore := &MockStore{}

	// Act
	authenticator := NewTokenAuthenticator(mockStore)

	// Assert
	assert.NotNil(t, authenticator)
	assert.Equal(t, mockStore, authenticator.store)
}

// TestGormTokenAuthenticator_Authenticate tests the Authenticate method
func TestGormTokenAuthenticator_Authenticate(t *testing.T) {
	// Setup test cases
	tests := []struct {
		name         string
		tokenValue   string
		setupMocks   func() (*MockStore, *MockTokenRepo)
		wantIdentity bool
	}{
		{
			name:       "Valid token",
			tokenValue: "valid-token",
			setupMocks: func() (*MockStore, *MockTokenRepo) {
				tokenRepo := &MockTokenRepo{}
				store := &MockStore{
					TokenRepoFunc: func() domain.TokenRepository {
						return tokenRepo
					},
				}

				tokenID := domain.NewUUID()
				providerID := domain.NewUUID()

				// Configure mock behavior
				tokenRepo.FindByHashedValueFunc = func(ctx context.Context, hashedValue string) (*domain.Token, error) {
					expectedHash := domain.HashTokenValue("valid-token")
					if hashedValue == expectedHash {
						return &domain.Token{
							BaseEntity:  domain.BaseEntity{ID: tokenID},
							Name:        "test-token",
							Role:        domain.RoleProviderAdmin,
							HashedValue: expectedHash,
							ProviderID:  &providerID,
							ExpireAt:    time.Now().Add(time.Hour), // Not expired
						}, nil
					}
					return nil, fmt.Errorf("unexpected hash value")
				}

				return store, tokenRepo
			},
			wantIdentity: true,
		},
		{
			name:       "Token not found",
			tokenValue: "invalid-token",
			setupMocks: func() (*MockStore, *MockTokenRepo) {
				tokenRepo := &MockTokenRepo{}
				store := &MockStore{
					TokenRepoFunc: func() domain.TokenRepository {
						return tokenRepo
					},
				}

				// Configure mock behavior
				tokenRepo.FindByHashedValueFunc = func(ctx context.Context, hashedValue string) (*domain.Token, error) {
					return nil, domain.NotFoundError{Err: fmt.Errorf("token not found")}
				}

				return store, tokenRepo
			},
			wantIdentity: false,
		},
		{
			name:       "Expired token",
			tokenValue: "expired-token",
			setupMocks: func() (*MockStore, *MockTokenRepo) {
				tokenRepo := &MockTokenRepo{}
				store := &MockStore{
					TokenRepoFunc: func() domain.TokenRepository {
						return tokenRepo
					},
				}

				tokenID := domain.NewUUID()

				// Configure mock behavior
				tokenRepo.FindByHashedValueFunc = func(ctx context.Context, hashedValue string) (*domain.Token, error) {
					expectedHash := domain.HashTokenValue("expired-token")
					if hashedValue == expectedHash {
						return &domain.Token{
							BaseEntity:  domain.BaseEntity{ID: tokenID},
							Name:        "expired-token",
							Role:        domain.RoleFulcrumAdmin,
							HashedValue: expectedHash,
							ExpireAt:    time.Now().Add(-time.Hour), // Expired 1 hour ago
						}, nil
					}
					return nil, fmt.Errorf("unexpected hash value")
				}

				return store, tokenRepo
			},
			wantIdentity: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			store, _ := tt.setupMocks()
			ctx := context.Background()

			// Create authenticator and run the test
			authenticator := NewTokenAuthenticator(store)
			identity := authenticator.Authenticate(ctx, tt.tokenValue)

			// Check the result
			if tt.wantIdentity {
				assert.NotNil(t, identity)
				assert.NotEmpty(t, identity.ID())
				assert.NotEmpty(t, identity.Name())
			} else {
				assert.Nil(t, identity)
			}
		})
	}
}
