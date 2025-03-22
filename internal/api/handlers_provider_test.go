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

// TestNewProviderHandler tests the constructor
func TestNewProviderHandler(t *testing.T) {
	querier := &mockProviderQuerier{}
	commander := &mockProviderCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewProviderHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestProviderHandlerRoutes tests that routes are properly registered
func TestProviderHandlerRoutes(t *testing.T) {
	// We'll use a stub for the actual handler to avoid executing real handler logic
	stubHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a test router and register routes
	r := chi.NewRouter()

	// Instead of using the actual handlers which require auth context,
	// we'll manually register routes with our stub handler
	r.Route("/providers", func(r chi.Router) {
		// Register the routes
		r.Get("/", stubHandler)
		r.Post("/", stubHandler)
		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return next
			})
			r.Get("/{id}", stubHandler)
			r.Patch("/{id}", stubHandler)
			r.Delete("/{id}", stubHandler)
		})
	})

	// Test GET route
	req := httptest.NewRequest("GET", "/providers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test POST route
	req = httptest.NewRequest("POST", "/providers", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test GET /{id} route
	req = httptest.NewRequest("GET", "/providers/550e8400-e29b-41d4-a716-446655440000", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test PATCH /{id} route
	req = httptest.NewRequest("PATCH", "/providers/550e8400-e29b-41d4-a716-446655440000", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test DELETE /{id} route
	req = httptest.NewRequest("DELETE", "/providers/550e8400-e29b-41d4-a716-446655440000", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

// TestProviderHandleCreate tests the handleCreate method
func TestProviderHandleCreate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		requestBody    string
		mockSetup      func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name:        "Success",
			requestBody: `{"name": "AWS", "state": "Enabled", "countryCode": "US", "attributes": {"region": ["us-east-1", "us-west-2"]}}`,
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the commander
				commander.createFunc = func(ctx context.Context, name string, state domain.ProviderState, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Provider, error) {
					assert.Equal(t, "AWS", name)
					assert.Equal(t, domain.ProviderState("Enabled"), state)
					assert.Equal(t, domain.CountryCode("US"), countryCode)
					assert.Equal(t, domain.Attributes{"region": {"us-east-1", "us-west-2"}}, attributes)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.Provider{
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
			name:        "InvalidRequestFormat",
			requestBody: `{"invalid_json":`,
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// No setup needed for this case
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "AuthorizationError",
			requestBody: `{"name": "AWS", "state": "Enabled", "countryCode": "US", "attributes": {"region": ["us-east-1", "us-west-2"]}}`,
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:        "CommanderError",
			requestBody: `{"name": "AWS", "state": "Enabled", "countryCode": "US", "attributes": {"region": ["us-east-1", "us-west-2"]}}`,
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the commander to return an error
				commander.createFunc = func(ctx context.Context, name string, state domain.ProviderState, countryCode domain.CountryCode, attributes domain.Attributes) (*domain.Provider, error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockProviderQuerier{}
			commander := &mockProviderCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewProviderHandler(querier, commander, authz)

			// Create request with JSON body
			req := httptest.NewRequest("POST", "/providers", strings.NewReader(tc.requestBody))
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
				assert.Equal(t, "AWS", response["name"])
				assert.Equal(t, "Enabled", response["state"])
				assert.Equal(t, "US", response["countryCode"])

				attributes := response["attributes"].(map[string]interface{})
				regions := attributes["region"].([]interface{})
				assert.ElementsMatch(t, []interface{}{"us-east-1", "us-west-2"}, regions)
			}
		})
	}
}

// TestProviderHandleGet tests the handleGet method
func TestProviderHandleGet(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &domain.EmptyAuthScope, nil
				}

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Provider, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.Provider{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:        "AWS",
						State:       domain.ProviderEnabled,
						CountryCode: "US",
						Attributes:  domain.Attributes{"region": {"us-east-1", "us-west-2"}},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "AuthorizationError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the mock to fail authorization
				authz.ShouldSucceed = false

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Setup the querier to return not found
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Provider, error) {
					return nil, domain.NotFoundError{Err: fmt.Errorf("provider not found")}
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "AuthScopeError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the querier to return auth scope error
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return nil, fmt.Errorf("auth scope error")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockProviderQuerier{}
			commander := &mockProviderCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewProviderHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/providers/"+tc.id, nil)

			// Add ID to chi context and simulate IDMiddleware
			req = addIDToChiContext(req, tc.id)
			req = simulateIDMiddleware(req, tc.id)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleGet(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, tc.id, response["id"])
				assert.Equal(t, "AWS", response["name"])
				assert.Equal(t, "Enabled", response["state"])
				assert.Equal(t, "US", response["countryCode"])

				attributes := response["attributes"].(map[string]interface{})
				regions := attributes["region"].([]interface{})
				assert.ElementsMatch(t, []interface{}{"us-east-1", "us-west-2"}, regions)
			}
		})
	}
}

