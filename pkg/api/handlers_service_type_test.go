package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/middlewares"
	"github.com/fulcrumproject/commons/properties"
	"github.com/fulcrumproject/core/pkg/domain"
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
	// Create mocks
	querier := &mockServiceTypeQuerier{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewServiceTypeHandler(querier, authz)

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
		case method == "GET" && route == "/{id}":
		case method == "POST" && route == "/{id}/validate":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestServiceTypeHandleGet tests the handleGet method
func TestServiceTypeHandleGet(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockServiceTypeQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockServiceTypeQuerier) {
				querier.findByIDFunc = func(ctx context.Context, id properties.UUID) (*domain.ServiceType, error) {
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
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockServiceTypeQuerier) {
				// Setup the querier to return not found
				querier.findByIDFunc = func(ctx context.Context, id properties.UUID) (*domain.ServiceType, error) {
					return nil, domain.NotFoundError{Err: fmt.Errorf("service type not found")}
				}
			},
			expectedStatus: http.StatusNotFound, // ErrDomain checks for NotFoundError and returns 404
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockServiceTypeQuerier{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier)

			// Create the handler
			handler := NewServiceTypeHandler(querier, authz)

			// Create request
			req := httptest.NewRequest("GET", "/service-types/"+tc.id, nil)

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.ID(http.HandlerFunc(handler.handleGet))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
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
		mockSetup      func(querier *mockServiceTypeQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockServiceTypeQuerier) {
				// Setup the mock to return service types
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.listFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceType], error) {
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
			name: "ListError",
			mockSetup: func(querier *mockServiceTypeQuerier) {
				querier.listFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceType], error) {
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
			tc.mockSetup(querier)

			// Create the handler
			handler := NewServiceTypeHandler(querier, authz)

			// Create request
			req := httptest.NewRequest("GET", "/service-types?page=1&pageSize=10", nil)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleList(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, float64(1), response["currentPage"])
				assert.Equal(t, float64(2), response["totalItems"])
				assert.Equal(t, float64(1), response["totalPages"])
				assert.Equal(t, false, response["hasNext"])
				assert.Equal(t, false, response["hasPrev"])

				items := response["items"].([]any)
				assert.Equal(t, 2, len(items))

				firstItem := items[0].(map[string]any)
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", firstItem["id"])
				assert.Equal(t, "VM Instance", firstItem["name"])

				secondItem := items[1].(map[string]any)
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
