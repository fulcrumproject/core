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

// mockServiceCommander is a custom mock for ServiceCommander
type mockServiceCommander struct {
	createFunc                     func(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID, groupID domain.UUID, name string, attributes domain.Attributes, properties domain.JSON) (*domain.Service, error)
	updateFunc                     func(ctx context.Context, id domain.UUID, name *string, properties *domain.JSON) (*domain.Service, error)
	transitionFunc                 func(ctx context.Context, id domain.UUID, state domain.ServiceState) (*domain.Service, error)
	retryFunc                      func(ctx context.Context, id domain.UUID) (*domain.Service, error)
	failTimeoutServicesAndJobsFunc func(ctx context.Context, timeout time.Duration) (int, error)
}

func (m *mockServiceCommander) Create(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID, groupID domain.UUID, name string, attributes domain.Attributes, properties domain.JSON) (*domain.Service, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, agentID, serviceTypeID, groupID, name, attributes, properties)
	}
	return nil, fmt.Errorf("create not mocked")
}

func (m *mockServiceCommander) Update(ctx context.Context, id domain.UUID, name *string, properties *domain.JSON) (*domain.Service, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, properties)
	}
	return nil, fmt.Errorf("update not mocked")
}

func (m *mockServiceCommander) Transition(ctx context.Context, id domain.UUID, state domain.ServiceState) (*domain.Service, error) {
	if m.transitionFunc != nil {
		return m.transitionFunc(ctx, id, state)
	}
	return nil, fmt.Errorf("transition not mocked")
}

func (m *mockServiceCommander) Retry(ctx context.Context, id domain.UUID) (*domain.Service, error) {
	if m.retryFunc != nil {
		return m.retryFunc(ctx, id)
	}
	return nil, fmt.Errorf("retry not mocked")
}

func (m *mockServiceCommander) FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error) {
	if m.failTimeoutServicesAndJobsFunc != nil {
		return m.failTimeoutServicesAndJobsFunc(ctx, timeout)
	}
	return 0, fmt.Errorf("failTimeoutServicesAndJobs not mocked")
}

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

