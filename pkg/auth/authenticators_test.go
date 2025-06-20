package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompositeAuthenticator_Authenticate(t *testing.T) {
	testUUID := properties.NewUUID()
	testUUID2 := properties.NewUUID()

	adminIdentity := &Identity{
		ID:   testUUID,
		Name: "admin-user",
		Role: RoleAdmin,
	}

	participantIdentity := &Identity{
		ID:   testUUID2,
		Name: "participant-user",
		Role: RoleParticipant,
		Scope: IdentityScope{
			ParticipantID: &testUUID2,
		},
	}

	authError := errors.New("authentication failed")

	tests := []struct {
		name              string
		authenticators    []*mockAuthenticator
		expectedIdentity  *Identity
		expectError       bool
		errorContains     string
		expectedCallCount []bool // which authenticators should be called
	}{
		{
			name: "First authenticator succeeds",
			authenticators: []*mockAuthenticator{
				{identity: adminIdentity, err: nil},
				{identity: nil, err: errors.New("should not be called")},
			},
			expectedIdentity:  adminIdentity,
			expectError:       false,
			expectedCallCount: []bool{true, false},
		},
		{
			name: "Second authenticator succeeds",
			authenticators: []*mockAuthenticator{
				{identity: nil, err: nil},
				{identity: participantIdentity, err: nil},
			},
			expectedIdentity:  participantIdentity,
			expectError:       false,
			expectedCallCount: []bool{true, true},
		},
		{
			name: "First authenticator fails with error",
			authenticators: []*mockAuthenticator{
				{identity: nil, err: authError},
				{identity: nil, err: errors.New("should not be called")},
			},
			expectedIdentity:  nil,
			expectError:       true,
			errorContains:     "authentication failed",
			expectedCallCount: []bool{true, false},
		},
		{
			name: "All authenticators return nil identity",
			authenticators: []*mockAuthenticator{
				{identity: nil, err: nil},
				{identity: nil, err: nil},
			},
			expectedIdentity:  nil,
			expectError:       true,
			errorContains:     "authentication failed: no valid identity found",
			expectedCallCount: []bool{true, true},
		},
		{
			name:              "No authenticators",
			authenticators:    []*mockAuthenticator{},
			expectedIdentity:  nil,
			expectError:       true,
			errorContains:     "authentication failed: no valid identity found",
			expectedCallCount: []bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to Authenticator interface slice
			auths := make([]Authenticator, len(tt.authenticators))
			for i, auth := range tt.authenticators {
				auths[i] = auth
			}

			composite := NewCompositeAuthenticator(auths...)
			ctx := context.Background()

			identity, err := composite.Authenticate(ctx, "test-token")

			if tt.expectError {
				require.Error(t, err, "Expected an error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected text")
				}
				assert.Nil(t, identity, "Should not return an identity")
			} else {
				assert.NoError(t, err, "Should not return an error")
				assert.Equal(t, tt.expectedIdentity, identity, "Should return expected identity")
			}

			// Verify which authenticators were called
			for i, expectedCalled := range tt.expectedCallCount {
				assert.Equal(t, expectedCalled, tt.authenticators[i].called,
					"Authenticator %d call status should match expected", i)
			}
		})
	}
}

// mockAuthenticator is a test helper that implements the Authenticator interface
type mockAuthenticator struct {
	identity      *Identity
	err           error
	called        bool
	receivedCtx   context.Context
	receivedToken string
}

func (m *mockAuthenticator) Authenticate(ctx context.Context, token string) (*Identity, error) {
	m.called = true
	m.receivedCtx = ctx
	m.receivedToken = token
	return m.identity, m.err
}
