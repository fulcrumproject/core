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
	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewServiceHandler tests the constructor
func TestNewServiceHandler(t *testing.T) {
	serviceQuerier := &mockServiceQuerier{}
	agentQuerier := &mockAgentQuerier{}
	serviceGroupQuerier := &mockServiceGroupQuerier{}
	commander := &mockServiceCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, serviceQuerier, handler.querier)
	assert.Equal(t, agentQuerier, handler.agentQuerier)
	assert.Equal(t, serviceGroupQuerier, handler.serviceGroupQuerier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestServiceHandlerRoutes tests that routes are properly registered with middlewares
func TestServiceHandlerRoutes(t *testing.T) {
	// Create mocks
	serviceQuerier := &mockServiceQuerier{}
	agentQuerier := &mockAgentQuerier{}
	serviceGroupQuerier := &mockServiceGroupQuerier{}
	commander := &mockServiceCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

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
			// Check for authorization middleware
			assert.GreaterOrEqual(t, len(middlewares), 1, "List route should have at least authorization middleware")
		case method == "POST" && route == "/":
			// Check for decode body and authorization middlewares
			assert.GreaterOrEqual(t, len(middlewares), 1, "Create route should have body decoder and specialized extractor middlewares")
		case method == "GET" && route == "/{id}":
			// Check for authorization middleware
			assert.GreaterOrEqual(t, len(middlewares), 1, "Get route should have authorization middleware")
		case method == "PATCH" && route == "/{id}":
			// Check for decode body and authorization middlewares
			assert.GreaterOrEqual(t, len(middlewares), 2, "Update route should have body decoder and authorization middlewares")
		case method == "POST" && route == "/{id}/start":
			// Check for authorization middleware
			assert.GreaterOrEqual(t, len(middlewares), 1, "Start route should have authorization middleware")
		case method == "POST" && route == "/{id}/stop":
			// Check for authorization middleware
			assert.GreaterOrEqual(t, len(middlewares), 1, "Stop route should have authorization middleware")
		case method == "DELETE" && route == "/{id}":
			// Check for authorization middleware
			assert.GreaterOrEqual(t, len(middlewares), 1, "Delete route should have authorization middleware")
		case method == "POST" && route == "/{id}/retry":
			// Check for authorization middleware
			assert.GreaterOrEqual(t, len(middlewares), 1, "Retry route should have authorization middleware")
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestServiceHandleCreate tests the handleCreate method
func TestServiceHandleCreate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		request        CreateServiceReq
		mockSetup      func(commander *mockServiceCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			request: CreateServiceReq{
				Name:          "Test Service",
				AgentID:       &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				GroupID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
				ServiceTypeID: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
				Properties:    properties.JSON{"prop": "value"},
			},
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander for successful creation
				commander.createFunc = func(ctx context.Context, params domain.CreateServiceParams) (*domain.Service, error) {
					assert.Equal(t, "Test Service", params.Name)
					assert.Equal(t, properties.JSON{"prop": "value"}, params.Properties)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					consumerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
					providerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:          params.Name,
						AgentID:       params.AgentID,
						ServiceTypeID: params.ServiceTypeID,
						GroupID:       params.GroupID,
						ConsumerID:    consumerID,
						ProviderID:    providerID,
						Status:        domain.ServiceNew,
						Properties:    &params.Properties,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "CommanderError",
			request: CreateServiceReq{
				Name:          "Test Service",
				AgentID:       &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				GroupID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
				ServiceTypeID: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
				Properties:    properties.JSON{"prop": "value"},
			},
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander to return an error
				commander.createFunc = func(ctx context.Context, params domain.CreateServiceParams) (*domain.Service, error) {
					return nil, domain.NewInvalidInputErrorf("invalid input")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			serviceQuerier := &mockServiceQuerier{}
			agentQuerier := &mockAgentQuerier{}
			serviceGroupQuerier := &mockServiceGroupQuerier{}
			commander := &mockServiceCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true} // Not used in handler tests
			tc.mockSetup(commander)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request with body
			bodyBytes, err := json.Marshal(tc.request)
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/services", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[CreateServiceReq]()(http.HandlerFunc(handler.Create))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusCreated {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, "aa0e8400-e29b-41d4-a716-446655440000", response["id"])
				assert.Equal(t, "Test Service", response["name"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
			}
		})
	}
}

// TestServiceHandleUpdate tests the handleUpdate method
func TestServiceHandleUpdate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		request        UpdateServiceReq
		mockSetup      func(commander *mockServiceCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceReq{
				Name:       helpers.StringPtr("Updated Service"),
				Properties: helpers.JSONPtr(properties.JSON{"updated": "value"}),
			},
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander for successful update
				commander.updateFunc = func(ctx context.Context, params domain.UpdateServiceParams) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), params.ID)
					assert.Equal(t, "Updated Service", *params.Name)
					assert.Equal(t, properties.JSON{"updated": "value"}, *params.Properties)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID:        params.ID,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       "Updated Service",
						Status:     domain.ServiceStarted,
						Properties: params.Properties,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "ValidationError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceReq{
				Name:       helpers.StringPtr("Invalid"),
				Properties: helpers.JSONPtr(properties.JSON{"invalid": "data"}),
			},
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander to return a validation error
				commander.updateFunc = func(ctx context.Context, params domain.UpdateServiceParams) (*domain.Service, error) {
					return nil, domain.NewInvalidInputErrorf("validation error")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceReq{
				Name:       helpers.StringPtr("Updated Service"),
				Properties: helpers.JSONPtr(properties.JSON{"updated": "value"}),
			},
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander to return not found
				commander.updateFunc = func(ctx context.Context, params domain.UpdateServiceParams) (*domain.Service, error) {
					return nil, domain.NewNotFoundErrorf("service not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			serviceQuerier := &mockServiceQuerier{}
			agentQuerier := &mockAgentQuerier{}
			serviceGroupQuerier := &mockServiceGroupQuerier{}
			commander := &mockServiceCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true} // Not used in handler tests
			tc.mockSetup(commander)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request
			// Create request with body
			bodyBytes, err := json.Marshal(tc.request)
			require.NoError(t, err)
			req := httptest.NewRequest("PATCH", "/services/"+tc.id, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[UpdateServiceReq]()(middlewares.ID(Update(handler.Update, ServiceToRes)))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, tc.id, response["id"])
				assert.Equal(t, "Updated Service", response["name"])
				props, ok := response["properties"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "value", props["updated"])
			}
		})
	}
}

