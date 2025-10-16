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

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJobHandleGetPendingJobs tests the handleGetPendingJobs method
func TestJobHandleGetPendingJobs(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return pending jobs
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				agentID := uuid.MustParse("850e8400-e29b-41d4-a716-446655440000")

				querier.getPendingJobsForAgentFunc = func(ctx context.Context, requestedAgentID properties.UUID, limit int) ([]*domain.Job, error) {
					return []*domain.Job{
						{
							BaseEntity: domain.BaseEntity{
								ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
								CreatedAt: createdAt,
								UpdatedAt: updatedAt,
							},
							ProviderID: uuid.MustParse("650e8400-e29b-41d4-a716-446655440000"),
							ConsumerID: uuid.MustParse("750e8400-e29b-41d4-a716-446655440000"),
							AgentID:    agentID,
							ServiceID:  uuid.MustParse("950e8400-e29b-41d4-a716-446655440000"),
							Action:     "create",
							Status:     domain.JobPending,
							Priority:   1,
						},
						{
							BaseEntity: domain.BaseEntity{
								ID:        uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
								CreatedAt: createdAt,
								UpdatedAt: updatedAt,
							},
							ProviderID: uuid.MustParse("650e8400-e29b-41d4-a716-446655440000"),
							ConsumerID: uuid.MustParse("750e8400-e29b-41d4-a716-446655440000"),
							AgentID:    agentID,
							ServiceID:  uuid.MustParse("950e8400-e29b-41d4-a716-446655440000"),
							Action:     "delete",
							Status:     domain.JobPending,
							Priority:   2,
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockJobQuerier{}
			commander := &mockJobCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewJobHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/jobs/pending?limit=10", nil)

			// Create agent identity
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.Pending(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response []any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, 2, len(response))
			}
		})
	}
}

