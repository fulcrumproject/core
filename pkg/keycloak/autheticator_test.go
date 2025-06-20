package keycloak

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticator_extractRole(t *testing.T) {
	config := &Config{
		ClientID: "test-client",
	}
	authenticator := &Authenticator{
		config: config,
	}

	tests := []struct {
		name         string
		claims       *Claims
		expectedRole auth.Role
		expectError  bool
	}{
		{
			name: "Direct role claim - admin",
			claims: &Claims{
				Role: "admin",
			},
			expectedRole: auth.RoleAdmin,
			expectError:  false,
		},
		{
			name: "Direct role claim - participant",
			claims: &Claims{
				Role: "participant",
			},
			expectedRole: auth.RoleParticipant,
			expectError:  false,
		},
		{
			name: "Direct role claim - agent",
			claims: &Claims{
				Role: "agent",
			},
			expectedRole: auth.RoleAgent,
			expectError:  false,
		},
		{
			name: "Realm role - admin",
			claims: &Claims{
				RealmAccess: struct {
					Roles []string `json:"roles"`
				}{
					Roles: []string{"participant", "admin", "user"},
				},
			},
			expectedRole: auth.RoleParticipant, // First valid role found
			expectError:  false,
		},
		{
			name: "Realm role - participant only",
			claims: &Claims{
				RealmAccess: struct {
					Roles []string `json:"roles"`
				}{
					Roles: []string{"participant"},
				},
			},
			expectedRole: auth.RoleParticipant,
			expectError:  false,
		},
		{
			name: "Client role",
			claims: &Claims{
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"test-client": {
						Roles: []string{"agent"},
					},
				},
			},
			expectedRole: auth.RoleAgent,
			expectError:  false,
		},
		{
			name: "Client role - multiple clients",
			claims: &Claims{
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"other-client": {
						Roles: []string{"admin"},
					},
					"test-client": {
						Roles: []string{"participant"},
					},
				},
			},
			expectedRole: auth.RoleParticipant,
			expectError:  false,
		},
		{
			name: "No valid role - invalid direct role",
			claims: &Claims{
				Role: "invalid-role",
			},
			expectError: true,
		},
		{
			name: "No valid role - invalid realm roles",
			claims: &Claims{
				RealmAccess: struct {
					Roles []string `json:"roles"`
				}{
					Roles: []string{"invalid", "unknown"},
				},
			},
			expectError: true,
		},
		{
			name: "No valid role - invalid client roles",
			claims: &Claims{
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"test-client": {
						Roles: []string{"invalid"},
					},
				},
			},
			expectError: true,
		},
		{
			name:        "Empty claims",
			claims:      &Claims{},
			expectError: true,
		},
		{
			name: "Role priority - direct role takes precedence",
			claims: &Claims{
				Role: "admin",
				RealmAccess: struct {
					Roles []string `json:"roles"`
				}{
					Roles: []string{"participant"},
				},
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"test-client": {
						Roles: []string{"agent"},
					},
				},
			},
			expectedRole: auth.RoleAdmin,
			expectError:  false,
		},
		{
			name: "Role priority - realm role over client role",
			claims: &Claims{
				RealmAccess: struct {
					Roles []string `json:"roles"`
				}{
					Roles: []string{"participant"},
				},
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"test-client": {
						Roles: []string{"agent"},
					},
				},
			},
			expectedRole: auth.RoleParticipant,
			expectError:  false,
		},
		{
			name: "Client role - wrong client ignored",
			claims: &Claims{
				ResourceAccess: map[string]struct {
					Roles []string `json:"roles"`
				}{
					"wrong-client": {
						Roles: []string{"admin"},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, err := authenticator.extractRole(tt.claims)

			if tt.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
				assert.Equal(t, tt.expectedRole, role, "Role should match expected value")
			}
		})
	}
}
