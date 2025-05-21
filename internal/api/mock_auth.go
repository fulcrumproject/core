package api

import (
	"fulcrumproject.org/core/internal/domain"
	"github.com/google/uuid"
)

// Updated constructor functions for the participant unification
func NewMockAuthParticipant() *MockAuthIdentity {
	participantID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")
	return &MockAuthIdentity{
		id:            uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		role:          domain.RoleParticipant,
		participantID: &participantID,
	}
}

func NewMockAuthFulcrumAdmin() *MockAuthIdentity {
	return &MockAuthIdentity{
		id:   uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		role: domain.RoleFulcrumAdmin,
	}
}

func NewMockAuthAgent() *MockAuthIdentity {
	agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
	participantID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")

	return &MockAuthIdentity{
		id:            uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		role:          domain.RoleAgent,
		participantID: &participantID,
		agentID:       &agentID,
	}
}

// Legacy function kept for compatibility with existing tests
// but updated to use RoleParticipant instead of RoleConsumer
func NewMockAuthConsumer() *MockAuthIdentity {
	participantID := uuid.MustParse("091c2e30-0706-11f0-a319-460683de5083")
	return &MockAuthIdentity{
		id:            uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		role:          domain.RoleParticipant,
		participantID: &participantID,
	}
}

// MockAuthIdentity implements the domain.AuthIdentity interface for testing
type MockAuthIdentity struct {
	id            domain.UUID
	role          domain.AuthRole
	agentID       *domain.UUID
	participantID *domain.UUID // Replaces providerID and consumerID
}

func (m MockAuthIdentity) ID() domain.UUID                  { return m.id }
func (m MockAuthIdentity) Name() string                     { return "test-admin" }
func (m MockAuthIdentity) Role() domain.AuthRole            { return m.role }
func (m MockAuthIdentity) IsRole(role domain.AuthRole) bool { return role == m.role }
func (m MockAuthIdentity) Scope() *domain.AuthIdentityScope {
	return &domain.AuthIdentityScope{
		AgentID:       m.agentID,
		ParticipantID: m.participantID,
	}
}
