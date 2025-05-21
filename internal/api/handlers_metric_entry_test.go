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
		mockSetup      func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return test metric entries
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				typeID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricEntry], error) {
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
			name: "Unauthorized",
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "ListError",
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.MetricEntry], error) {
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
			tc.mockSetup(querier, serviceQuerier, commander, authz)

			// Create the handler
			handler := NewMetricEntryHandler(querier, serviceQuerier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/metric-entries?page=1&pageSize=10", nil)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
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
				assert.Equal(t, "aa0e8400-e29b-41d4-a716-446655440000", firstItem["id"])
				assert.Equal(t, "resource-1", firstItem["resourceId"])
				assert.Equal(t, 123.45, firstItem["value"])

				secondItem := items[1].(map[string]interface{})
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
		requestBody    string
		mockSetup      func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name:        "SuccessWithServiceID",
			requestBody: `{"serviceId":"550e8400-e29b-41d4-a716-446655440000","resourceId":"resource-1","value":123.45,"typeName":"cpu"}`,
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Use the same agent ID that's in NewMockAuthAgent
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				typeID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
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
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.AuthScope{
						AgentID: &agentID,
					}, nil
				}

				// Setup the commander
				commander.createFunc = func(ctx context.Context, typeName string, agID domain.UUID, svcID domain.UUID, resourceID string, value float64) (*domain.MetricEntry, error) {
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
			name:        "SuccessWithExternalID",
			requestBody: `{"externalId":"service-ext-1","resourceId":"resource-1","value":123.45,"typeName":"cpu"}`,
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Use the same agent ID that's in NewMockAuthAgent
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				typeID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.findByExternalIDFunc = func(ctx context.Context, aID domain.UUID, extID string) (*domain.Service, error) {
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
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.AuthScope{
						AgentID: &agentID,
					}, nil
				}

				// Setup the commander
				commander.createWithExternalIDFunc = func(ctx context.Context, typeName string, agID domain.UUID, extID string, resourceID string, value float64) (*domain.MetricEntry, error) {
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
			name:        "InvalidRequestFormat",
			requestBody: `{"invalid"json}`,
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				// No setup needed for this case
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "MissingRequiredFields",
			requestBody: `{}`,
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				// No setup needed for this case
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "ServiceQueryError",
			requestBody: `{"serviceId":"550e8400-e29b-41d4-a716-446655440000","resourceId":"resource-1","value":123.45,"typeName":"cpu"}`,
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				// Setup the service querier to return an error
				serviceQuerier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
					return nil, fmt.Errorf("service not found")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "AuthScopeError",
			requestBody: `{"serviceId":"550e8400-e29b-41d4-a716-446655440000","resourceId":"resource-1","value":123.45,"typeName":"cpu"}`,
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
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
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return nil, fmt.Errorf("auth scope error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "AuthorizationError",
			requestBody: `{"serviceId":"550e8400-e29b-41d4-a716-446655440000","resourceId":"resource-1","value":123.45,"typeName":"cpu"}`,
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				// Use the same agent ID that's in NewMockAuthAgent
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
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
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.AuthScope{
						AgentID: &agentID,
					}, nil
				}

				// Make authorization fail
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden, // Authorization failures return 403 Forbidden
		},
		{
			name:        "CommanderError",
			requestBody: `{"serviceId":"550e8400-e29b-41d4-a716-446655440000","resourceId":"resource-1","value":123.45,"typeName":"cpu"}`,
			mockSetup: func(querier *mockMetricEntryQuerier, serviceQuerier *mockServiceQuerier, commander *mockMetricEntryCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Use the same agent ID that's in NewMockAuthAgent
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
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
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.AuthScope{
						AgentID: &agentID,
					}, nil
				}

				// Setup the commander to return an error
				commander.createFunc = func(ctx context.Context, typeName string, agID domain.UUID, svcID domain.UUID, resourceID string, value float64) (*domain.MetricEntry, error) {
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
			tc.mockSetup(querier, serviceQuerier, commander, authz)

			// Create the handler
			handler := NewMetricEntryHandler(querier, serviceQuerier, commander, authz)

			// Create request with JSON body
			req := httptest.NewRequest("POST", "/metric-entries", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Add auth identity to context for authorization
			// The NewMockAuthAgent() function already provides an initialized agent identity
			authIdentity := NewMockAuthAgent()
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
