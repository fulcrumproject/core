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

// TestBrokerHandleCreate tests the handleCreate method
func TestBrokerHandleCreate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		requestBody    string
		mockSetup      func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			requestBody: `{
				"name": "TestBroker"
			}`,
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return a test broker
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				commander.createFunc = func(ctx context.Context, name string) (*domain.Broker, error) {
					assert.Equal(t, "TestBroker", name)

					return &domain.Broker{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name: name,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"id":        "550e8400-e29b-41d4-a716-446655440000",
				"name":      "TestBroker",
				"createdAt": "2023-01-01T00:00:00Z",
				"updatedAt": "2023-01-01T00:00:00Z",
			},
		},
		{
			name:        "InvalidRequest",
			requestBody: `{invalid json`,
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Create mock for the commander even though it won't reach it
				commander.createFunc = func(ctx context.Context, name string) (*domain.Broker, error) {
					return &domain.Broker{
						BaseEntity: domain.BaseEntity{
							ID: uuid.New(),
						},
						Name: name,
					}, nil
				}
				// Authorization should not be called for invalid requests
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "UnauthorizedCreate",
			requestBody: `{
				"name": "TestBroker"
			}`,
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden, // The handler returns ErrDomain for auth failures
		},
		{
			name: "CommanderError",
			requestBody: `{
				"name": "TestBroker"
			}`,
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return an error
				commander.createFunc = func(ctx context.Context, name string) (*domain.Broker, error) {
					return nil, domain.NewInvalidInputErrorf("broker creation failed")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockBrokerQuerier{}
			commander := &mockBrokerCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewBrokerHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("POST", "/brokers", strings.NewReader(tc.requestBody))
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
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

// TestBrokerHandleGet tests the handleGet method
func TestBrokerHandleGet(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return a test broker
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Broker, error) {
					return &domain.Broker{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name: "TestBroker",
					}, nil
				}

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"id":        "550e8400-e29b-41d4-a716-446655440000",
				"name":      "TestBroker",
				"createdAt": "2023-01-01T00:00:00Z",
				"updatedAt": "2023-01-01T00:00:00Z",
			},
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Broker, error) {
					return nil, domain.NewNotFoundErrorf("broker not found")
				}

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Unauthorized",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "AuthScopeError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return nil, domain.NewNotFoundErrorf("scope not found")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockBrokerQuerier{}
			commander := &mockBrokerCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewBrokerHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/brokers/"+tc.id, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)

			// We need to add the UUID to the context directly since we're not using the middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

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
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

// TestBrokerHandleList tests the handleList method
func TestBrokerHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return test brokers
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Broker], error) {
					return &domain.PageResponse[domain.Broker]{
						Items: []domain.Broker{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name: "TestBroker1",
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name: "TestBroker2",
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
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "InvalidPageRequest",
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true
			},
			expectedStatus: http.StatusOK, // parsePageRequest doesn't return errors for invalid page params, it uses defaults
		},
		{
			name: "ListError",
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Broker], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockBrokerQuerier{}
			commander := &mockBrokerCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewBrokerHandler(querier, commander, authz)

			// Create request
			var req *http.Request
			if tc.name == "InvalidPageRequest" {
				// Create an invalid page request
				req = httptest.NewRequest("GET", "/brokers?page=-1&pageSize=invalid", nil)
			} else {
				req = httptest.NewRequest("GET", "/brokers?page=1&pageSize=10", nil)
			}

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleList(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK && tc.name == "Success" {
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
				assert.Equal(t, "TestBroker1", firstItem["name"])

				secondItem := items[1].(map[string]interface{})
				assert.Equal(t, "660e8400-e29b-41d4-a716-446655440000", secondItem["id"])
				assert.Equal(t, "TestBroker2", secondItem["name"])
			}
		})
	}
}

// TestBrokerHandleUpdate tests the handleUpdate method
func TestBrokerHandleUpdate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		requestBody    string
		mockSetup      func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"name": "UpdatedBroker"
			}`,
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return a test broker
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string) (*domain.Broker, error) {
					updatedName := "UpdatedBroker"
					assert.Equal(t, &updatedName, name)

					return &domain.Broker{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name: updatedName,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"id":        "550e8400-e29b-41d4-a716-446655440000",
				"name":      "UpdatedBroker",
				"createdAt": "2023-01-01T00:00:00Z",
				"updatedAt": "2023-01-02T00:00:00Z",
			},
		},
		{
			name: "InvalidRequest",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"invalid": json
			`,
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Auth should not be called for invalid requests
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"name": "UpdatedBroker"
			}`,
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "UpdateError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"name": "UpdatedBroker"
			}`,
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string) (*domain.Broker, error) {
					return nil, domain.NewInvalidInputErrorf("validation error")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockBrokerQuerier{}
			commander := &mockBrokerCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewBrokerHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("PATCH", "/brokers/"+tc.id, strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)

			// We need to add the UUID to the context directly since we're not using the middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

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
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

// TestBrokerHandleDelete tests the handleDelete method
func TestBrokerHandleDelete(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				commander.deleteFunc = func(ctx context.Context, id domain.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "Unauthorized",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "DeleteError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockBrokerQuerier, commander *mockBrokerCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

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
			querier := &mockBrokerQuerier{}
			commander := &mockBrokerCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewBrokerHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("DELETE", "/brokers/"+tc.id, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)

			// We need to add the UUID to the context directly since we're not using the middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

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

// TestNewBrokerHandler tests the constructor
func TestNewBrokerHandler(t *testing.T) {
	querier := &mockBrokerQuerier{}
	commander := &mockBrokerCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewBrokerHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestBrokerHandlerRoutes tests that routes are properly registered
func TestBrokerHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockBrokerQuerier{}
	commander := &mockBrokerCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewBrokerHandler(querier, commander, authz)

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

// TestBrokerToResponse tests the brokerToResponse function
func TestBrokerToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	broker := &domain.Broker{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name: "TestBroker",
	}

	response := brokerToResponse(broker)

	assert.Equal(t, broker.ID, response.ID)
	assert.Equal(t, broker.Name, response.Name)
	assert.Equal(t, JSONUTCTime(broker.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(broker.UpdatedAt), response.UpdatedAt)
}

// TestBrokerAuthorize tests the authorize function
func TestBrokerAuthorize(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name        string
		mockSetup   func(querier *mockBrokerQuerier, authz *MockAuthorizer)
		expectError bool
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockBrokerQuerier, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectError: false,
		},
		{
			name: "AuthScopeError",
			mockSetup: func(querier *mockBrokerQuerier, authz *MockAuthorizer) {
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return nil, domain.NewNotFoundErrorf("scope not found")
				}
			},
			expectError: true,
		},
		{
			name: "AuthorizationError",
			mockSetup: func(querier *mockBrokerQuerier, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockBrokerQuerier{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, authz)

			// Create the handler
			handler := NewBrokerHandler(querier, nil, authz)

			// Execute authorize
			ctx := context.Background()
			id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
			action := domain.ActionRead

			scope, err := handler.authorize(ctx, id, action)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, scope)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scope)
			}
		})
	}
}