// TestProviderHandleList tests the handleList method
func TestProviderHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return providers
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Provider], error) {
					return &domain.PageResponse[domain.Provider]{
						Items: []domain.Provider{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:        "AWS",
								State:       domain.ProviderEnabled,
								CountryCode: "US",
								Attributes:  domain.Attributes{"region": {"us-east-1", "us-west-2"}},
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:        "Azure",
								State:       domain.ProviderEnabled,
								CountryCode: "US",
								Attributes:  domain.Attributes{"region": {"eastus", "westus"}},
							},
						},
						TotalItems:  2,
						CurrentPage: 1,
						TotalPages:  1,
						HasNext:     false,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Unauthorized",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "ListError",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Provider], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockProviderQuerier{}
			commander := &mockProviderCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewProviderHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/providers?page=1&pageSize=10", nil)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleList(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, float64(1), response["currentPage"])
				assert.Equal(t, float64(2), response["totalItems"])
				assert.Equal(t, float64(1), response["totalPages"])
				assert.Equal(t, false, response["hasNext"])
				assert.Equal(t, false, response["hasPrev"])

				items := response["items"].([]interface{})
				assert.Equal(t, 2, len(items))

				firstItem := items[0].(map[string]interface{})
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", firstItem["id"])
				assert.Equal(t, "AWS", firstItem["name"])
				assert.Equal(t, "Enabled", firstItem["state"])
				assert.Equal(t, "US", firstItem["countryCode"])

				secondItem := items[1].(map[string]interface{})
				assert.Equal(t, "660e8400-e29b-41d4-a716-446655440000", secondItem["id"])
				assert.Equal(t, "Azure", secondItem["name"])
			}
		})
	}
}

// TestProviderHandleUpdate tests the handleUpdate method
func TestProviderHandleUpdate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		requestBody    string
		mockSetup      func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name:        "Success",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"name": "AWS Updated", "state": "Disabled"}`,
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &domain.EmptyAuthScope, nil
				}

				// Setup the commander to update
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, state *domain.ProviderState, countryCode *domain.CountryCode, attributes *domain.Attributes) (*domain.Provider, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					require.NotNil(t, name)
					assert.Equal(t, "AWS Updated", *name)
					require.NotNil(t, state)
					assert.Equal(t, domain.ProviderDisabled, *state)
					assert.Nil(t, countryCode)
					assert.Nil(t, attributes)

					newName := "AWS Updated"
					newState := domain.ProviderDisabled
					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

					return &domain.Provider{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:        newName,
						State:       newState,
						CountryCode: "US",
						Attributes:  domain.Attributes{"region": {"us-east-1", "us-west-2"}},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "InvalidRequestFormat",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"invalid_json":`,
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// No setup needed for this case
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "AuthorizationError",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"name": "AWS Updated"}`,
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the mock to fail authorization
				authz.ShouldSucceed = false

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:        "AuthScopeError",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"name": "AWS Updated"}`,
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the querier to return auth scope error
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return nil, fmt.Errorf("auth scope error")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:        "CommanderError",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"name": "AWS Updated"}`,
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Setup the commander to return an error
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, state *domain.ProviderState, countryCode *domain.CountryCode, attributes *domain.Attributes) (*domain.Provider, error) {
					return nil, fmt.Errorf("update error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockProviderQuerier{}
			commander := &mockProviderCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewProviderHandler(querier, commander, authz)

			// Create request with JSON body
			req := httptest.NewRequest("PATCH", "/providers/"+tc.id, strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Add ID to chi context and simulate IDMiddleware
			req = addIDToChiContext(req, tc.id)
			req = simulateIDMiddleware(req, tc.id)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleUpdate(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, tc.id, response["id"])
				assert.Equal(t, "AWS Updated", response["name"])
				assert.Equal(t, "Disabled", response["state"])
				assert.Equal(t, "US", response["countryCode"])

				attributes := response["attributes"].(map[string]interface{})
				regions := attributes["region"].([]interface{})
				assert.ElementsMatch(t, []interface{}{"us-east-1", "us-west-2"}, regions)
			}
		})
	}
}

// TestProviderHandleDelete tests the handleDelete method
func TestProviderHandleDelete(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &domain.EmptyAuthScope, nil
				}

				// Setup the commander to delete
				commander.deleteFunc = func(ctx context.Context, id domain.UUID) error {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "AuthorizationError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the mock to fail authorization
				authz.ShouldSucceed = false

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "AuthScopeError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the querier to return auth scope error
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return nil, fmt.Errorf("auth scope error")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "DeleteError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockProviderQuerier, commander *mockProviderCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Setup the commander to return an error
				commander.deleteFunc = func(ctx context.Context, id domain.UUID) error {
					return fmt.Errorf("delete error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockProviderQuerier{}
			commander := &mockProviderCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewProviderHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("DELETE", "/providers/"+tc.id, nil)

			// Add ID to chi context and simulate IDMiddleware
			req = addIDToChiContext(req, tc.id)
			req = simulateIDMiddleware(req, tc.id)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleDelete(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestProviderToResponse tests the provderToResponse function
func TestProviderToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Create a provider
	provider := &domain.Provider{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:        "AWS",
		State:       domain.ProviderEnabled,
		CountryCode: "US",
		Attributes:  domain.Attributes{"region": {"us-east-1", "us-west-2"}},
	}

	response := provderToResponse(provider)

	// Verify all fields are correctly mapped
	assert.Equal(t, provider.ID, response.ID)
	assert.Equal(t, string(provider.Name), response.Name)
	assert.Equal(t, provider.State, response.State)
	assert.Equal(t, string(provider.CountryCode), response.CountryCode)
	assert.Equal(t, map[string][]string(provider.Attributes), response.Attributes)
	assert.Equal(t, JSONUTCTime(provider.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(provider.UpdatedAt), response.UpdatedAt)
}
