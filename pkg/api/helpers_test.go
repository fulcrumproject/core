package api

import (
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
)

// Test helper functions for creating mock auth identities

func newMockAuthAdmin() *auth.Identity {
	return &auth.Identity{
		ID:   uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		Name: "test-admin",
		Role: auth.RoleAdmin,
	}
}

func newMockAuthAgent() *auth.Identity {
	agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
	participantID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")

	return &auth.Identity{
		ID:   uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		Name: "test-agent",
		Role: auth.RoleAgent,
		Scope: auth.IdentityScope{
			ParticipantID: &participantID,
			AgentID:       &agentID,
		},
	}
}

func newMockAuthAgentWithID(agentID properties.UUID) *auth.Identity {
	participantID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")

	return &auth.Identity{
		ID:   agentID,
		Name: "test-agent",
		Role: auth.RoleAgent,
		Scope: auth.IdentityScope{
			ParticipantID: &participantID,
			AgentID:       &agentID,
		},
	}
}

