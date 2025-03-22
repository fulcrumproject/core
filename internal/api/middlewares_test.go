package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// MockAuthenticator implements domain.Authenticator for testing
type MockAuthenticator struct {
	ValidToken       string
	IdentityToReturn domain.AuthIdentity
	ShouldReturnNil  bool
}

func (m *MockAuthenticator) Authenticate(ctx context.Context, token string) domain.AuthIdentity {
	if m.ShouldReturnNil || token != m.ValidToken {
		return nil
	}
	return m.IdentityToReturn
}

func TestExtractTokenFromRequest(t *testing.T) {
	tests := []struct {
		name          string
		headers       map[string]string
		expectedToken string
	}{
		{
			name: "Valid Bearer Token",
			headers: map[string]string{
				"Authorization": "Bearer token123",
			},
			expectedToken: "token123",
		},
		{
			name:          "Missing Authorization Header",
			headers:       map[string]string{},
			expectedToken: "",
		},
		{
			name: "Authorization Header Without Bearer Prefix",
			headers: map[string]string{
				"Authorization": "token123",
			},
			expectedToken: "",
		},
		{
			name: "Empty Token After Bearer Prefix",
			headers: map[string]string{
				"Authorization": "Bearer ",
			},
			expectedToken: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			token := extractTokenFromRequest(req)
			assert.Equal(t, tc.expectedToken, token)
		})
	}
}

// Simple test for AuthMiddleware that verifies the middleware behavior
func TestAuthMiddleware(t *testing.T) {
	// Define test cases
	tests := []struct {
		name           string
		token          string
		authenticator  *MockAuthenticator
		expectedStatus int
	}{
		{
			name:  "Valid Token",
			token: "valid-token",
			authenticator: &MockAuthenticator{
				ValidToken:       "valid-token",
				IdentityToReturn: NewMockAuthFulcrumAdmin(),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "No Token",
			token: "",
			authenticator: &MockAuthenticator{
				ValidToken:       "valid-token",
				IdentityToReturn: NewMockAuthFulcrumAdmin(),
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:  "Invalid Token",
			token: "invalid-token",
			authenticator: &MockAuthenticator{
				ValidToken:       "valid-token",
				IdentityToReturn: NewMockAuthFulcrumAdmin(),
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:  "Auth Fails",
			token: "valid-token",
			authenticator: &MockAuthenticator{
				ValidToken:      "valid-token",
				ShouldReturnNil: true,
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a simple test handler that always returns OK
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Always return OK for the test handler
				w.WriteHeader(http.StatusOK)
			})

			// Create middleware chain
			middleware := AuthMiddleware(tc.authenticator)
			handler := middleware(testHandler)

			// Create request with appropriate token
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			// Execute the request
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			// Verify status code
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestAuthMiddlewareContext tests that the middleware adds the identity to the context
func TestAuthMiddlewareContext(t *testing.T) {
	validToken := "valid-token"
	testIdentity := NewMockAuthFulcrumAdmin()

	// Create authenticator that returns identity for valid token
	auth := &MockAuthenticator{
		ValidToken:       validToken,
		IdentityToReturn: testIdentity,
	}

	// Create a test handler that checks for identity in context
	var identityFound bool
	var capturedIdentity domain.AuthIdentity

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use defer/recover to safely try to get identity
		defer func() {
			recover() // Just recover and continue
		}()

		// Try to access identity - if it exists, capture it
		capturedIdentity = domain.MustGetAuthIdentity(r.Context())
		identityFound = (capturedIdentity != nil)

		w.WriteHeader(http.StatusOK)
	})

	// Create middleware
	middleware := AuthMiddleware(auth)
	handler := middleware(testHandler)

	// Test with valid token
	t.Run("Valid token adds identity to context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+validToken)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, identityFound, "Identity should be found in context")
		assert.Equal(t, testIdentity.ID(), capturedIdentity.ID())
		assert.Equal(t, testIdentity.Role(), capturedIdentity.Role())
	})

	// Test with invalid token
	t.Run("Invalid token does not add identity", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestIDMiddleware(t *testing.T) {
	// Create test cases
	tests := []struct {
		name           string
		urlParam       string
		expectedStatus int
		shouldHaveID   bool
	}{
		{
			name:           "Valid UUID",
			urlParam:       "550e8400-e29b-41d4-a716-446655440000",
			expectedStatus: http.StatusOK,
			shouldHaveID:   true,
		},
		{
			name:           "Invalid UUID",
			urlParam:       "not-a-uuid",
			expectedStatus: http.StatusBadRequest,
			shouldHaveID:   false,
		},
		{
			name:           "Empty UUID",
			urlParam:       "",
			expectedStatus: http.StatusNotFound, // Chi returns 404 when URL param doesn't match
			shouldHaveID:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var capturedUUID domain.UUID
			var idInContext bool

			// Create a test handler that verifies the ID is in the context
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Try to extract UUID value from context
				id, ok := r.Context().Value(uuidContextKey).(domain.UUID)
				idInContext = ok
				if ok {
					capturedUUID = id
				}

				// Always return success from the handler itself
				w.WriteHeader(http.StatusOK)
			})

			// Create a router and use the middleware
			r := chi.NewRouter()
			// Only add ID middleware in the route with ID
			r.Route("/{id}", func(r chi.Router) {
				r.Use(IDMiddleware)
				r.Get("/", testHandler)
			})

			// Create the request
			var req *http.Request
			var err error
			if tc.urlParam != "" {
				req, err = http.NewRequest("GET", "/"+tc.urlParam+"/", nil)
			} else {
				req, err = http.NewRequest("GET", "/", nil)
			}
			assert.NoError(t, err)

			// Execute the request
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			// Verify the expected status
			assert.Equal(t, tc.expectedStatus, w.Code)

			// If we expected a successful extraction of a UUID
			if tc.shouldHaveID {
				assert.True(t, idInContext, "UUID should be in the context")
				assert.Equal(t, tc.urlParam, capturedUUID.String(), "UUID should match expected value")
			}
		})
	}
}

func TestMustGetID(t *testing.T) {
	// Test the happy path
	testUUID := uuid.New()
	testID := testUUID
	r := httptest.NewRequest("GET", "/test", nil)

	// Set the ID in the context using the same key as IDMiddleware
	r = r.WithContext(context.WithValue(r.Context(), uuidContextKey, testID))

	// Call MustGetID
	id := MustGetID(r)
	assert.Equal(t, testUUID.String(), id.String())

	// Test the panic case by creating a sub-test to capture the panic
	t.Run("Panic case", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustGetID did not panic when ID was missing from context")
			}
		}()

		// Create a request without an ID in the context
		r := httptest.NewRequest("GET", "/test", nil)
		// This should panic
		_ = MustGetID(r)
	})
}
