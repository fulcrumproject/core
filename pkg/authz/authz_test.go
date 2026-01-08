package authz

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
)

func TestAllwaysMatchObjectScope_Matches(t *testing.T) {
	scope := AllwaysMatchObjectScope{}
	result := scope.Matches(&auth.Identity{})
	assert.Equal(t, true, result)
}

func TestDefaultObjectScope_Matches(t *testing.T) {
	// Helper to create test UUIDs
	participantID1 := properties.NewUUID()
	participantID2 := properties.NewUUID()
	agentID1 := properties.NewUUID()
	agentID2 := properties.NewUUID()
	providerID1 := properties.NewUUID()
	consumerID1 := properties.NewUUID()

	tests := []struct {
		name     string
		target   *DefaultObjectScope
		identity *auth.Identity
		expected bool
	}{
		{
			name:     "Nil identity should not match",
			target:   &DefaultObjectScope{},
			identity: nil,
			expected: false,
		},
		{
			name: "Admin access - unrestricted caller (both ParticipantID and AgentID nil)",
			target: &DefaultObjectScope{
				ParticipantID: &participantID1,
				ProviderID:    &providerID1,
				ConsumerID:    &consumerID1,
				AgentID:       &agentID1,
			},
			identity: &auth.Identity{
				Role: auth.RoleAdmin,
				Scope: auth.IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			expected: true,
		},
		{
			name: "Global access - all target fields nil",
			target: &DefaultObjectScope{
				ParticipantID: nil,
				ProviderID:    nil,
				ConsumerID:    nil,
				AgentID:       nil,
			},
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &participantID1,
					AgentID:       nil,
				},
			},
			expected: true,
		},
		{
			name: "Participant match - same participant ID",
			target: &DefaultObjectScope{
				ParticipantID: &participantID1,
				ProviderID:    nil,
				ConsumerID:    nil,
				AgentID:       nil,
			},
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &participantID1,
					AgentID:       nil,
				},
			},
			expected: true,
		},
		{
			name: "Provider match - participant matches provider ID",
			target: &DefaultObjectScope{
				ParticipantID: nil,
				ProviderID:    &participantID1,
				ConsumerID:    nil,
				AgentID:       nil,
			},
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &participantID1,
					AgentID:       nil,
				},
			},
			expected: true,
		},
		{
			name: "Consumer match - participant matches consumer ID",
			target: &DefaultObjectScope{
				ParticipantID: nil,
				ProviderID:    nil,
				ConsumerID:    &participantID1,
				AgentID:       nil,
			},
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &participantID1,
					AgentID:       nil,
				},
			},
			expected: true,
		},
		{
			name: "Agent match - same agent ID",
			target: &DefaultObjectScope{
				ParticipantID: nil,
				ProviderID:    nil,
				ConsumerID:    nil,
				AgentID:       &agentID1,
			},
			identity: &auth.Identity{
				Role: auth.RoleAgent,
				Scope: auth.IdentityScope{
					ParticipantID: &participantID1,
					AgentID:       &agentID1,
				},
			},
			expected: true,
		},
		{
			name: "No match - different participant IDs",
			target: &DefaultObjectScope{
				ParticipantID: &participantID1,
				ProviderID:    nil,
				ConsumerID:    nil,
				AgentID:       nil,
			},
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &participantID2,
					AgentID:       nil,
				},
			},
			expected: false,
		},
		{
			name: "No match - different agent IDs",
			target: &DefaultObjectScope{
				ParticipantID: nil,
				ProviderID:    nil,
				ConsumerID:    nil,
				AgentID:       &agentID1,
			},
			identity: &auth.Identity{
				Role: auth.RoleAgent,
				Scope: auth.IdentityScope{
					ParticipantID: &participantID1,
					AgentID:       &agentID2,
				},
			},
			expected: false,
		},
		{
			name: "No match - agent ID in target but nil in identity",
			target: &DefaultObjectScope{
				ParticipantID: nil,
				ProviderID:    nil,
				ConsumerID:    nil,
				AgentID:       &agentID1,
			},
			identity: &auth.Identity{
				Role: auth.RoleParticipant,
				Scope: auth.IdentityScope{
					ParticipantID: &participantID1,
					AgentID:       nil,
				},
			},
			expected: false,
		},
		{
			name: "No match - participant required but identity has nil participant",
			target: &DefaultObjectScope{
				ParticipantID: &participantID1,
				ProviderID:    nil,
				ConsumerID:    nil,
				AgentID:       nil,
			},
			identity: &auth.Identity{
				Role: auth.RoleAdmin,
				Scope: auth.IdentityScope{
					ParticipantID: nil,
					AgentID:       &agentID1,
				},
			},
			expected: false, // Should not match because identity has AgentID (not unrestricted) and nil ParticipantID
		},
		{
			name: "Edge case - identity with only agent ID, target with participant",
			target: &DefaultObjectScope{
				ParticipantID: &participantID1,
				ProviderID:    nil,
				ConsumerID:    nil,
				AgentID:       nil,
			},
			identity: &auth.Identity{
				Role: auth.RoleAgent,
				Scope: auth.IdentityScope{
					ParticipantID: nil,
					AgentID:       &agentID1,
				},
			},
			expected: false,
		},
		{
			name: "Complex match - participant matches and agent matches",
			target: &DefaultObjectScope{
				ParticipantID: &participantID1,
				ProviderID:    nil,
				ConsumerID:    nil,
				AgentID:       &agentID1,
			},
			identity: &auth.Identity{
				Role: auth.RoleAgent,
				Scope: auth.IdentityScope{
					ParticipantID: &participantID1,
					AgentID:       &agentID1,
				},
			},
			expected: true,
		},
		{
			name: "Complex no match - participant matches but agent doesn't",
			target: &DefaultObjectScope{
				ParticipantID: &participantID1,
				ProviderID:    nil,
				ConsumerID:    nil,
				AgentID:       &agentID1,
			},
			identity: &auth.Identity{
				Role: auth.RoleAgent,
				Scope: auth.IdentityScope{
					ParticipantID: &participantID1,
					AgentID:       &agentID2,
				},
			},
			expected: true, // Should match because participant matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.target.Matches(tt.identity)
			assert.Equal(t, tt.expected, result)
		})
	}
}
