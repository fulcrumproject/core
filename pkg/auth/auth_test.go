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
