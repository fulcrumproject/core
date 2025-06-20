package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentTypeHandleGet tests the handleGet method (pure business logic)
func TestAgentTypeHandleGet(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockAgentTypeQuerier)
		expectedStatus int
		expectedBody   map[string]any
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentTypeQuerier) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.GetFunc = func(ctx context.Context, id properties.UUID) (*domain.AgentType, error) {
					return &domain.AgentType{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name: "TestAgentType",
						ServiceTypes: []domain.ServiceType{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("650e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name: "TestServiceType",
							},
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]any{
				"id":        "550e8400-e29b-41d4-a716-446655440000",
				"name":      "TestAgentType",
				"createdAt": "2023-01-01T00:00:00Z",
				"updatedAt": "2023-01-01T00:00:00Z",
				"serviceTypes": []any{
					map[string]any{
						"id":        "650e8400-e29b-41d4-a716-446655440000",
						"name":      "TestServiceType",
						"createdAt": "2023-01-01T00:00:00Z",
						"updatedAt": "2023-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentTypeQuerier) {
				querier.GetFunc = func(ctx context.Context, id properties.UUID) (*domain.AgentType, error) {
					return nil, domain.NewNotFoundErrorf("agent type not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockAgentTypeQuerier{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier)

			// Create the handler
			handler := NewAgentTypeHandler(querier, authz)

			// Create request with simulated middleware context
			req := httptest.NewRequest("GET", "/agent-types/"+tc.id, nil)

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context (required by all handlers)
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.ID(http.HandlerFunc(handler.handleGet))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

// TestAgentTypeHandleList tests the handleList method (pure business logic)
func TestAgentTypeHandleList(t *testing.T) {
	testCases := []struct {
		name           string
		queryParams    string
		mockSetup      func(querier *mockAgentTypeQuerier)
		expectedStatus int
		expectedBody   map[string]any
	}{
		{
			name:        "Success",
			queryParams: "?page=1&pageSize=10",
			mockSetup: func(querier *mockAgentTypeQuerier) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.ListFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.AgentType], error) {
					return &domain.PageResponse[domain.AgentType]{
						Items: []domain.AgentType{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name: "TestAgentType1",
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name: "TestAgentType2",
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
			expectedBody: map[string]any{
				"items": []any{
					map[string]any{
						"id":           "550e8400-e29b-41d4-a716-446655440000",
						"name":         "TestAgentType1",
						"createdAt":    "2023-01-01T00:00:00Z",
						"updatedAt":    "2023-01-01T00:00:00Z",
						"serviceTypes": []any{},
					},
					map[string]any{
						"id":           "660e8400-e29b-41d4-a716-446655440000",
						"name":         "TestAgentType2",
						"createdAt":    "2023-01-01T00:00:00Z",
						"updatedAt":    "2023-01-01T00:00:00Z",
						"serviceTypes": []any{},
					},
				},
				"currentPage": float64(1),
				"totalItems":  float64(2),
				"totalPages":  float64(1),
				"hasNext":     false,
				"hasPrev":     false,
			},
		},
		{
			name:        "EmptyResult",
			queryParams: "?page=1&pageSize=10",
			mockSetup: func(querier *mockAgentTypeQuerier) {
				querier.ListFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.AgentType], error) {
					return &domain.PageResponse[domain.AgentType]{
						Items:       []domain.AgentType{},
						TotalItems:  0,
						CurrentPage: 1,
						TotalPages:  0,
						HasNext:     false,
						HasPrev:     false,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]any{
				"items":       []any{},
				"totalItems":  float64(0),
				"totalPages":  float64(0),
				"currentPage": float64(1),
				"hasNext":     false,
				"hasPrev":     false,
			},
		},
		{
			name:        "InvalidPageRequest",
			queryParams: "?page=invalid",
			mockSetup: func(querier *mockAgentTypeQuerier) {
				// parsePageRequest uses defaults for invalid values, so this will still call the querier
				querier.ListFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.AgentType], error) {
					return &domain.PageResponse[domain.AgentType]{
						Items:       []domain.AgentType{},
						TotalItems:  0,
						CurrentPage: 1,
						TotalPages:  0,
						HasNext:     false,
						HasPrev:     false,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]any{
				"items":       []any{},
				"totalItems":  float64(0),
				"totalPages":  float64(0),
				"currentPage": float64(1),
				"hasNext":     false,
				"hasPrev":     false,
			},
		},
		{
			name:        "ListError",
			queryParams: "?page=1&pageSize=10",
			mockSetup: func(querier *mockAgentTypeQuerier) {
				querier.ListFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.AgentType], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockAgentTypeQuerier{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier)

			// Create the handler
			handler := NewAgentTypeHandler(querier, authz)

			// Create request with simulated middleware context
			req := httptest.NewRequest("GET", "/agent-types"+tc.queryParams, nil)

			// Add auth identity to context (required by all handlers)
			authIdentity := NewMockAuthAgent()
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
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

// TestNewAgentTypeHandler tests the constructor
func TestNewAgentTypeHandler(t *testing.T) {
	querier := &mockAgentTypeQuerier{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewAgentTypeHandler(querier, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, authz, handler.authz)
}

// TestAgentTypeHandlerRoutes tests that routes are properly registered
func TestAgentTypeHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockAgentTypeQuerier{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewAgentTypeHandler(querier, authz)

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
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestAgentTypeToResponse tests the agentTypeToResponse function
func TestAgentTypeToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	agentType := &domain.AgentType{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name: "TestAgentType",
		ServiceTypes: []domain.ServiceType{
			{
				BaseEntity: domain.BaseEntity{
					ID:        uuid.MustParse("650e8400-e29b-41d4-a716-446655440000"),
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Name: "TestServiceType",
			},
		},
	}

	response := agentTypeToResponse(agentType)

	assert.Equal(t, agentType.ID, response.ID)
	assert.Equal(t, agentType.Name, response.Name)
	assert.Equal(t, JSONUTCTime(agentType.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(agentType.UpdatedAt), response.UpdatedAt)
	assert.Len(t, response.ServiceTypes, 1)
	assert.Equal(t, agentType.ServiceTypes[0].ID, response.ServiceTypes[0].ID)
	assert.Equal(t, agentType.ServiceTypes[0].Name, response.ServiceTypes[0].Name)
}
