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

			// Create request with simulated middleware context
			req := httptest.NewRequest("POST", "/metric-types", nil)
			req = req.WithContext(context.WithValue(req.Context(), decodedBodyContextKey, tc.requestBody))
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), NewMockAuthAdmin()))

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
				assert.Equal(t, "CPU Usage", response["name"])
				assert.Equal(t, "service", response["entityType"])
			}
		})
	}
}

// TestMetricTypeHandleGet tests the handleGet method
func TestMetricTypeHandleGet(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockMetricTypeQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockMetricTypeQuerier) {
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.MetricType, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.MetricType{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       "CPU Usage",
						EntityType: domain.MetricEntityType("service"),
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockMetricTypeQuerier) {
				// Setup the querier to return not found
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.MetricType, error) {
					return nil, domain.NotFoundError{Err: fmt.Errorf("metric type not found")}
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockMetricTypeQuerier{}
			commander := &mockMetricTypeCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier)

			// Create the handler
			handler := NewMetricTypeHandler(querier, commander, authz)

			// Create request with simulated middleware context
			req := httptest.NewRequest("GET", "/metric-types/"+tc.id, nil)

			// Simulate ID middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), NewMockAuthAdmin()))

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
				assert.Equal(t, "CPU Usage", response["name"])
				assert.Equal(t, "service", response["entityType"])
			}
		})
	}
}

// TestMetricTypeHandleList tests the handleList method
func TestMetricTypeHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockMetricTypeQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockMetricTypeQuerier) {
				// Setup the mock to return metric types
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricType], error) {
					return &domain.PageResponse[domain.MetricType]{
						Items: []domain.MetricType{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:       "CPU Usage",
								EntityType: domain.MetricEntityType("service"),
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:       "Memory Usage",
								EntityType: domain.MetricEntityType("service"),
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
			mockSetup: func(querier *mockMetricTypeQuerier) {
				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricType], error) {
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
			tc.mockSetup(querier)

			// Create the handler
			handler := NewMetricTypeHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/metric-types?page=1&pageSize=10", nil)

			// Add auth identity to context (required by handler)
			authIdentity := NewMockAuthAdmin()
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
				assert.Equal(t, "CPU Usage", firstItem["name"])
				assert.Equal(t, "service", firstItem["entityType"])

				secondItem := items[1].(map[string]interface{})
				assert.Equal(t, "660e8400-e29b-41d4-a716-446655440000", secondItem["id"])
				assert.Equal(t, "Memory Usage", secondItem["name"])
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
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string) (*domain.MetricType, error) {
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
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string) (*domain.MetricType, error) {
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

			// Create request with simulated middleware context
			req := httptest.NewRequest("PATCH", "/metric-types/"+tc.id, nil)

			// Simulate ID and DecodeBody middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
			req = req.WithContext(context.WithValue(req.Context(), decodedBodyContextKey, tc.requestBody))
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), NewMockAuthAdmin()))

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
				assert.Equal(t, "Updated CPU Usage", response["name"])
				assert.Equal(t, "service", response["entityType"])
			}
		})
	}
}

// TestMetricTypeHandleDelete tests the handleDelete method
func TestMetricTypeHandleDelete(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockMetricTypeQuerier, commander *mockMetricTypeCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockMetricTypeQuerier, commander *mockMetricTypeCommander) {
				// Setup the querier to find the metric type
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.MetricType, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.MetricType{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       "CPU Usage",
						EntityType: domain.MetricEntityType("service"),
					}, nil
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
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockMetricTypeQuerier, commander *mockMetricTypeCommander) {
				// Setup the querier to find no metric type
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.MetricType, error) {
					return nil, domain.NotFoundError{Err: fmt.Errorf("metric type not found")}
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "DeleteError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockMetricTypeQuerier, commander *mockMetricTypeCommander) {
				// Setup the querier to find the metric type
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.MetricType, error) {
					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.MetricType{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       "CPU Usage",
						EntityType: domain.MetricEntityType("service"),
					}, nil
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
			querier := &mockMetricTypeQuerier{}
			commander := &mockMetricTypeCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander)

			// Create the handler
			handler := NewMetricTypeHandler(querier, commander, authz)

			// Create request with simulated middleware context
			req := httptest.NewRequest("DELETE", "/metric-types/"+tc.id, nil)

			// Simulate ID middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), NewMockAuthAdmin()))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleDelete(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
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