// TestJobHandleClaimJob tests the handleClaimJob method
func TestJobHandleClaimJob(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.AuthScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				commander.claimFunc = func(ctx context.Context, jobID properties.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "ClaimError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.AuthScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				commander.claimFunc = func(ctx context.Context, jobID properties.UUID) error {
					return domain.NewInvalidInputErrorf("job already claimed")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockJobQuerier{}
			commander := &mockJobCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewJobHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("POST", "/jobs/"+tc.id+"/claim", nil)

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.ID(CommandWithoutBody(handler.commander.Claim))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestJobHandleCompleteJob tests the handleCompleteJob method
func TestJobHandleCompleteJob(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		requestBody    string
		mockSetup      func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"agentData": {"cpu": 2, "memory": 4},
				"agentInstanceID": "ext-123"
			}`,
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.AuthScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				commander.completeFunc = func(ctx context.Context, params domain.CompleteJobParams) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "CompleteError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"agentData": {"cpu": 2, "memory": 4},
				"agentInstanceID": "ext-123"
			}`,
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.AuthScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				commander.completeFunc = func(ctx context.Context, params domain.CompleteJobParams) error {
					return domain.NewInvalidInputErrorf("job already completed")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "SuccessWithProperties",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"agentData": {"cpu": 2, "memory": 4},
				"agentInstanceID": "ext-123",
				"properties": {"ipAddress": "192.168.1.100", "port": 8080}
			}`,
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.AuthScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				commander.completeFunc = func(ctx context.Context, params domain.CompleteJobParams) error {
					// Verify properties were passed correctly
					assert.NotNil(t, params.Properties)
					assert.Equal(t, "192.168.1.100", params.Properties["ipAddress"])
					assert.Equal(t, float64(8080), params.Properties["port"])
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "PropertyValidationError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"agentData": {"cpu": 2, "memory": 4},
				"agentInstanceID": "ext-123",
				"properties": {"instanceName": "new-name"}
			}`,
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.AuthScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				commander.completeFunc = func(ctx context.Context, params domain.CompleteJobParams) error {
					// Simulate validation error for user-source property
					return domain.NewInvalidInputErrorf("property 'instanceName' cannot be updated by agent (source: input)")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "SuccessWithoutProperties",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"agentData": {"cpu": 2, "memory": 4},
				"agentInstanceID": "ext-123"
			}`,
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.AuthScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				commander.completeFunc = func(ctx context.Context, params domain.CompleteJobParams) error {
					// Verify properties is nil/empty when not provided
					assert.Nil(t, params.Properties)
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockJobQuerier{}
			commander := &mockJobCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewJobHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("POST", "/jobs/"+tc.id+"/complete", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[CompleteJobReq]()(middlewares.ID(Command(handler.Complete)))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestJobHandleFailJob tests the handleFailJob method
func TestJobHandleFailJob(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		requestBody    string
		mockSetup      func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"errorMessage": "Resource allocation failed"
			}`,
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.AuthScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				commander.failFunc = func(ctx context.Context, params domain.FailJobParams) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "FailError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"errorMessage": "Resource allocation failed"
			}`,
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.AuthScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				commander.failFunc = func(ctx context.Context, params domain.FailJobParams) error {
					return domain.NewInvalidInputErrorf("job already failed")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockJobQuerier{}
			commander := &mockJobCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewJobHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("POST", "/jobs/"+tc.id+"/fail", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[FailJobReq]()(middlewares.ID(Command(handler.Fail)))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestJobToResponse tests the jobToResponse function
func TestJobToResponse(t *testing.T) {
	// Setup test
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	claimedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	job := &domain.Job{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		ProviderID:   uuid.MustParse("650e8400-e29b-41d4-a716-446655440000"),
		ConsumerID:   uuid.MustParse("750e8400-e29b-41d4-a716-446655440000"),
		AgentID:      uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
		ServiceID:    uuid.MustParse("950e8400-e29b-41d4-a716-446655440000"),
		Action:       "create",
		Status:       domain.JobProcessing,
		Priority:     1,
		ClaimedAt:    &claimedAt,
		ErrorMessage: "",
	}

	// Execute
	response := JobToRes(job)

	// Assert
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", response.ID.String())
	assert.Equal(t, "650e8400-e29b-41d4-a716-446655440000", response.ProviderID.String())
	assert.Equal(t, "750e8400-e29b-41d4-a716-446655440000", response.ConsumerID.String())
	assert.Equal(t, "850e8400-e29b-41d4-a716-446655440000", response.AgentID.String())
	assert.Equal(t, "950e8400-e29b-41d4-a716-446655440000", response.ServiceID.String())
	assert.Equal(t, "create", response.Action)
	assert.Equal(t, domain.JobProcessing, response.Status)
	assert.Equal(t, 1, response.Priority)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
	assert.Equal(t, (*JSONUTCTime)(&claimedAt), response.ClaimedAt)
	assert.Nil(t, response.CompletedAt)
}

// TestNewJobHandler tests the NewJobHandler function
func TestNewJobHandler(t *testing.T) {
	// Create mocks
	querier := &mockJobQuerier{}
	commander := &mockJobCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Execute
	handler := NewJobHandler(querier, commander, authz)

	// Assert
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestJobHandlerRoutes tests the Routes function
func TestJobHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockJobQuerier{}
	commander := &mockJobCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewJobHandler(querier, commander, authz)

	// Execute
	routeFunc := handler.Routes()
	assert.NotNil(t, routeFunc)

	// Create a chi router and apply the routes
	r := chi.NewRouter()
	routeFunc(r)

	// Assert that endpoints are registered
	// We can't directly test chi router internals, but we can check
	// that the router has registered handlers for the expected patterns
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		// Check expected routes exist - we can't access exact handler functions,
		// but we can verify the routes are registered
		switch {
		case method == "GET" && route == "/":
		case method == "GET" && route == "/{id}":
		case method == "GET" && route == "/pending":
		case method == "POST" && route == "/{id}/claim":
		case method == "POST" && route == "/{id}/complete":
		case method == "POST" && route == "/{id}/fail":
		case method == "POST" && route == "/{id}/unsupported":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}
