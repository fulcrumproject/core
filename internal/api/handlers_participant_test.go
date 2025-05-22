package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewParticipantHandler tests the constructor
func TestNewParticipantHandler(t *testing.T) {
	querier := &mockParticipantQuerier{}
	commander := &mockParticipantCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewParticipantHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestParticipantHandlerRoutes tests that routes are properly registered
func TestParticipantHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockParticipantQuerier{}
	commander := &mockParticipantCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewParticipantHandler(querier, commander, authz)

	// Execute
	routeFunc := handler.Routes()
	assert.NotNil(t, routeFunc)

	// Create a chi router and apply the routes
	r := chi.NewRouter()
	routeFunc(r)

	// Assert that endpoints are registered
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		// Check expected routes exist
		switch {
		case method == "GET" && route == "/":
		case method == "POST" && route == "/":
		case method == "GET" && route == "/{id}":
		case method == "PATCH" && route == "/{id}":
		case method == "DELETE" && route == "/{id}":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestParticipantHandleCreate tests the handleCreate method
func TestParticipantHandleCreate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		requestBody    string
		mockSetup      func(querier *mockParticipantQuerier, commander *mockParticipantCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name:        "Success",
			requestBody: `{"name": "Example Org", "state": "Enabled", "countryCode": "US", "attributes": {"region": ["us-east-1", "us-west-2"]}}`,
			mockSetup: func(querier *mockParticipantQuerier, commander *mockParticipantCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the commander
				commander.createFunc = func(ctx context.Context, name string, state domain.ParticipantState, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Participant, error) {
					assert.Equal(t, "Example Org", name)
					assert.Equal(t, domain.ParticipantState("Enabled"), state)
					assert.Equal(t, domain.CountryCode("US"), countryCode)
					assert.Equal(t, domain.Attributes{"region": {"us-east-1", "us-west-2"}}, attributes)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.Participant{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:        name,
						State:       state,
						CountryCode: countryCode,
						Attributes:  attributes,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:        "AuthorizationError",
			requestBody: `{"name": "Example Org", "state": "Enabled", "countryCode": "US", "attributes": {"region": ["us-east-1", "us-west-2"]}}`,
			mockSetup: func(querier *mockParticipantQuerier, commander *mockParticipantCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockParticipantQuerier{}
			commander := &mockParticipantCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewParticipantHandler(querier, commander, authz)

			// Create request with JSON body
			req := httptest.NewRequest("POST", "/participants", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleCreate(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusCreated {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", response["id"])
				assert.Equal(t, "Example Org", response["name"])
				assert.Equal(t, "Enabled", response["state"])
				assert.Equal(t, "US", response["countryCode"])
			}
		})
	}
}
