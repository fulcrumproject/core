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

// TestNewMetricEntryHandler tests the constructor
func TestNewMetricEntryHandler(t *testing.T) {
	querier := &mockMetricEntryQuerier{}
	serviceQuerier := &mockServiceQuerier{}
	commander := &mockMetricEntryCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewMetricEntryHandler(querier, serviceQuerier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, serviceQuerier, handler.serviceQuerier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestMetricEntryHandlerRoutes tests that routes are properly registered
func TestMetricEntryHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockMetricEntryQuerier{}
	serviceQuerier := &mockServiceQuerier{}
	commander := &mockMetricEntryCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewMetricEntryHandler(querier, serviceQuerier, commander, authz)

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
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestMetricEntryHandleList tests the handleList method
func TestMetricEntryHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockMetricEntryQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockMetricEntryQuerier) {
				// Setup the mock to return test metric entries
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				typeID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

				querier.listFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricEntry], error) {
					return &domain.PageResponse[domain.MetricEntry]{
						Items: []domain.MetricEntry{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								ServiceID:  serviceID,
								AgentID:    agentID,
								ConsumerID: consumerID,
								ProviderID: providerID,
								TypeID:     typeID,
								ResourceID: "resource-1",
								Value:      123.45,
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("bb0e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								ServiceID:  serviceID,
								AgentID:    agentID,
								ConsumerID: consumerID,
								ProviderID: providerID,
								TypeID:     typeID,
								ResourceID: "resource-2",
								Value:      678.90,
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
			mockSetup: func(querier *mockMetricEntryQuerier) {
				querier.listFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricEntry], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockMetricEntryQuerier{}
			serviceQuerier := &mockServiceQuerier{}
			commander := &mockMetricEntryCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier)

			// Create the handler
			handler := NewMetricEntryHandler(querier, serviceQuerier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/metric-entries?page=1&pageSize=10", nil)

			// Add auth identity to context (required by handler)
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleList(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK && tc.name == "Success" {
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
				assert.Equal(t, "aa0e8400-e29b-41d4-a716-446655440000", firstItem["id"])
				assert.Equal(t, "resource-1", firstItem["resourceId"])
				assert.Equal(t, 123.45, firstItem["value"])

				secondItem := items[1].(map[string]any)
				assert.Equal(t, "bb0e8400-e29b-41d4-a716-446655440000", secondItem["id"])
				assert.Equal(t, "resource-2", secondItem["resourceId"])
				assert.Equal(t, 678.9, secondItem["value"])
			}
		})
	}
}

// TestMetricEntryHandleCreate tests the handleCreate method
func TestMetricEntryHandleCreate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		requestBody    CreateMetricEntryRequest
		mockSetup      func(serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander)
		expectedStatus int
	}{
		{
			name: "SuccessWithServiceID",
			requestBody: CreateMetricEntryRequest{
				ServiceID:  &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				ResourceID: "resource-1",
				Value:      123.45,
				TypeName:   "cpu",
			},
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander) {
				// Use the same agent ID that's in NewMockAuthAgent
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				typeID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.findByIDFunc = func(ctx context.Context, id properties.UUID) (*domain.Service, error) {
					assert.Equal(t, serviceID, id)
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: serviceID,
						},
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
					}, nil
				}

				// Setup the auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.DefaultObjectScope{
						AgentID: &agentID,
					}, nil
				}

				// Setup the commander
				commander.createFunc = func(ctx context.Context, typeName string, agID properties.UUID, svcID properties.UUID, resourceID string, value float64) (*domain.MetricEntry, error) {
					assert.Equal(t, "cpu", typeName)
					assert.Equal(t, agentID, agID)
					assert.Equal(t, serviceID, svcID)
					assert.Equal(t, "resource-1", resourceID)
					assert.Equal(t, 123.45, value)

					return &domain.MetricEntry{
						BaseEntity: domain.BaseEntity{
							ID: uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
						},
						ServiceID:  serviceID,
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
						TypeID:     typeID,
						ResourceID: resourceID,
						Value:      value,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "SuccessWithExternalID",
			requestBody: CreateMetricEntryRequest{
				ExternalID: &[]string{"service-ext-1"}[0],
				ResourceID: "resource-1",
				Value:      123.45,
				TypeName:   "cpu",
			},
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander) {
				// Use the same agent ID that's in NewMockAuthAgent
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				typeID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.findByExternalIDFunc = func(ctx context.Context, aID properties.UUID, extID string) (*domain.Service, error) {
					assert.Equal(t, agentID, aID)
					assert.Equal(t, "service-ext-1", extID)
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: serviceID,
						},
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
					}, nil
				}

				// Setup the auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.DefaultObjectScope{
						AgentID: &agentID,
					}, nil
				}

				// Setup the commander
				commander.createWithExternalIDFunc = func(ctx context.Context, typeName string, agID properties.UUID, extID string, resourceID string, value float64) (*domain.MetricEntry, error) {
					assert.Equal(t, "cpu", typeName)
					assert.Equal(t, agentID, agID)
					assert.Equal(t, "service-ext-1", extID)
					assert.Equal(t, "resource-1", resourceID)
					assert.Equal(t, 123.45, value)

					return &domain.MetricEntry{
						BaseEntity: domain.BaseEntity{
							ID: uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
						},
						ServiceID:  serviceID,
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
						TypeID:     typeID,
						ResourceID: resourceID,
						Value:      value,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "ServiceQueryError",
			requestBody: CreateMetricEntryRequest{
				ServiceID:  &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				ResourceID: "resource-1",
				Value:      123.45,
				TypeName:   "cpu",
			},
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander) {
				// Setup the service querier to return an error
				serviceQuerier.findByIDFunc = func(ctx context.Context, id properties.UUID) (*domain.Service, error) {
					return nil, fmt.Errorf("service not found")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "AuthScopeError",
			requestBody: CreateMetricEntryRequest{
				ServiceID:  &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				ResourceID: "resource-1",
				Value:      123.45,
				TypeName:   "cpu",
			},
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander) {
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.findByIDFunc = func(ctx context.Context, id properties.UUID) (*domain.Service, error) {
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: serviceID,
						},
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
					}, nil
				}

				// Setup the auth scope to return an error
				serviceQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return nil, fmt.Errorf("auth scope error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "CommanderError",
			requestBody: CreateMetricEntryRequest{
				ServiceID:  &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				ResourceID: "resource-1",
				Value:      123.45,
				TypeName:   "cpu",
			},
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander) {
				// Use the same agent ID that's in NewMockAuthAgent
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.findByIDFunc = func(ctx context.Context, id properties.UUID) (*domain.Service, error) {
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: serviceID,
						},
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
					}, nil
				}

				// Setup the auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.DefaultObjectScope{
						AgentID: &agentID,
					}, nil
				}

				// Setup the commander to return an error
				commander.createFunc = func(ctx context.Context, typeName string, agID properties.UUID, svcID properties.UUID, resourceID string, value float64) (*domain.MetricEntry, error) {
					return nil, fmt.Errorf("metric creation error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockMetricEntryQuerier{}
			serviceQuerier := &mockServiceQuerier{}
			commander := &mockMetricEntryCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(serviceQuerier, commander)

			// Create the handler
			handler := NewMetricEntryHandler(querier, serviceQuerier, commander, authz)

			// Create request with body
			bodyBytes, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/metric-entries", bytes.NewReader(bodyBytes))
			req = req.WithContext(auth.WithIdentity(req.Context(), NewMockAuthAgent()))
			req.Header.Set("Content-Type", "application/json")

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[CreateMetricEntryRequest]()(http.HandlerFunc(handler.handleCreate))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusCreated {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Check basic structure
				assert.NotNil(t, response["id"])
				assert.Equal(t, "resource-1", response["resourceId"])
				assert.Equal(t, 123.45, response["value"])
			}
		})
	}
}

