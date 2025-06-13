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
		request        CreateServiceRequest
		mockSetup      func(commander *mockServiceCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			request: CreateServiceRequest{
				Name:          "Test Service",
				AgentID:       &[]domain.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				GroupID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
				ServiceTypeID: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
				Properties:    domain.JSON{"prop": "value"},
			},
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander for successful creation
				commander.createFunc = func(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID, groupID domain.UUID, name string, properties domain.JSON) (*domain.Service, error) {
					assert.Equal(t, "Test Service", name)
					assert.Equal(t, domain.JSON{"prop": "value"}, properties)

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
						Name:              name,
						AgentID:           agentID,
						ServiceTypeID:     serviceTypeID,
						GroupID:           groupID,
						ConsumerID:        consumerID,
						ProviderID:        providerID,
						CurrentStatus:     domain.ServiceCreated,
						CurrentProperties: &properties,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "CommanderError",
			request: CreateServiceRequest{
				Name:          "Test Service",
				AgentID:       &[]domain.UUID{uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")}[0],
				GroupID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
				ServiceTypeID: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
				Properties:    domain.JSON{"prop": "value"},
			},
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander to return an error
				commander.createFunc = func(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID, groupID domain.UUID, name string, properties domain.JSON) (*domain.Service, error) {
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

			// Create request
			req := httptest.NewRequest("POST", "/services", nil)

			// Simulate middleware by adding decoded body to context
			ctx := context.WithValue(req.Context(), decodedBodyContextKey, tc.request)

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			ctx = domain.WithAuthIdentity(ctx, authIdentity)

			req = req.WithContext(ctx)

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
				assert.Equal(t, "aa0e8400-e29b-41d4-a716-446655440000", response["id"])
				assert.Equal(t, "Test Service", response["name"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
			}
		})
	}
}

// TestServiceHandleGet tests the handleGet method
func TestServiceHandleGet(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockServiceQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockServiceQuerier) {
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
					serviceTypeID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
					groupID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
					consumerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
					providerID := uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000")

					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:          "Test Service",
						AgentID:       agentID,
						ServiceTypeID: serviceTypeID,
						GroupID:       groupID,
						ConsumerID:    consumerID,
						ProviderID:    providerID,
						CurrentStatus: domain.ServiceStarted,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockServiceQuerier) {
				// Setup the querier to return not found
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
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
			tc.mockSetup(serviceQuerier)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/services/"+tc.id, nil)

			// Simulate ID middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			ctx := context.WithValue(req.Context(), uuidContextKey, parsedUUID)

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			ctx = domain.WithAuthIdentity(ctx, authIdentity)

			req = req.WithContext(ctx)

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
				assert.Equal(t, "Test Service", response["name"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
			}
		})
	}
}

// TestServiceHandleList tests the handleList method
func TestServiceHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockServiceQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockServiceQuerier) {
				// Setup the querier for successful list operation
				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error) {
					// Verify pagination
					assert.Equal(t, 1, req.Page) // Default page is 1, not 0
					assert.Equal(t, 10, req.PageSize)

					// Create sample services
					services := []domain.Service{
						{
							BaseEntity: domain.BaseEntity{
								ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
								CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
								UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
							},
							Name:          "Service 1",
							AgentID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
							CurrentStatus: domain.ServiceStarted,
						},
						{
							BaseEntity: domain.BaseEntity{
								ID:        uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
								CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
								UpdatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
							},
							Name:          "Service 2",
							AgentID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
							CurrentStatus: domain.ServiceStopped,
						},
					}

					return &domain.PageResponse[domain.Service]{
						Items:      services,
						TotalItems: 2,
						TotalPages: 1,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "InvalidParametersUseDefaults",
			mockSetup: func(querier *mockServiceQuerier) {
				// Setup the querier to handle default values when invalid params are provided
				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error) {
					// Verify that default values are used
					assert.Equal(t, 1, req.Page)
					assert.Equal(t, 10, req.PageSize)

					// Create sample services - same as success case
					services := []domain.Service{
						{
							BaseEntity: domain.BaseEntity{
								ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
								CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
								UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
							},
							Name:          "Service 1",
							AgentID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
							CurrentStatus: domain.ServiceStarted,
						},
						{
							BaseEntity: domain.BaseEntity{
								ID:        uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
								CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
								UpdatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
							},
							Name:          "Service 2",
							AgentID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
							CurrentStatus: domain.ServiceStopped,
						},
					}

					return &domain.PageResponse[domain.Service]{
						Items:      services,
						TotalItems: 2,
						TotalPages: 1,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "DatabaseError",
			mockSetup: func(querier *mockServiceQuerier) {
				// Setup the querier to return an error
				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error) {
					// Even with an error, we need to verify we got the pagination params first
					assert.Equal(t, 1, req.Page)
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
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
			tc.mockSetup(serviceQuerier)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request
			var req *http.Request
			if tc.name == "InvalidParametersUseDefaults" {
				// Create a request with invalid page query params
				req = httptest.NewRequest("GET", "/services?page=invalid&pageSize=invalid", nil)
			} else {
				// Create a valid request - note: page is 1-based in this system
				req = httptest.NewRequest("GET", "/services?page=1&pageSize=10", nil)
			}

			// Add auth identity to context (always required)
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
				items, ok := response["items"].([]interface{})
				require.True(t, ok)
				assert.Equal(t, 2, len(items))
				assert.Equal(t, float64(2), response["totalItems"])
				assert.Equal(t, float64(1), response["totalPages"])
			}
		})
	}
}

