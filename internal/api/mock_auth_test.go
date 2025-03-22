package api

import (
	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
)

func NewMockAuthFulcrumAdmin() *MockAuthIdentity {
	return &MockAuthIdentity{
		id:   uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		role: domain.RoleFulcrumAdmin,
	}
}

func NewMockAuthProviderAdmin() *MockAuthIdentity {
	providerID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")
	return &MockAuthIdentity{
		id:         uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		role:       domain.RoleProviderAdmin,
		providerID: &providerID,
	}
}

func NewMockAuthAgent() *MockAuthIdentity {
	agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
	providerID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")

	return &MockAuthIdentity{
		id:         uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		role:       domain.RoleAgent,
		providerID: &providerID,
		agentID:    &agentID,
	}
}

func NewMockAuthBroker() *MockAuthIdentity {
	bokerID := uuid.MustParse("091c2e30-0706-11f0-a319-460683de5083")
	return &MockAuthIdentity{
		id:       uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		role:     domain.RoleBroker,
		brokerID: &bokerID,
	}
}

// MockAdminIdentity implements the domain.AuthIdentity interface for testing
type MockAuthIdentity struct {
	id         domain.UUID
	role       domain.AuthRole
	agentID    *domain.UUID
	providerID *domain.UUID
	brokerID   *domain.UUID
}

func (m MockAuthIdentity) ID() domain.UUID                  { return m.id }
func (m MockAuthIdentity) Name() string                     { return "test-admin" }
func (m MockAuthIdentity) Role() domain.AuthRole            { return m.role }
func (m MockAuthIdentity) IsRole(role domain.AuthRole) bool { return role == m.role }
func (m MockAuthIdentity) Scope() *domain.AuthScope {
	return &domain.AuthScope{
		AgentID:    m.agentID,
		ProviderID: m.providerID,
		BrokerID:   m.brokerID,
	}
}