// TestMetricEntryToResponse tests the metricEntryToResponse function
func TestMetricEntryToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
	serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
	typeID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

	// Create a metric entry with all fields populated
	metricEntry := &domain.MetricEntry{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		ServiceID:  serviceID,
		AgentID:    agentID,
		ConsumerID: consumerID,
		ProviderID: providerID,
		TypeID:     typeID,
		ResourceID: "resource-1",
		Value:      123.45,
		// Add relationships
		Agent: &domain.Agent{
			BaseEntity: domain.BaseEntity{
				ID: agentID,
			},
			Name: "test-agent",
		},
		Service: &domain.Service{
			BaseEntity: domain.BaseEntity{
				ID: serviceID,
			},
			Name: "test-service",
		},
		Type: &domain.MetricType{
			BaseEntity: domain.BaseEntity{
				ID: typeID,
			},
			Name: "cpu",
		},
	}

	response := metricEntryToResponse(metricEntry)

	// Verify all fields are correctly mapped
	assert.Equal(t, metricEntry.ID, response.ID)
	assert.Equal(t, metricEntry.ResourceID, response.ResourceID)
	assert.Equal(t, metricEntry.Value, response.Value)
	assert.Equal(t, metricEntry.AgentID, response.AgentID)
	assert.Equal(t, metricEntry.ServiceID, response.ServiceID)
	assert.Equal(t, metricEntry.ConsumerID, response.ConsumerID)
	assert.Equal(t, metricEntry.ProviderID, response.ProviderID)
	assert.Equal(t, metricEntry.TypeID.String(), response.TypeID)
	assert.Equal(t, JSONUTCTime(metricEntry.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(metricEntry.UpdatedAt), response.UpdatedAt)

	// Verify relationships
	assert.NotNil(t, response.Agent)
	assert.Equal(t, metricEntry.Agent.ID, response.Agent.ID)
	assert.Equal(t, metricEntry.Agent.Name, response.Agent.Name)

	assert.NotNil(t, response.Service)
	assert.Equal(t, metricEntry.Service.ID, response.Service.ID)
	assert.Equal(t, metricEntry.Service.Name, response.Service.Name)

	assert.NotNil(t, response.Type)
	assert.Equal(t, metricEntry.Type.ID, response.Type.ID)
	assert.Equal(t, metricEntry.Type.Name, response.Type.Name)

	// Test with nil relationships
	metricEntry.Agent = nil
	metricEntry.Service = nil
	metricEntry.Type = nil

	responseSparse := metricEntryToResponse(metricEntry)

	assert.Nil(t, responseSparse.Agent)
	assert.Nil(t, responseSparse.Service)
	assert.Nil(t, responseSparse.Type)
}
