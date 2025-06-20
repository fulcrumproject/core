package auth

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleBasedAuthorizer_Authorize(t *testing.T) {
	// Setup test data
	testUUID := properties.NewUUID()
	testUUID2 := properties.NewUUID()

	rules := []AuthorizationRule{
		{Roles: []Role{RoleAdmin}, Action: "read", Object: "user"},
		{Roles: []Role{RoleAdmin}, Action: "write", Object: "user"},
		{Roles: []Role{RoleParticipant}, Action: "read", Object: "data"},
		{Roles: []Role{RoleAgent}, Action: "write", Object: "agent_data"},
		{Roles: []Role{RoleAdmin, RoleParticipant}, Action: "read", Object: "shared_data"},
		{Roles: []Role{RoleParticipant, RoleAgent}, Action: "write", Object: "participant_data"},
	}

	authorizer := NewRuleBasedAuthorizer(rules)

	tests := []struct {
		name          string
		identity      *Identity
		action        Action
		object        ObjectType
		objectContext ObjectScope
		expectError   bool
		errorContains string
	}{
		{
			name: "Admin can read user",
			identity: &Identity{
				Role: RoleAdmin,
				Scope: IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			action:        "read",
			object:        "user",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   false,
		},
		{
			name: "Admin can write user",
			identity: &Identity{
				Role: RoleAdmin,
				Scope: IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			action:        "write",
			object:        "user",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   false,
		},
		{
			name: "Participant can read data",
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       nil,
				},
			},
			action:        "read",
			object:        "data",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   false,
		},
		{
			name: "Agent can write agent_data",
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       &testUUID2,
				},
			},
			action:        "write",
			object:        "agent_data",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   false,
		},
		{
			name: "Participant cannot write user",
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       nil,
				},
			},
			action:        "write",
			object:        "user",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   true,
			errorContains: "access denied: no matching authorization rule found",
		},
		{
			name: "Agent cannot read user",
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       &testUUID2,
				},
			},
			action:        "read",
			object:        "user",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   true,
			errorContains: "access denied: no matching authorization rule found",
		},
		{
			name: "Unknown action denied",
			identity: &Identity{
				Role: RoleAdmin,
				Scope: IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			action:        "delete",
			object:        "user",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   true,
			errorContains: "access denied: no matching authorization rule found",
		},
		{
			name: "Unknown object denied",
			identity: &Identity{
				Role: RoleAdmin,
				Scope: IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			action:        "read",
			object:        "unknown",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   true,
			errorContains: "access denied: no matching authorization rule found",
		},
		{
			name: "Admin can read shared_data (multiple roles rule)",
			identity: &Identity{
				Role: RoleAdmin,
				Scope: IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			action:        "read",
			object:        "shared_data",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   false,
		},
		{
			name: "Participant can read shared_data (multiple roles rule)",
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       nil,
				},
			},
			action:        "read",
			object:        "shared_data",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   false,
		},
		{
			name: "Agent cannot read shared_data (not in multiple roles rule)",
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       &testUUID2,
				},
			},
			action:        "read",
			object:        "shared_data",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   true,
			errorContains: "access denied: no matching authorization rule found",
		},
		{
			name: "Participant can write participant_data (multiple roles rule)",
			identity: &Identity{
				Role: RoleParticipant,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       nil,
				},
			},
			action:        "write",
			object:        "participant_data",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   false,
		},
		{
			name: "Agent can write participant_data (multiple roles rule)",
			identity: &Identity{
				Role: RoleAgent,
				Scope: IdentityScope{
					ParticipantID: &testUUID,
					AgentID:       &testUUID2,
				},
			},
			action:        "write",
			object:        "participant_data",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   false,
		},
		{
			name: "Admin cannot write participant_data (not in multiple roles rule)",
			identity: &Identity{
				Role: RoleAdmin,
				Scope: IdentityScope{
					ParticipantID: nil,
					AgentID:       nil,
				},
			},
			action:        "write",
			object:        "participant_data",
			objectContext: AllwaysMatchObjectScope{},
			expectError:   true,
			errorContains: "access denied: no matching authorization rule found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authorizer.Authorize(tt.identity, tt.action, tt.object, tt.objectContext)

			if tt.expectError {
				require.Error(t, err, "Expected an error")
				assert.Contains(t, err.Error(), tt.errorContains, "Error message should contain expected text")
			} else {
				assert.NoError(t, err, "Expected no error")
			}
		})
	}
}

func TestRuleBasedAuthorizer_Authorize_ObjectContextMismatch(t *testing.T) {
	rules := []AuthorizationRule{
		{Roles: []Role{RoleAdmin}, Action: "read", Object: "user"},
	}

	authorizer := NewRuleBasedAuthorizer(rules)

	// Create a mock ObjectScope that never matches
	mockObjectScope := &mockObjectScope{shouldMatch: false}

	identity := &Identity{
		Role: RoleAdmin,
		Scope: IdentityScope{
			ParticipantID: nil,
			AgentID:       nil,
		},
	}

	err := authorizer.Authorize(identity, "read", "user", mockObjectScope)

	require.Error(t, err, "Expected an error")
	assert.Contains(t, err.Error(), "access denied: object context does not match identity", "Error should indicate object context mismatch")
}

func TestRuleBasedAuthorizer_Authorize_NilObjectContext(t *testing.T) {
	rules := []AuthorizationRule{
		{Roles: []Role{RoleAdmin}, Action: "read", Object: "user"},
	}

	authorizer := NewRuleBasedAuthorizer(rules)

	identity := &Identity{
		Role: RoleAdmin,
		Scope: IdentityScope{
			ParticipantID: nil,
			AgentID:       nil,
		},
	}

	err := authorizer.Authorize(identity, "read", "user", nil)

	assert.NoError(t, err, "Should succeed when object context is nil")
}

// mockObjectScope is a test helper that implements ObjectScope
type mockObjectScope struct {
	shouldMatch bool
}

func (m *mockObjectScope) Matches(identity *Identity) bool {
	return m.shouldMatch
}
