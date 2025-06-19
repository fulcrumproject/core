package api

import (
	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"github.com/google/uuid"
)

// Updated constructor functions for the participant unification
func NewMockAuthParticipant() *auth.Identity {
	participantID := uuid.MustParse("1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d")
	return &auth.Identity{
		ID:   uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		Name: "test-participant",
		Role: auth.RoleParticipant,
		Scope: auth.IdentityScope{
			ParticipantID: &participantID,
		},
	}
}

func NewMockAuthAdmin() *auth.Identity {
	return &auth.Identity{
		ID:   uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		Name: "test-admin",
		Role: auth.RoleAdmin,
	}
}

func NewMockAuthAgent() *auth.Identity {
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

// NewMockAuthAgentWithID creates a mock agent identity with a specific agent ID
func NewMockAuthAgentWithID(agentID properties.UUID) *auth.Identity {
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

// Legacy function kept for compatibility with existing tests
// but updated to use RoleParticipant instead of RoleConsumer
func NewMockAuthConsumer() *auth.Identity {
	participantID := uuid.MustParse("091c2e30-0706-11f0-a319-460683de5083")
	return &auth.Identity{
		ID:   uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		Name: "test-consumer",
		Role: auth.RoleParticipant,
		Scope: auth.IdentityScope{
			ParticipantID: &participantID,
		},
	}
}