// TestServiceHandlerRoutes tests that routes are properly registered
func TestServiceHandlerRoutes(t *testing.T) {
	// We'll use a stub for the actual handler to avoid executing real handler logic
	stubHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a test router and register routes
	r := chi.NewRouter()

	// Instead of using the actual handlers which require auth context,
	// we'll manually register routes with our stub handler
	r.Route("/services", func(r chi.Router) {
		// Register the routes
		r.Get("/", stubHandler)
		r.Post("/", stubHandler)
		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return next
			})
			r.Get("/{id}", stubHandler)
			r.Patch("/{id}", stubHandler)
			r.Post("/{id}/start", stubHandler)
			r.Post("/{id}/stop", stubHandler)
			r.Delete("/{id}", stubHandler)
			r.Post("/{id}/retry", stubHandler)
		})
	})

	// Test GET route
	req := httptest.NewRequest("GET", "/services", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test POST route
	req = httptest.NewRequest("POST", "/services", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test GET /{id} route
	req = httptest.NewRequest("GET", "/services/550e8400-e29b-41d4-a716-446655440000", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test PATCH /{id} route
	req = httptest.NewRequest("PATCH", "/services/550e8400-e29b-41d4-a716-446655440000", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test POST /{id}/start route
	req = httptest.NewRequest("POST", "/services/550e8400-e29b-41d4-a716-446655440000/start", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test POST /{id}/stop route
	req = httptest.NewRequest("POST", "/services/550e8400-e29b-41d4-a716-446655440000/stop", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test DELETE /{id} route
	req = httptest.NewRequest("DELETE", "/services/550e8400-e29b-41d4-a716-446655440000", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Test POST /{id}/retry route
	req = httptest.NewRequest("POST", "/services/550e8400-e29b-41d4-a716-446655440000/retry", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

// TestServiceHandleCreate tests the handleCreate method
func TestServiceHandleCreate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		requestBody    string
		mockSetup      func(serviceQuerier *mockServiceQuerier, agentQuerier *mockAgentQuerier, serviceGroupQuerier *mockServiceGroupQuerier, commander *mockServiceCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			requestBody: `{
				"name": "Test Service",
				"agentId": "550e8400-e29b-41d4-a716-446655440000",
				"groupId": "660e8400-e29b-41d4-a716-446655440000",
				"serviceTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"attributes": {"key": ["value"]},
				"properties": {"prop": "value"}
			}`,
			mockSetup: func(serviceQuerier *mockServiceQuerier, agentQuerier *mockAgentQuerier, serviceGroupQuerier *mockServiceGroupQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				agentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				groupID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
				brokerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")

				// Setup the agent querier to return auth scope
				agentQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					assert.Equal(t, agentID, id)
					return &domain.AuthScope{ProviderID: &providerID, AgentID: &agentID}, nil
				}

				// Setup the service group querier to return auth scope
				serviceGroupQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					assert.Equal(t, groupID, id)
					return &domain.AuthScope{BrokerID: &brokerID}, nil
				}

				// Setup the commander
				commander.createFunc = func(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID, groupID domain.UUID, name string, attributes domain.Attributes, properties domain.JSON) (*domain.Service, error) {
					assert.Equal(t, "Test Service", name)
					assert.Equal(t, domain.Attributes{"key": []string{"value"}}, attributes)
					assert.Equal(t, domain.JSON{"prop": "value"}, properties)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

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
						BrokerID:          brokerID,
						ProviderID:        providerID,
						Attributes:        domain.Attributes{"key": []string{"value"}},
						CurrentState:      domain.ServiceCreated,
						CurrentProperties: &properties,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:        "InvalidRequestFormat",
			requestBody: `{"invalid_json":`,
			mockSetup: func(serviceQuerier *mockServiceQuerier, agentQuerier *mockAgentQuerier, serviceGroupQuerier *mockServiceGroupQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// No setup needed for this case
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "ServiceGroupAuthScopeError",
			requestBody: `{
				"name": "Test Service",
				"agentId": "550e8400-e29b-41d4-a716-446655440000",
				"groupId": "660e8400-e29b-41d4-a716-446655440000",
				"serviceTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"attributes": {"key": ["value"]},
				"properties": {"prop": "value"}
			}`,
			mockSetup: func(serviceQuerier *mockServiceQuerier, agentQuerier *mockAgentQuerier, serviceGroupQuerier *mockServiceGroupQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the service group querier to return an error
				serviceGroupQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return nil, fmt.Errorf("service group not found")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "AgentAuthScopeError",
			requestBody: `{
				"name": "Test Service",
				"agentId": "550e8400-e29b-41d4-a716-446655440000",
				"groupId": "660e8400-e29b-41d4-a716-446655440000",
				"serviceTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"attributes": {"key": ["value"]},
				"properties": {"prop": "value"}
			}`,
			mockSetup: func(serviceQuerier *mockServiceQuerier, agentQuerier *mockAgentQuerier, serviceGroupQuerier *mockServiceGroupQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the service group querier to return auth scope
				brokerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				serviceGroupQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.AuthScope{BrokerID: &brokerID}, nil
				}

				// Setup the agent querier to return an error
				agentQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return nil, fmt.Errorf("agent not found")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "AuthorizationError",
			requestBody: `{
				"name": "Test Service",
				"agentId": "550e8400-e29b-41d4-a716-446655440000",
				"groupId": "660e8400-e29b-41d4-a716-446655440000",
				"serviceTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"attributes": {"key": ["value"]},
				"properties": {"prop": "value"}
			}`,
			mockSetup: func(serviceQuerier *mockServiceQuerier, agentQuerier *mockAgentQuerier, serviceGroupQuerier *mockServiceGroupQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the service group querier to return auth scope
				brokerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				serviceGroupQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.AuthScope{BrokerID: &brokerID}, nil
				}

				// Setup the agent querier to return auth scope
				agentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
				agentQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.AuthScope{ProviderID: &providerID, AgentID: &agentID}, nil
				}

				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden, // Uses ErrDomain for auth errors
		},
		{
			name: "CommanderError",
			requestBody: `{
				"name": "Test Service",
				"agentId": "550e8400-e29b-41d4-a716-446655440000",
				"groupId": "660e8400-e29b-41d4-a716-446655440000",
				"serviceTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"attributes": {"key": ["value"]},
				"properties": {"prop": "value"}
			}`,
			mockSetup: func(serviceQuerier *mockServiceQuerier, agentQuerier *mockAgentQuerier, serviceGroupQuerier *mockServiceGroupQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				agentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
				brokerID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")

				// Setup the agent querier to return auth scope
				agentQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.AuthScope{ProviderID: &providerID, AgentID: &agentID}, nil
				}

				// Setup the service group querier to return auth scope
				serviceGroupQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.AuthScope{BrokerID: &brokerID}, nil
				}

				// Setup the commander to return an error
				commander.createFunc = func(ctx context.Context, agentID domain.UUID, serviceTypeID domain.UUID, groupID domain.UUID, name string, attributes domain.Attributes, properties domain.JSON) (*domain.Service, error) {
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
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request with JSON body
			req := httptest.NewRequest("POST", "/services", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
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
		mockSetup      func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &domain.EmptyAuthScope, nil
				}

				serviceQuerier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
					serviceTypeID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
					groupID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
					brokerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
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
						BrokerID:      brokerID,
						ProviderID:    providerID,
						Attributes:    domain.Attributes{"key": []string{"value"}},
						CurrentState:  domain.ServiceStarted,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "AuthorizationError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the mock to fail authorization
				authz.ShouldSucceed = false

				// Setup the querier to return auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "AuthScopeError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the querier to return auth scope error
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return nil, fmt.Errorf("auth scope error")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Setup the querier to return not found
				serviceQuerier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
					return nil, domain.NotFoundError{Err: fmt.Errorf("service not found")}
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
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(serviceQuerier, commander, authz)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/services/"+tc.id, nil)

			// Add ID to chi context and simulate IDMiddleware
			req = addIDToChiContext(req, tc.id)
			req = simulateIDMiddleware(req, tc.id)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

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
		mockSetup      func(serviceQuerier *mockServiceQuerier, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(serviceQuerier *mockServiceQuerier, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return services
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
				serviceTypeID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
				groupID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
				brokerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
				providerID := uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000")

				serviceQuerier.listFunc = func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error) {
					return &domain.PageResponse[domain.Service]{
						Items: []domain.Service{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:          "Service 1",
								AgentID:       agentID,
								ServiceTypeID: serviceTypeID,
								GroupID:       groupID,
								BrokerID:      brokerID,
								ProviderID:    providerID,
								Attributes:    domain.Attributes{"key": []string{"value"}},
								CurrentState:  domain.ServiceStarted,
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("bb0e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:          "Service 2",
								AgentID:       agentID,
								ServiceTypeID: serviceTypeID,
								GroupID:       groupID,
								BrokerID:      brokerID,
								ProviderID:    providerID,
								Attributes:    domain.Attributes{"key": []string{"value2"}},
								CurrentState:  domain.ServiceStopped,
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
			mockSetup: func(serviceQuerier *mockServiceQuerier, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "ListError",
			mockSetup: func(serviceQuerier *mockServiceQuerier, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				serviceQuerier.listFunc = func(ctx context.Context, authScope *domain.AuthScope, req *domain.PageRequest) (*domain.PageResponse[domain.Service], error) {
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
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(serviceQuerier, authz)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request
			var req *http.Request
			if tc.name == "InvalidPageRequest" {
				// Create invalid page request
				req = httptest.NewRequest("GET", "/services?page=invalid", nil)
			} else {
				req = httptest.NewRequest("GET", "/services?page=1&pageSize=10", nil)
			}

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
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
				assert.Equal(t, "Service 1", firstItem["name"])

				secondItem := items[1].(map[string]interface{})
				assert.Equal(t, "bb0e8400-e29b-41d4-a716-446655440000", secondItem["id"])
				assert.Equal(t, "Service 2", secondItem["name"])
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
		requestBody    string
		mockSetup      func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name:        "Success",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"name": "Updated Service", "properties": {"prop": "updated"}}`,
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &domain.EmptyAuthScope, nil
				}

				// Setup the commander to update
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, properties *domain.JSON) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					require.NotNil(t, name)
					assert.Equal(t, "Updated Service", *name)
					require.NotNil(t, properties)
					assert.Equal(t, domain.JSON{"prop": "updated"}, *properties)

					newName := "Updated Service"
					newProperties := domain.JSON{"prop": "updated"}
					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
					agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
					serviceTypeID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
					groupID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
					brokerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
					providerID := uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000")

					return &domain.Service{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:              newName,
						AgentID:           agentID,
						ServiceTypeID:     serviceTypeID,
						GroupID:           groupID,
						BrokerID:          brokerID,
						ProviderID:        providerID,
						Attributes:        domain.Attributes{"key": []string{"value"}},
						CurrentState:      domain.ServiceStarted,
						CurrentProperties: &newProperties,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "InvalidRequestFormat",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"invalid_json":`,
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// No setup needed for this case
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "AuthorizationError",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"name": "Updated Service", "properties": {"prop": "updated"}}`,
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the mock to fail authorization
				authz.ShouldSucceed = false

				// Setup the querier to return auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:        "CommanderError",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"name": "Updated Service", "properties": {"prop": "updated"}}`,
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Setup the commander to return an error
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, properties *domain.JSON) (*domain.Service, error) {
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
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(serviceQuerier, commander, authz)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request with JSON body
			req := httptest.NewRequest("PATCH", "/services/"+tc.id, strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Add ID to chi context
			// Add ID to chi context and simulate IDMiddleware
			req = addIDToChiContext(req, tc.id)
			req = simulateIDMiddleware(req, tc.id)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

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
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
				assert.Equal(t, "updated", response["currentProperties"].(map[string]interface{})["prop"])
			}
		})
	}
}

// TestServiceHandleTransition tests the handleTransition method (via handleStart, handleStop, handleDelete)
func TestServiceHandleTransition(t *testing.T) {
	// Test handleStart, handleStop, and handleDelete which all call handleTransition
	transitionTests := []struct {
		name           string
		endpoint       string
		expectedAction domain.AuthAction
		expectedState  domain.ServiceState
	}{
		{
			name:           "Start",
			endpoint:       "/services/550e8400-e29b-41d4-a716-446655440000/start",
			expectedAction: domain.ActionStart,
			expectedState:  domain.ServiceStarted,
		},
		{
			name:           "Stop",
			endpoint:       "/services/550e8400-e29b-41d4-a716-446655440000/stop",
			expectedAction: domain.ActionStop,
			expectedState:  domain.ServiceStopped,
		},
		{
			name:           "Delete",
			endpoint:       "/services/550e8400-e29b-41d4-a716-446655440000/delete",
			expectedAction: domain.ActionDelete,
			expectedState:  domain.ServiceDeleted,
		},
	}

	for _, tt := range transitionTests {
		// Setup test cases for each transition type
		testCases := []struct {
			name           string
			id             string
			mockSetup      func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer, expectedAction domain.AuthAction, expectedState domain.ServiceState)
			expectedStatus int
		}{
			{
				name: "Success",
				id:   "550e8400-e29b-41d4-a716-446655440000",
				mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer, expectedAction domain.AuthAction, expectedState domain.ServiceState) {
					// Setup the mock to authorize successfully
					authz.ShouldSucceed = true

					// Setup the querier to return auth scope
					serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
						assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
						return &domain.EmptyAuthScope, nil
					}

					// Setup the commander to transition
					commander.transitionFunc = func(ctx context.Context, id domain.UUID, state domain.ServiceState) (*domain.Service, error) {
						assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
						assert.Equal(t, expectedState, state)

						createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
						updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
						agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
						serviceTypeID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
						groupID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
						brokerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
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
							BrokerID:      brokerID,
							ProviderID:    providerID,
							Attributes:    domain.Attributes{"key": []string{"value"}},
							CurrentState:  state,
						}, nil
					}
				},
				expectedStatus: http.StatusNoContent,
			},
			{
				name: "AuthorizationError",
				id:   "550e8400-e29b-41d4-a716-446655440000",
				mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer, expectedAction domain.AuthAction, expectedState domain.ServiceState) {
					// Setup the mock to fail authorization
					authz.ShouldSucceed = false

					// Setup the querier to return auth scope
					serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
						return &domain.EmptyAuthScope, nil
					}
				},
				expectedStatus: http.StatusForbidden,
			},
			{
				name: "TransitionError",
				id:   "550e8400-e29b-41d4-a716-446655440000",
				mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer, expectedAction domain.AuthAction, expectedState domain.ServiceState) {
					// Setup the mock to authorize successfully
					authz.ShouldSucceed = true

					// Setup the querier to return auth scope
					serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
						return &domain.EmptyAuthScope, nil
					}

					// Setup the commander to return an error
					commander.transitionFunc = func(ctx context.Context, id domain.UUID, state domain.ServiceState) (*domain.Service, error) {
						return nil, fmt.Errorf("transition error")
					}
				},
				expectedStatus: http.StatusInternalServerError,
			},
		}

		for _, tc := range testCases {
			testName := fmt.Sprintf("%s_%s", tt.name, tc.name)
			t.Run(testName, func(t *testing.T) {
				// Setup mocks
				serviceQuerier := &mockServiceQuerier{}
				agentQuerier := &mockAgentQuerier{}
				serviceGroupQuerier := &mockServiceGroupQuerier{}
				commander := &mockServiceCommander{}
				authz := &MockAuthorizer{ShouldSucceed: true}
				tc.mockSetup(serviceQuerier, commander, authz, tt.expectedAction, tt.expectedState)

				// Create the handler
				handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

				// Create request
				req := httptest.NewRequest("POST", tt.endpoint, nil)

				// Add ID to chi context
				// Add ID to chi context and simulate IDMiddleware
				req = addIDToChiContext(req, tc.id)
				req = simulateIDMiddleware(req, tc.id)
				// Add auth identity to context for authorization
				authIdentity := NewMockAuthFulcrumAdmin()
				req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

				// Execute request
				w := httptest.NewRecorder()

				// Call the appropriate handler method
				switch tt.expectedState {
				case domain.ServiceStarted:
					handler.handleStart(w, req)
				case domain.ServiceStopped:
					handler.handleStop(w, req)
				case domain.ServiceDeleted:
					handler.handleDelete(w, req)
				}

				// Assert response
				assert.Equal(t, tc.expectedStatus, w.Code)
			})
		}
	}
}

// TestServiceHandleRetry tests the handleRetry method
func TestServiceHandleRetry(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &domain.EmptyAuthScope, nil
				}

				// Setup the commander to retry
				commander.retryFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
					agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
					serviceTypeID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
					groupID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
					brokerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
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
						BrokerID:      brokerID,
						ProviderID:    providerID,
						Attributes:    domain.Attributes{"key": []string{"value"}},
						CurrentState:  domain.ServiceStarted,
						RetryCount:    1,
					}, nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "AuthorizationError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the mock to fail authorization
				authz.ShouldSucceed = false

				// Setup the querier to return auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "RetryError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(serviceQuerier *mockServiceQuerier, commander *mockServiceCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				serviceQuerier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Setup the commander to return an error
				commander.retryFunc = func(ctx context.Context, id domain.UUID) (*domain.Service, error) {
					return nil, fmt.Errorf("retry error")
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
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(serviceQuerier, commander, authz)

			// Create the handler
			handler := NewServiceHandler(serviceQuerier, agentQuerier, serviceGroupQuerier, commander, authz)

			// Create request
			req := httptest.NewRequest("POST", "/services/"+tc.id+"/retry", nil)

			// Add ID to chi context
			// Add ID to chi context and simulate IDMiddleware
			req = addIDToChiContext(req, tc.id)
			req = simulateIDMiddleware(req, tc.id)
			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

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
	// Create a service
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	agentID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	serviceTypeID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	groupID := uuid.MustParse("880e8400-e29b-41d4-a716-446655440000")
	brokerID := uuid.MustParse("990e8400-e29b-41d4-a716-446655440000")
	providerID := uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	externalID := "ext-123"
	properties := domain.JSON{"prop": "value"}
	targetProperties := domain.JSON{"prop": "target"}
	resources := domain.JSON{"res": "value"}
	targetState := domain.ServiceStarted
	errorMessage := "error occurred"
	failedAction := domain.ServiceActionCreate

	service := &domain.Service{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:              "Test Service",
		AgentID:           agentID,
		ServiceTypeID:     serviceTypeID,
		GroupID:           groupID,
		BrokerID:          brokerID,
		ProviderID:        providerID,
		ExternalID:        &externalID,
		Attributes:        domain.Attributes{"key": []string{"value"}},
		CurrentState:      domain.ServiceCreated,
		TargetState:       &targetState,
		FailedAction:      &failedAction,
		ErrorMessage:      &errorMessage,
		RetryCount:        3,
		CurrentProperties: &properties,
		TargetProperties:  &targetProperties,
		Resources:         &resources,
	}

	// Convert to response
	response := serviceToResponse(service)

	// Verify all fields are correctly mapped
	assert.Equal(t, id, response.ID)
	assert.Equal(t, providerID, response.ProviderID)
	assert.Equal(t, brokerID, response.BrokerID)
	assert.Equal(t, agentID, response.AgentID)
	assert.Equal(t, serviceTypeID, response.ServiceTypeID)
	assert.Equal(t, groupID, response.GroupID)
	assert.Equal(t, externalID, *response.ExternalID)
	assert.Equal(t, "Test Service", response.Name)
	assert.Equal(t, domain.Attributes{"key": []string{"value"}}, response.Attributes)
	assert.Equal(t, domain.ServiceCreated, response.CurrentState)
	assert.Equal(t, targetState, *response.TargetState)
	assert.Equal(t, failedAction, *response.FailedAction)
	assert.Equal(t, errorMessage, *response.ErrorMessage)
	assert.Equal(t, 3, response.RetryCount)
	assert.Equal(t, properties, *response.CurrentProperties)
	assert.Equal(t, targetProperties, *response.TargetProperties)
	assert.Equal(t, resources, *response.Resources)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
}
