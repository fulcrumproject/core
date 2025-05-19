package domain

import (
	"context"
)

// MockAuthIdentity implements AuthIdentity for testing
type MockAuthIdentity struct {
	id          UUID
	name        string
	role        AuthRole
	participant *UUID
	agent       *UUID
	scope       *AuthScope
}

// NewMockAuthIdentity creates a new mock identity with the specified ID and role
func NewMockAuthIdentity(id UUID, role AuthRole) *MockAuthIdentity {
	return &MockAuthIdentity{
		id:    id,
		name:  id.String(), // Use ID as name by default
		role:  role,
		scope: &AuthScope{}, // Initialize with empty scope
	}
}

func (m *MockAuthIdentity) ID() UUID {
	return m.id
}

func (m *MockAuthIdentity) Name() string {
	return m.name
}

func (m *MockAuthIdentity) WithName(name string) *MockAuthIdentity {
	m.name = name
	return m
}

func (m *MockAuthIdentity) Role() AuthRole {
	return m.role
}

func (m *MockAuthIdentity) IsRole(role AuthRole) bool {
	return m.role == role
}

func (m *MockAuthIdentity) Scope() *AuthScope {
	return m.scope
}

func (m *MockAuthIdentity) WithScope(scope *AuthScope) *MockAuthIdentity {
	m.scope = scope
	return m
}

func (m *MockAuthIdentity) ParticipantID() *UUID {
	return m.participant
}

func (m *MockAuthIdentity) WithParticipantID(id *UUID) *MockAuthIdentity {
	m.participant = id
	if m.scope == nil {
		m.scope = &AuthScope{}
	}
	m.scope.ParticipantID = id
	return m
}

func (m *MockAuthIdentity) AgentID() *UUID {
	return m.agent
}

func (m *MockAuthIdentity) WithAgentID(id *UUID) *MockAuthIdentity {
	m.agent = id
	if m.scope == nil {
		m.scope = &AuthScope{}
	}
	m.scope.AgentID = id
	return m
}

// ContextWithMockAuth adds a mock auth identity to the context
func ContextWithMockAuth(ctx context.Context, identity AuthIdentity) context.Context {
	return context.WithValue(ctx, identityContextKey, identity)
}
