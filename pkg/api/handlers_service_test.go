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
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestNewServiceHandler tests the constructor
func TestNewServiceHandler(t *testing.T) {
	serviceQuerier := domain.NewMockServiceQuerier(t)
	agentQuerier := domain.NewMockAgentQuerier(t)
	serviceGroupQuerier := domain.NewMockServiceGroupQuerier(t)
	commander := domain.NewMockServiceCommander(t)
	authz := authz.NewMockAuthorizer(t)

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
	serviceQuerier := domain.NewMockServiceQuerier(t)
	agentQuerier := domain.NewMockAgentQuerier(t)
	serviceGroupQuerier := domain.NewMockServiceGroupQuerier(t)
	commander := domain.NewMockServiceCommander(t)
	authz := authz.NewMockAuthorizer(t)

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
		case method == "DELETE" && route == "/{id}":
			// Check for authorization middleware
			assert.GreaterOrEqual(t, len(middlewares), 1, "Delete route should have authorization middleware")
		case method == "POST" && route == "/{id}/retry":
			// Check for authorization middleware
			assert.GreaterOrEqual(t, len(middlewares), 1, "Retry route should have authorization middleware")
		case method == "POST" && route == "/{id}/{action}":
			// Generic action route - check for action name middleware and authorization
			assert.GreaterOrEqual(t, len(middlewares), 2, "Generic action route should have action name middleware and authorization middleware")
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
		mockSetup      func(commander *domain.MockServiceCommander)
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
			mockSetup: func(commander *domain.MockServiceCommander) {
				// Setup the commander for successful creation
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				consumerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

				commander.EXPECT().
					Create(mock.Anything, mock.MatchedBy(func(params domain.CreateServiceParams) bool {
						return params.Name == "Test Service" &&
							params.Properties["prop"] == "value"
					})).
					Return(&domain.Service{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:          "Test Service",
						AgentID:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						ServiceTypeID: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
						GroupID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
						ConsumerID:    consumerID,
						ProviderID:    providerID,
						Status:        "New",
						Properties:    &properties.JSON{"prop": "value"},
					}, nil)
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
			mockSetup: func(commander *domain.MockServiceCommander) {
				// Setup the commander to return an error
				commander.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(nil, domain.NewInvalidInputErrorf("invalid input"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			serviceQuerier := domain.NewMockServiceQuerier(t)
			agentQuerier := domain.NewMockAgentQuerier(t)
			serviceGroupQuerier := domain.NewMockServiceGroupQuerier(t)
			commander := domain.NewMockServiceCommander(t)
			authz := authz.NewMockAuthorizer(t) // Not used in handler tests
			tc.mockSetup(commander)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request with body
			bodyBytes, err := json.Marshal(tc.request)
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/services", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Add auth identity to context (always required)
			authIdentity := newMockAuthAdmin()
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
		mockSetup      func(commander *domain.MockServiceCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceReq{
				Name:       helpers.StringPtr("Updated Service"),
				Properties: helpers.JSONPtr(properties.JSON{"updated": "value"}),
			},
			mockSetup: func(commander *domain.MockServiceCommander) {
				// Setup the commander for successful update
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
				updatedName := "Updated Service"
				updatedProps := properties.JSON{"updated": "value"}

				commander.EXPECT().
					Update(mock.Anything, mock.MatchedBy(func(params domain.UpdateServiceParams) bool {
						return params.ID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440000") &&
							*params.Name == "Updated Service" &&
							(*params.Properties)["updated"] == "value"
					})).
					Return(&domain.Service{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       updatedName,
						Status:     "Started",
						Properties: &updatedProps,
					}, nil)
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
			mockSetup: func(commander *domain.MockServiceCommander) {
				// Setup the commander to return a validation error
				commander.EXPECT().
					Update(mock.Anything, mock.Anything).
					Return(nil, domain.NewInvalidInputErrorf("validation error"))
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
			mockSetup: func(commander *domain.MockServiceCommander) {
				// Setup the commander to return not found
				commander.EXPECT().
					Update(mock.Anything, mock.Anything).
					Return(nil, domain.NewNotFoundErrorf("service not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			serviceQuerier := domain.NewMockServiceQuerier(t)
			agentQuerier := domain.NewMockAgentQuerier(t)
			serviceGroupQuerier := domain.NewMockServiceGroupQuerier(t)
			commander := domain.NewMockServiceCommander(t)
			authz := authz.NewMockAuthorizer(t) // Not used in handler tests
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
			authIdentity := newMockAuthAdmin()
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
		transitionTo   string
		mockSetup      func(commander *domain.MockServiceCommander)
		expectedStatus int
	}{
		{
			name:         "SuccessfulStart",
			id:           "550e8400-e29b-41d4-a716-446655440000",
			transitionTo: "Started",
			mockSetup: func(commander *domain.MockServiceCommander) {
				// Setup the commander for successful transition
				commander.EXPECT().
					DoAction(mock.Anything, mock.MatchedBy(func(params domain.DoServiceActionParams) bool {
						return params.ID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440000") &&
							params.Action == "start"
					})).
					Return(&domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						},
						Status: "Started",
					}, nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:         "SuccessfulStop",
			id:           "550e8400-e29b-41d4-a716-446655440000",
			transitionTo: "Stopped",
			mockSetup: func(commander *domain.MockServiceCommander) {
				// Setup the commander for successful transition
				commander.EXPECT().
					DoAction(mock.Anything, mock.MatchedBy(func(params domain.DoServiceActionParams) bool {
						return params.ID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440000") &&
							params.Action == "stop"
					})).
					Return(&domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						},
						Status: "Stopped",
					}, nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:         "SuccessfulDelete",
			id:           "550e8400-e29b-41d4-a716-446655440000",
			transitionTo: "Deleted",
			mockSetup: func(commander *domain.MockServiceCommander) {
				// Setup the commander for successful transition
				commander.EXPECT().
					DoAction(mock.Anything, mock.MatchedBy(func(params domain.DoServiceActionParams) bool {
						return params.ID == uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
					})).
					Return(&domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
						},
						Status: "Deleted",
					}, nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:         "InvalidStatusTransition",
			id:           "550e8400-e29b-41d4-a716-446655440000",
			transitionTo: "Started",
			mockSetup: func(commander *domain.MockServiceCommander) {
				// Setup the commander to return an error for invalid transition
				commander.EXPECT().
					DoAction(mock.Anything, mock.Anything).
					Return(nil, domain.NewInvalidInputErrorf("invalid status transition"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "ServiceNotFound",
			id:           "550e8400-e29b-41d4-a716-446655440000",
			transitionTo: "Started",
			mockSetup: func(commander *domain.MockServiceCommander) {
				// Setup the commander to return not found
				commander.EXPECT().
					DoAction(mock.Anything, mock.Anything).
					Return(nil, domain.NewNotFoundErrorf("service not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			serviceQuerier := domain.NewMockServiceQuerier(t)
			agentQuerier := domain.NewMockAgentQuerier(t)
			serviceGroupQuerier := domain.NewMockServiceGroupQuerier(t)
			commander := domain.NewMockServiceCommander(t)
			authz := authz.NewMockAuthorizer(t) // Not used in handler tests
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
			authIdentity := newMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.ID(CommandWithoutBody(func(ctx context.Context, id properties.UUID) error {
				// All transitions now use DoAction except delete
				switch tc.transitionTo {
				case "Deleted":
					return handler.Delete(ctx, id)
				case "Started":
					params := domain.DoServiceActionParams{ID: id, Action: "start"}
					_, err := commander.DoAction(ctx, params)
					return err
				case "Stopped":
					params := domain.DoServiceActionParams{ID: id, Action: "stop"}
					_, err := commander.DoAction(ctx, params)
					return err
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

// TestServicePropertyValidation tests property validation in service operations
func TestServicePropertyValidation(t *testing.T) {
	testCases := []struct {
		name           string
		operation      string // "create" or "update"
		serviceID      string // only for update
		request        any
		mockSetup      func(commander *domain.MockServiceCommander)
		expectedStatus int
		checkError     func(t *testing.T, errorText string)
	}{
		{
			name:      "UpdateWithMutableProperty",
			operation: "update",
			serviceID: "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceReq{
				Properties: helpers.JSONPtr(properties.JSON{"instanceName": "new-name"}),
			},
			mockSetup: func(commander *domain.MockServiceCommander) {
				commander.EXPECT().
					Update(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, params domain.UpdateServiceParams) (*domain.Service, error) {
						return &domain.Service{
							BaseEntity: domain.BaseEntity{
								ID: params.ID,
							},
							Properties: params.Properties,
						}, nil
					})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "UpdateWithImmutableProperty",
			operation: "update",
			serviceID: "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceReq{
				Properties: helpers.JSONPtr(properties.JSON{"uuid": "new-uuid"}),
			},
			mockSetup: func(commander *domain.MockServiceCommander) {
				commander.EXPECT().
					Update(mock.Anything, mock.Anything).
					Return(nil, domain.NewInvalidInputErrorf("property 'uuid' cannot be updated (updatable: never)"))
			},
			expectedStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, errorText string) {
				assert.Contains(t, errorText, "uuid")
				assert.Contains(t, errorText, "cannot be updated")
				assert.Contains(t, errorText, "never")
			},
		},
		{
			name:      "UpdateWithWrongStateProperty",
			operation: "update",
			serviceID: "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceReq{
				Properties: helpers.JSONPtr(properties.JSON{"diskSize": 500}),
			},
			mockSetup: func(commander *domain.MockServiceCommander) {
				commander.EXPECT().
					Update(mock.Anything, mock.Anything).
					Return(nil, domain.NewInvalidInputErrorf("property 'diskSize' cannot be updated in status 'Started' (allowed statuses: [Stopped])"))
			},
			expectedStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, errorText string) {
				assert.Contains(t, errorText, "diskSize")
				assert.Contains(t, errorText, "cannot be updated in status")
				assert.Contains(t, errorText, "Stopped")
			},
		},
		{
			name:      "UpdateWithAgentProperty",
			operation: "update",
			serviceID: "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceReq{
				Properties: helpers.JSONPtr(properties.JSON{"ipAddress": "192.168.1.100"}),
			},
			mockSetup: func(commander *domain.MockServiceCommander) {
				commander.EXPECT().
					Update(mock.Anything, mock.Anything).
					Return(nil, domain.NewInvalidInputErrorf("ipAddress: property can only be set by: [agent]"))
			},
			expectedStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, errorText string) {
				assert.Contains(t, errorText, "ipAddress")
				assert.Contains(t, errorText, "can only be set by")
				assert.Contains(t, errorText, "agent")
			},
		},
		{
			name:      "CreateWithAgentProperty",
			operation: "create",
			request: CreateServiceReq{
				Name:          "Test Service",
				AgentID:       &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				GroupID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
				ServiceTypeID: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
				Properties:    properties.JSON{"ipAddress": "192.168.1.100"},
			},
			mockSetup: func(commander *domain.MockServiceCommander) {
				commander.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(nil, domain.NewInvalidInputErrorf("ipAddress: property can only be set by: [agent]"))
			},
			expectedStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, errorText string) {
				assert.Contains(t, errorText, "ipAddress")
				assert.Contains(t, errorText, "can only be set by")
				assert.Contains(t, errorText, "agent")
			},
		},
		{
			name:      "CreateWithMutableProperties",
			operation: "create",
			request: CreateServiceReq{
				Name:          "Test Service",
				AgentID:       &[]properties.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				GroupID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
				ServiceTypeID: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
				Properties:    properties.JSON{"instanceName": "my-instance"},
			},
			mockSetup: func(commander *domain.MockServiceCommander) {
				commander.EXPECT().
					Create(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, params domain.CreateServiceParams) (*domain.Service, error) {
						return &domain.Service{
							BaseEntity: domain.BaseEntity{
								ID: uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
							},
							Properties: &params.Properties,
						}, nil
					})
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			serviceQuerier := domain.NewMockServiceQuerier(t)
			agentQuerier := domain.NewMockAgentQuerier(t)
			serviceGroupQuerier := domain.NewMockServiceGroupQuerier(t)
			commander := domain.NewMockServiceCommander(t)
			authz := authz.NewMockAuthorizer(t)
			tc.mockSetup(commander)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			var req *http.Request
			var middlewareHandler http.Handler

			if tc.operation == "update" {
				// Create update request
				updateReq := tc.request.(UpdateServiceReq)
				bodyBytes, err := json.Marshal(updateReq)
				require.NoError(t, err)
				req = httptest.NewRequest("PATCH", "/services/"+tc.serviceID, bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				// Set up chi router context for URL parameters
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", tc.serviceID)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

				// Add auth identity
				authIdentity := newMockAuthAdmin()
				req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

				// Setup middleware chain
				middlewareHandler = middlewares.DecodeBody[UpdateServiceReq]()(middlewares.ID(Update(handler.Update, ServiceToRes)))
			} else {
				// Create create request
				createReq := tc.request.(CreateServiceReq)
				bodyBytes, err := json.Marshal(createReq)
				require.NoError(t, err)
				req = httptest.NewRequest("POST", "/services", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				// Add auth identity
				authIdentity := newMockAuthAdmin()
				req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

				// Setup middleware chain
				middlewareHandler = middlewares.DecodeBody[CreateServiceReq]()(http.HandlerFunc(handler.Create))
			}

			// Execute request
			w := httptest.NewRecorder()
			middlewareHandler.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tc.expectedStatus, w.Code)

			// Check error message if expected
			if tc.checkError != nil {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				errorText, ok := response["error"].(string)
				require.True(t, ok, "Error message should be present")
				tc.checkError(t, errorText)
			}
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

	agentInstanceID := "ext-123"
	props := properties.JSON{"key": "value"}
	resources := properties.JSON{"cpu": "1", "memory": "2GB"}

	service := &domain.Service{
		BaseEntity: domain.BaseEntity{
			ID:        serviceID,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:              "Test Service",
		AgentID:           agentID,
		ServiceTypeID:     serviceTypeID,
		GroupID:           groupID,
		ConsumerID:        consumerID,
		ProviderID:        providerID,
		AgentInstanceID:   &agentInstanceID,
		Status:            "New",
		Properties:        &props,
		AgentInstanceData: &resources,
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
	assert.Equal(t, agentInstanceID, *response.AgentInstanceID)
	assert.Equal(t, "New", response.Status)
	assert.Equal(t, props, *response.Properties)
	assert.Equal(t, resources, *response.AgentInstanceData)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
}