// TestServiceHandleTransition tests handleStart, handleStop, and handleDelete via handleTransition
func TestServiceHandleTransition(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		transitionTo   domain.ServiceStatus
		mockSetup      func(commander *mockServiceCommander)
		expectedStatus int
	}{
		{
			name:         "SuccessfulStart",
			id:           "550e8400-e29b-41d4-a716-446655440000",
			transitionTo: domain.ServiceStarted,
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander for successful transition
				commander.doActionFunc = func(ctx context.Context, params domain.DoServiceActionParams) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), params.ID)
					assert.Equal(t, domain.ServiceActionStart, params.Action)
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: params.ID,
						},
						Status: domain.ServiceStarted,
					}, nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:         "SuccessfulStop",
			id:           "550e8400-e29b-41d4-a716-446655440000",
			transitionTo: domain.ServiceStopped,
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander for successful transition
				commander.doActionFunc = func(ctx context.Context, params domain.DoServiceActionParams) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), params.ID)
					assert.Equal(t, domain.ServiceActionStop, params.Action)
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: params.ID,
						},
						Status: domain.ServiceStopped,
					}, nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:         "SuccessfulDelete",
			id:           "550e8400-e29b-41d4-a716-446655440000",
			transitionTo: domain.ServiceDeleted,
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander for successful transition
				commander.doActionFunc = func(ctx context.Context, params domain.DoServiceActionParams) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), params.ID)
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: params.ID,
						},
						Status: domain.ServiceDeleted,
					}, nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:         "InvalidStatusTransition",
			id:           "550e8400-e29b-41d4-a716-446655440000",
			transitionTo: domain.ServiceStarted,
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander to return an error for invalid transition
				commander.doActionFunc = func(ctx context.Context, params domain.DoServiceActionParams) (*domain.Service, error) {
					return nil, domain.NewInvalidInputErrorf("invalid status transition")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "ServiceNotFound",
			id:           "550e8400-e29b-41d4-a716-446655440000",
			transitionTo: domain.ServiceStarted,
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander to return not found
				commander.doActionFunc = func(ctx context.Context, params domain.DoServiceActionParams) (*domain.Service, error) {
					return nil, domain.NewNotFoundErrorf("service not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			serviceQuerier := &mockServiceQuerier{}
			agentQuerier := &mockAgentQuerier{}
			serviceGroupQuerier := &mockServiceGroupQuerier{}
			commander := &mockServiceCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true} // Not used in handler tests
			tc.mockSetup(commander)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request
			req := httptest.NewRequest("POST", "/services/"+tc.id+"/action", nil)

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.ID(CommandWithoutBody(func(ctx context.Context, id properties.UUID) error {
				switch tc.transitionTo {
				case domain.ServiceStarted:
					return handler.Start(ctx, id)
				case domain.ServiceStopped:
					return handler.Stop(ctx, id)
				case domain.ServiceDeleted:
					return handler.Delete(ctx, id)
				default:
					return fmt.Errorf("unsupported transition: %v", tc.transitionTo)
				}
			}))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestServiceHandleRetry tests the handleRetry method
func TestServiceHandleRetry(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(commander *mockServiceCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander for successful retry
				commander.retryFunc = func(ctx context.Context, id properties.UUID) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: id,
						},
						Status: domain.ServiceNew,
					}, nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "ServiceNotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander to return not found
				commander.retryFunc = func(ctx context.Context, id properties.UUID) (*domain.Service, error) {
					return nil, domain.NewNotFoundErrorf("service not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "InvalidRetry",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander to return an error for invalid retry
				commander.retryFunc = func(ctx context.Context, id properties.UUID) (*domain.Service, error) {
					return nil, domain.NewInvalidInputErrorf("service is not in a failed status")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			serviceQuerier := &mockServiceQuerier{}
			agentQuerier := &mockAgentQuerier{}
			serviceGroupQuerier := &mockServiceGroupQuerier{}
			commander := &mockServiceCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true} // Not used in handler tests
			tc.mockSetup(commander)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request
			req := httptest.NewRequest("POST", "/services/"+tc.id+"/retry", nil)

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.ID(CommandWithoutBody(handler.Retry))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestServiceToResponse tests the serviceToResponse function
func TestServiceToResponse(t *testing.T) {
	// Create a domain.Service
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	agentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	serviceTypeID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	groupID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	consumerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
	providerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
	serviceID := uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000")

	externalID := "ext-123"
	props := properties.JSON{"key": "value"}
	resources := properties.JSON{"cpu": "1", "memory": "2GB"}

	service := &domain.Service{
		BaseEntity: domain.BaseEntity{
			ID:        serviceID,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:          "Test Service",
		AgentID:       agentID,
		ServiceTypeID: serviceTypeID,
		GroupID:       groupID,
		ConsumerID:    consumerID,
		ProviderID:    providerID,
		ExternalID:    &externalID,
		Status:        domain.ServiceNew,
		Properties:    &props,
		Resources:     &resources,
	}

	// Convert to response
	response := ServiceToRes(service)

	// Verify response
	assert.Equal(t, serviceID, response.ID)
	assert.Equal(t, "Test Service", response.Name)
	assert.Equal(t, agentID, response.AgentID)
	assert.Equal(t, serviceTypeID, response.ServiceTypeID)
	assert.Equal(t, groupID, response.GroupID)
	assert.Equal(t, consumerID, response.ConsumerID)
	assert.Equal(t, providerID, response.ProviderID)
	assert.Equal(t, externalID, *response.ExternalID)
	assert.Equal(t, domain.ServiceNew, response.Status)
	assert.Equal(t, props, *response.Properties)
	assert.Equal(t, resources, *response.Resources)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
}
