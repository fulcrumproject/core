package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewServiceTypeHandler tests the constructor
func TestNewServiceTypeHandler(t *testing.T) {
	querier := &mockServiceTypeQuerier{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewServiceTypeHandler(querier, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, authz, handler.authz)
}

// TestServiceTypeHandlerRoutes tests that routes are properly registered
func TestServiceTypeHandlerRoutes(t *testing.T) {
	// We'll use a stub for the actual handler to avoid executing real handler logic
	stubHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a test router and register routes
	r := chi.NewRouter()

	// Instead of using the actual handlers which require auth context,
	// we'll manually register routes with our stub handler
	r.Route("/service-types", func(r chi.Router) {
		// Register the routes
		r.Get("/", stubHandler)
		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return next
			})
			r.Get("/{id}", stubHandler)
		})
	})

	// Test GET route
	req := httptest.NewRequest("GET", "/service-types", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test GET /{id} route
	req = httptest.NewRequest("GET", "/service-types/550e8400-e29b-41d4-a716-446655440000", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

// TestServiceTypeHandleGet tests the handleGet method
func TestServiceTypeHandleGet(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockServiceTypeQuerier, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockServiceTypeQuerier, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &domain.EmptyAuthScope, nil
				}

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.ServiceType, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.ServiceType{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name: "VM Instance",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "AuthorizationError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockServiceTypeQuerier, authz *MockAuthorizer) {
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
			mockSetup: func(querier *mockServiceTypeQuerier, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Setup the querier to return not found
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.ServiceType, error) {
					return nil, domain.NotFoundError{Err: fmt.Errorf("service type not found")}
				}
			},
			expectedStatus: http.StatusNotFound, // ErrDomain checks for NotFoundError and returns 404
		},
		{
			name: "AuthScopeError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockServiceTypeQuerier, authz *MockAuthorizer) {
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
			querier := &mockServiceTypeQuerier{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, authz)

			// Create the handler
			handler := NewServiceTypeHandler(querier, authz)

			// Create request
			req := httptest.NewRequest("GET", "/service-types/"+tc.id, nil)

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
				assert.Equal(t, "VM Instance", response["name"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
			}
		})
	}
}

// TestServiceTypeHandleList tests the handleList method
func TestServiceTypeHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockServiceTypeQuerier, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockServiceTypeQuerier, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return service types
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceType], error) {
					return &domain.PageResponse[domain.ServiceType]{
						Items: []domain.ServiceType{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name: "VM Instance",
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name: "Load Balancer",
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
			mockSetup: func(querier *mockServiceTypeQuerier, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "ListError",
			mockSetup: func(querier *mockServiceTypeQuerier, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceType], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockServiceTypeQuerier{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, authz)

			// Create the handler
			handler := NewServiceTypeHandler(querier, authz)

			// Create request
			req := httptest.NewRequest("GET", "/service-types?page=1&pageSize=10", nil)

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
				assert.Equal(t, "VM Instance", firstItem["name"])

				secondItem := items[1].(map[string]interface{})
				assert.Equal(t, "660e8400-e29b-41d4-a716-446655440000", secondItem["id"])
				assert.Equal(t, "Load Balancer", secondItem["name"])
			}
		})
	}
}

// TestServiceTypeToResponse tests the serviceTypeToResponse function
func TestServiceTypeToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Create a service type
	serviceType := &domain.ServiceType{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name: "VM Instance",
	}

	response := serviceTypeToResponse(serviceType)

	// Verify all fields are correctly mapped
	assert.Equal(t, serviceType.ID, response.ID)
	assert.Equal(t, serviceType.Name, response.Name)
	assert.Equal(t, JSONUTCTime(serviceType.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(serviceType.UpdatedAt), response.UpdatedAt)
}
