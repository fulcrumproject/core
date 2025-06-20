package api

import (
	"bytes"
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

// TestNewMetricTypeHandler tests the constructor
func TestNewMetricTypeHandler(t *testing.T) {
	querier := &mockMetricTypeQuerier{}
	commander := &mockMetricTypeCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewMetricTypeHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestMetricTypeHandlerRoutes tests that routes are properly registered
func TestMetricTypeHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockMetricTypeQuerier{}
	commander := &mockMetricTypeCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewMetricTypeHandler(querier, commander, authz)

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

// TestMetricTypeHandleCreate tests the handleCreate method
func TestMetricTypeHandleCreate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		requestBody    CreateMetricTypeRequest
		mockSetup      func(commander *mockMetricTypeCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			requestBody: CreateMetricTypeRequest{
				Name:       "CPU Usage",
				EntityType: domain.MetricEntityType("service"),
			},
			mockSetup: func(commander *mockMetricTypeCommander) {
				// Setup the commander
				commander.createFunc = func(ctx context.Context, name string, entityType domain.MetricEntityType) (*domain.MetricType, error) {
					assert.Equal(t, "CPU Usage", name)
					assert.Equal(t, domain.MetricEntityType("service"), entityType)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.MetricType{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       name,
						EntityType: entityType,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "CommanderError",
			requestBody: CreateMetricTypeRequest{
				Name:       "CPU Usage",
				EntityType: domain.MetricEntityType("service"),
			},
			mockSetup: func(commander *mockMetricTypeCommander) {
				// Setup the commander to return an error
				commander.createFunc = func(ctx context.Context, name string, entityType domain.MetricEntityType) (*domain.MetricType, error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockMetricTypeQuerier{}
			commander := &mockMetricTypeCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(commander)

			// Create the handler
			handler := NewMetricTypeHandler(querier, commander, authz)

			// Create request with body
			bodyBytes, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/metric-types", bytes.NewReader(bodyBytes))
			req = req.WithContext(auth.WithIdentity(req.Context(), NewMockAuthAdmin()))
			req.Header.Set("Content-Type", "application/json")

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[CreateMetricTypeRequest]()(http.HandlerFunc(handler.handleCreate))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusCreated {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", response["id"])
				assert.Equal(t, "CPU Usage", response["name"])
				assert.Equal(t, "service", response["entityType"])
			}
		})
	}
}

// TestMetricTypeHandleUpdate tests the handleUpdate method
func TestMetricTypeHandleUpdate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		requestBody    UpdateMetricTypeRequest
		mockSetup      func(commander *mockMetricTypeCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: UpdateMetricTypeRequest{
				Name: &[]string{"Updated CPU Usage"}[0],
			},
			mockSetup: func(commander *mockMetricTypeCommander) {
				// Setup the commander
				commander.updateFunc = func(ctx context.Context, id properties.UUID, name *string) (*domain.MetricType, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					assert.Equal(t, "Updated CPU Usage", *name)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.MetricType{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       *name,
						EntityType: domain.MetricEntityType("service"),
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "CommanderError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: UpdateMetricTypeRequest{
				Name: &[]string{"Updated CPU Usage"}[0],
			},
			mockSetup: func(commander *mockMetricTypeCommander) {
				// Setup the commander to return an error
				commander.updateFunc = func(ctx context.Context, id properties.UUID, name *string) (*domain.MetricType, error) {
					return nil, fmt.Errorf("update error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockMetricTypeQuerier{}
			commander := &mockMetricTypeCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(commander)

			// Create the handler
			handler := NewMetricTypeHandler(querier, commander, authz)

			// Create request with body
			bodyBytes, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)
			req := httptest.NewRequest("PATCH", "/metric-types/"+tc.id, bytes.NewReader(bodyBytes))
			req = req.WithContext(auth.WithIdentity(req.Context(), NewMockAuthAdmin()))
			req.Header.Set("Content-Type", "application/json")

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[UpdateMetricTypeRequest]()(middlewares.ID(http.HandlerFunc(handler.handleUpdate)))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, tc.id, response["id"])
				assert.Equal(t, "Updated CPU Usage", response["name"])
				assert.Equal(t, "service", response["entityType"])
			}
		})
	}
}

// TestMetricTypeToResponse tests the metricTypeToResponse function
func TestMetricTypeToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Create a metric type
	metricType := &domain.MetricType{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:       "CPU Usage",
		EntityType: domain.MetricEntityType("service"),
	}

	response := metricTypeToResponse(metricType)

	// Verify all fields are correctly mapped
	assert.Equal(t, metricType.ID, response.ID)
	assert.Equal(t, metricType.Name, response.Name)
	assert.Equal(t, metricType.EntityType, response.EntityType)
	assert.Equal(t, JSONUTCTime(metricType.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(metricType.UpdatedAt), response.UpdatedAt)
}
