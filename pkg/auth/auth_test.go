package auth

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRole_Validate(t *testing.T) {
	tests := []struct {
		name        string
		role        Role
		expectError bool
	}{
		{
			name:        "Valid admin role",
			role:        RoleAdmin,
			expectError: false,
		},
		{
			name:        "Valid participant role",
			role:        RoleParticipant,
			expectError: false,
		},
		{
			name:        "Valid agent role",
			role:        RoleAgent,
			expectError: false,
		},
		{
			name:        "Invalid role",
			role:        Role("invalid"),
			expectError: true,
		},
		{
			name:        "Empty role",
			role:        Role(""),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.role.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIdentity_HasRole(t *testing.T) {
	identity := &Identity{
		Role: RoleAdmin,
	}

	tests := []struct {
		name     string
		role     Role
		expected bool
	}{
		{
			name:     "Has admin role",
			role:     RoleAdmin,
			expected: true,
		},
		{
			name:     "Does not have participant role",
			role:     RoleParticipant,
			expected: false,
		},
		{
			name:     "Does not have agent role",
			role:     RoleAgent,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := identity.HasRole(tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIdentity_Validate(t *testing.T) {
	// Helper to create test UUIDs
	testUUID := properties.NewUUID()
	testUUID2 := properties.NewUUID()

	tests := []struct {
		name        string
		identity    *Identity
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid admin identity",
			identity: &Identity{
				Role: RoleAdmin,
				Scope: IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			expectError: false,
		},
		{
			name: "Valid participant identity",
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       nil,
				},
			},
			expectError: false,
		},
		{
			name: "Valid agent identity",
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       &testUUID2,
				},
			},
			expectError: false,
		},
		{
			name: "Invalid participant - missing participant ID",
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			expectError: true,
			errorMsg:    "participant role requires participant id",
		},
		{
			name: "Invalid agent - missing participant ID",
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
					ParticipantID: nil,
					AgentID:       &testUUID,
				},
			},
			expectError: true,
			errorMsg:    "agent role requires participant id",
		},
		{
			name: "Invalid agent - missing agent ID",
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       nil,
				},
			},
			expectError: true,
			errorMsg:    "agent role requires agent id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.identity.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAllwaysMatchObjectScope_Matches(t *testing.T) {
	scope := AllwaysMatchObjectScope{}
	result := scope.Matches(&Identity{})
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
		identity *Identity
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
			identity: &Identity{
				Role: RoleAdmin,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleAdmin,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
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
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
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