// Helper functions for update tests
func jsonPtr(j domain.JSON) *domain.JSON {
	return &j
}

// TestServiceHandleUpdate tests the handleUpdate method
func TestServiceHandleUpdate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		request        UpdateServiceRequest
		mockSetup      func(commander *mockServiceCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceRequest{
				Name:       stringPtr("Updated Service"),
				Properties: jsonPtr(domain.JSON{"updated": "value"}),
			},
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander for successful update
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, properties *domain.JSON) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					assert.Equal(t, "Updated Service", *name)
					assert.Equal(t, domain.JSON{"updated": "value"}, *properties)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:              "Updated Service",
						CurrentStatus:     domain.ServiceStarted,
						CurrentProperties: properties,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "ValidationError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceRequest{
				Name:       stringPtr("Invalid"),
				Properties: jsonPtr(domain.JSON{"invalid": "data"}),
			},
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander to return a validation error
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, properties *domain.JSON) (*domain.Service, error) {
					return nil, domain.NewInvalidInputErrorf("validation error")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			request: UpdateServiceRequest{
				Name:       stringPtr("Updated Service"),
				Properties: jsonPtr(domain.JSON{"updated": "value"}),
			},
			mockSetup: func(commander *mockServiceCommander) {
				// Setup the commander to return not found
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, properties *domain.JSON) (*domain.Service, error) {
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
			req := httptest.NewRequest("PATCH", "/services/"+tc.id, nil)

			// Simulate ID middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			ctx := context.WithValue(req.Context(), uuidContextKey, parsedUUID)

			// Simulate body decode middleware
			ctx = context.WithValue(ctx, decodedBodyContextKey, tc.request)

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			ctx = domain.WithAuthIdentity(ctx, authIdentity)

			req = req.WithContext(ctx)

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
				assert.Equal(t, "Updated Service", response["name"])
				props, ok := response["currentProperties"].(map[string]interface{})
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
				commander.transitionFunc = func(ctx context.Context, id domain.UUID, status domain.ServiceStatus) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					assert.Equal(t, domain.ServiceStarted, status)
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: id,
						},
						CurrentStatus: domain.ServiceStarting,
						TargetStatus:  &status,
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
				commander.transitionFunc = func(ctx context.Context, id domain.UUID, status domain.ServiceStatus) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					assert.Equal(t, domain.ServiceStopped, status)
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: id,
						},
						CurrentStatus: domain.ServiceStopping,
						TargetStatus:  &status,
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
				commander.transitionFunc = func(ctx context.Context, id domain.UUID, status domain.ServiceStatus) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					assert.Equal(t, domain.ServiceDeleted, status)
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: id,
						},
						CurrentStatus: domain.ServiceDeleting,
						TargetStatus:  &status,
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
				commander.transitionFunc = func(ctx context.Context, id domain.UUID, status domain.ServiceStatus) (*domain.Service, error) {
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
				commander.transitionFunc = func(ctx context.Context, id domain.UUID, status domain.ServiceStatus) (*domain.Service, error) {
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

			// Simulate ID middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			ctx := context.WithValue(req.Context(), uuidContextKey, parsedUUID)

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			ctx = domain.WithAuthIdentity(ctx, authIdentity)

			req = req.WithContext(ctx)

			// Execute request
			w := httptest.NewRecorder()
			handler.handleTransition(w, req, tc.transitionTo)

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
				commander.retryFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID: id,
						},
						CurrentStatus: domain.ServiceCreated,
						RetryCount:    1,
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
				commander.retryFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
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
				commander.retryFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
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

			// Simulate ID middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			ctx := context.WithValue(req.Context(), uuidContextKey, parsedUUID)

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			ctx = domain.WithAuthIdentity(ctx, authIdentity)

			req = req.WithContext(ctx)

			// Execute request
			w := httptest.NewRecorder()
			handler.handleRetry(w, req)

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

	targetStatus := domain.ServiceStarted
	failedAction := domain.ServiceActionStart
	errorMessage := "Failed to start service"
	externalID := "ext-123"
	properties := domain.JSON{"key": "value"}
	resources := domain.JSON{"cpu": "1", "memory": "2GB"}

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
		ExternalID:        &externalID,
		CurrentStatus:     domain.ServiceCreated,
		TargetStatus:      &targetStatus,
		FailedAction:      &failedAction,
		ErrorMessage:      &errorMessage,
		RetryCount:        2,
		CurrentProperties: &properties,
		TargetProperties:  &properties,
		Resources:         &resources,
	}

	// Convert to response
	response := serviceToResponse(service)

	// Verify response
	assert.Equal(t, serviceID, response.ID)
	assert.Equal(t, "Test Service", response.Name)
	assert.Equal(t, agentID, response.AgentID)
	assert.Equal(t, serviceTypeID, response.ServiceTypeID)
	assert.Equal(t, groupID, response.GroupID)
	assert.Equal(t, consumerID, response.ConsumerID)
	assert.Equal(t, providerID, response.ProviderID)
	assert.Equal(t, externalID, *response.ExternalID)
	assert.Equal(t, domain.ServiceCreated, response.CurrentStatus)
	assert.Equal(t, targetStatus, *response.TargetStatus)
	assert.Equal(t, failedAction, *response.FailedAction)
	assert.Equal(t, errorMessage, *response.ErrorMessage)
	assert.Equal(t, 2, response.RetryCount)
	assert.Equal(t, properties, *response.CurrentProperties)
	assert.Equal(t, properties, *response.TargetProperties)
	assert.Equal(t, resources, *response.Resources)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
}
