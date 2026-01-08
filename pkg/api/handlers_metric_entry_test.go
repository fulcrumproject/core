package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestNewMetricEntryHandler tests the constructor
func TestNewMetricEntryHandler(t *testing.T) {
	querier := domain.NewMockMetricEntryQuerier(t)
	serviceQuerier := domain.NewMockServiceQuerier(t)
	commander := domain.NewMockMetricEntryCommander(t)
	authz := authz.NewMockAuthorizer(t)

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
	querier := domain.NewMockMetricEntryQuerier(t)
	serviceQuerier := domain.NewMockServiceQuerier(t)
	commander := domain.NewMockMetricEntryCommander(t)
	authz := authz.NewMockAuthorizer(t)

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

// TestMetricEntryHandleCreate tests the handleCreate method
func TestMetricEntryHandleCreate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		requestBody    CreateMetricEntryReq
		mockSetup      func(serviceQuerier *domain.MockServiceQuerier, commander *domain.MockMetricEntryCommander)
		expectedStatus int
	}{
		{
			name: "SuccessWithServiceID",
			requestBody: CreateMetricEntryReq{
				ServiceID:  &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				ResourceID: "resource-1",
				Value:      123.45,
				TypeName:   "cpu",
			},
			mockSetup: func(serviceQuerier *domain.MockServiceQuerier, commander *domain.MockMetricEntryCommander) {
				// Use the same agent ID that's in NewMockAuthAgent
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				typeID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.EXPECT().
					Get(mock.Anything, serviceID).
					Return(&domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: serviceID,
						},
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
					}, nil)

				// Setup the commander
				commander.EXPECT().
					Create(mock.Anything, mock.MatchedBy(func(params domain.CreateMetricEntryParams) bool {
						return params.TypeName == "cpu" &&
							params.AgentID == agentID &&
							params.ServiceID == serviceID &&
							params.ResourceID == "resource-1" &&
							params.Value == 123.45
					})).
					Return(&domain.MetricEntry{
						ID:         uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
						ServiceID:  serviceID,
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
						TypeID:     typeID,
						ResourceID: "resource-1",
						Value:      123.45,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "SuccessWithAgentInstanceID",
			requestBody: CreateMetricEntryReq{
				AgentInstanceID: &[]string{"service-inst-1"}[0],
				ResourceID:      "resource-1",
				Value:           123.45,
				TypeName:        "cpu",
			},
			mockSetup: func(serviceQuerier *domain.MockServiceQuerier, commander *domain.MockMetricEntryCommander) {
				// Use the same agent ID that's in NewMockAuthAgent
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				typeID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.EXPECT().
					FindByAgentInstanceID(mock.Anything, agentID, "service-inst-1").
					Return(&domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: serviceID,
						},
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
					}, nil)

				// Setup the commander
				commander.EXPECT().
					CreateWithAgentInstanceID(mock.Anything, mock.MatchedBy(func(params domain.CreateMetricEntryWithAgentInstanceIDParams) bool {
						return params.TypeName == "cpu" &&
							params.AgentID == agentID &&
							params.AgentInstanceID == "service-inst-1" &&
							params.ResourceID == "resource-1" &&
							params.Value == 123.45
					})).
					Return(&domain.MetricEntry{
						ID:         uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
						ServiceID:  serviceID,
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
						TypeID:     typeID,
						ResourceID: "resource-1",
						Value:      123.45,
					}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "ServiceQueryError",
			requestBody: CreateMetricEntryReq{
				ServiceID:  &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				ResourceID: "resource-1",
				Value:      123.45,
				TypeName:   "cpu",
			},
			mockSetup: func(serviceQuerier *domain.MockServiceQuerier, commander *domain.MockMetricEntryCommander) {
				// Setup the service querier to return an error
				serviceQuerier.EXPECT().
					Get(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("service not found"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		// AuthScope is not called in the current handler implementation
		// so we skip this test case
		{
			name: "CommanderError",
			requestBody: CreateMetricEntryReq{
				ServiceID:  &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				ResourceID: "resource-1",
				Value:      123.45,
				TypeName:   "cpu",
			},
			mockSetup: func(serviceQuerier *domain.MockServiceQuerier, commander *domain.MockMetricEntryCommander) {
				// Use the same agent ID that's in NewMockAuthAgent
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")
				serviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

				// Setup the service querier
				serviceQuerier.EXPECT().
					Get(mock.Anything, serviceID).
					Return(&domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: serviceID,
						},
						AgentID:    agentID,
						ConsumerID: consumerID,
						ProviderID: providerID,
					}, nil)

				// Setup the commander to return an error
				commander.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("metric creation error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := domain.NewMockMetricEntryQuerier(t)
			serviceQuerier := domain.NewMockServiceQuerier(t)
			commander := domain.NewMockMetricEntryCommander(t)
			authz := authz.NewMockAuthorizer(t)
			tc.mockSetup(serviceQuerier, commander)

			// Create the handler
			handler := NewMetricEntryHandler(querier, serviceQuerier, commander, authz)

			// Create request with body
			bodyBytes, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/metric-entries", bytes.NewReader(bodyBytes))
			req = req.WithContext(auth.WithIdentity(req.Context(), newMockAuthAgent()))
			req.Header.Set("Content-Type", "application/json")

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[CreateMetricEntryReq]()(http.HandlerFunc(handler.Create))
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
		ID:         uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
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

	response := MetricEntryToRes(metricEntry)

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

	responseSparse := MetricEntryToRes(metricEntry)

	assert.Nil(t, responseSparse.Agent)
	assert.Nil(t, responseSparse.Service)
	assert.Nil(t, responseSparse.Type)
}
