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

// TestJobHandleList tests the handleList method (pure business logic)
func TestJobHandleList(t *testing.T) {
	testCases := []struct {
		name           string
		queryParams    string
		mockSetup      func(querier *mockJobQuerier)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "Success",
			queryParams: "?page=1&pageSize=10",
			mockSetup: func(querier *mockJobQuerier) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Job], error) {
					return &domain.PageResponse[domain.Job]{
						Items: []domain.Job{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								ProviderID: uuid.MustParse("650e8400-e29b-41d4-a716-446655440000"),
								ConsumerID: uuid.MustParse("750e8400-e29b-41d4-a716-446655440000"),
								AgentID:    uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
								ServiceID:  uuid.MustParse("950e8400-e29b-41d4-a716-446655440000"),
								Action:     domain.ServiceActionCreate,
								State:      domain.JobPending,
								Priority:   1,
							},
						},
						TotalItems:  1,
						CurrentPage: 1,
						TotalPages:  1,
						HasNext:     false,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"currentPage": float64(1),
				"totalItems":  float64(1),
				"totalPages":  float64(1),
				"hasNext":     false,
				"hasPrev":     false,
			},
		},
		{
			name:        "EmptyResult",
			queryParams: "?page=1&pageSize=10",
			mockSetup: func(querier *mockJobQuerier) {
				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Job], error) {
					return &domain.PageResponse[domain.Job]{
						Items:       []domain.Job{},
						TotalItems:  0,
						CurrentPage: 1,
						TotalPages:  0,
						HasNext:     false,
						HasPrev:     false,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"items":       []interface{}{},
				"totalItems":  float64(0),
				"totalPages":  float64(0),
				"currentPage": float64(1),
				"hasNext":     false,
				"hasPrev":     false,
			},
		},
		{
			name:        "ListError",
			queryParams: "?page=1&pageSize=10",
			mockSetup: func(querier *mockJobQuerier) {
				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Job], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockJobQuerier{}
			commander := &mockJobCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier)

			// Create the handler
			handler := NewJobHandler(querier, commander, authz)

			// Create request with simulated middleware context
			req := httptest.NewRequest("GET", "/jobs"+tc.queryParams, nil)

			// Add auth identity to context (required by all handlers)
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

				for key, expectedValue := range tc.expectedBody {
					assert.Equal(t, expectedValue, response[key])
				}

				if tc.name == "Success" {
					items := response["items"].([]interface{})
					assert.Equal(t, 1, len(items))

					firstItem := items[0].(map[string]interface{})
					assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", firstItem["id"])
					assert.Equal(t, "ServiceCreate", firstItem["action"])
					assert.Equal(t, "Pending", firstItem["state"])
				}
			}
		})
	}
}

// TestJobHandleGet tests the handleGet method
func TestJobHandleGet(t *testing.T) {
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

				// Setup the mock to return a test job
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Job, error) {
					return &domain.Job{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						ProviderID: uuid.MustParse("650e8400-e29b-41d4-a716-446655440000"),
						ConsumerID: uuid.MustParse("750e8400-e29b-41d4-a716-446655440000"),
						AgentID:    uuid.MustParse("850e8400-e29b-41d4-a716-446655440000"),
						ServiceID:  uuid.MustParse("950e8400-e29b-41d4-a716-446655440000"),
						Action:     domain.ServiceActionCreate,
						State:      domain.JobPending,
						Priority:   1,
					}, nil
				}

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
					return &domain.EmptyAuthTargetScope, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Job, error) {
					return nil, domain.NewNotFoundErrorf("job not found")
				}

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
					return &domain.EmptyAuthTargetScope, nil
				}
			},
			expectedStatus: http.StatusNotFound,
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
			req := httptest.NewRequest("GET", "/jobs/"+tc.id, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)

			// We need to add the UUID to the context directly since we're not using the middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthFulcrumAdmin()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleGet(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

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

				querier.getPendingJobsForAgentFunc = func(ctx context.Context, requestedAgentID domain.UUID, limit int) ([]*domain.Job, error) {
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
							Action:     domain.ServiceActionCreate,
							State:      domain.JobPending,
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
							Action:     domain.ServiceActionDelete,
							State:      domain.JobPending,
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
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleGetPendingJobs(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response []interface{}
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

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
					return &domain.EmptyAuthTargetScope, nil
				}

				commander.claimFunc = func(ctx context.Context, jobID domain.UUID) error {
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

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
					return &domain.EmptyAuthTargetScope, nil
				}

				commander.claimFunc = func(ctx context.Context, jobID domain.UUID) error {
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
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)

			// We need to add the UUID to the context directly since we're not using the middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleClaimJob(w, req)

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
				"resources": {"cpu": 2, "memory": 4},
				"externalID": "ext-123"
			}`,
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
					return &domain.EmptyAuthTargetScope, nil
				}

				commander.completeFunc = func(ctx context.Context, jobID domain.UUID, resources *domain.JSON, externalID *string) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "CompleteError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"resources": {"cpu": 2, "memory": 4},
				"externalID": "ext-123"
			}`,
			mockSetup: func(querier *mockJobQuerier, commander *mockJobCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
					return &domain.EmptyAuthTargetScope, nil
				}

				commander.completeFunc = func(ctx context.Context, jobID domain.UUID, resources *domain.JSON, externalID *string) error {
					return domain.NewInvalidInputErrorf("job already completed")
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
			req := httptest.NewRequest("POST", "/jobs/"+tc.id+"/complete", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)

			// We need to add the UUID to the context directly since we're not using the middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Simulate body middleware for tests that need it
			var body CompleteJobRequest
			json.Unmarshal([]byte(tc.requestBody), &body)
			req = req.WithContext(context.WithValue(req.Context(), decodedBodyContextKey, body))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleCompleteJob(w, req)

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

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
					return &domain.EmptyAuthTargetScope, nil
				}

				commander.failFunc = func(ctx context.Context, jobID domain.UUID, errorMessage string) error {
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

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
					return &domain.EmptyAuthTargetScope, nil
				}

				commander.failFunc = func(ctx context.Context, jobID domain.UUID, errorMessage string) error {
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
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)

			// We need to add the UUID to the context directly since we're not using the middleware
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Simulate body middleware for tests that need it
			var body FailJobRequest
			json.Unmarshal([]byte(tc.requestBody), &body)
			req = req.WithContext(context.WithValue(req.Context(), decodedBodyContextKey, body))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleFailJob(w, req)

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
		Action:       domain.ServiceActionCreate,
		State:        domain.JobProcessing,
		Priority:     1,
		ClaimedAt:    &claimedAt,
		ErrorMessage: "",
	}

	// Execute
	response := jobToResponse(job)

	// Assert
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", response.ID.String())
	assert.Equal(t, "650e8400-e29b-41d4-a716-446655440000", response.ProviderID.String())
	assert.Equal(t, "750e8400-e29b-41d4-a716-446655440000", response.ConsumerID.String())
	assert.Equal(t, "850e8400-e29b-41d4-a716-446655440000", response.AgentID.String())
	assert.Equal(t, "950e8400-e29b-41d4-a716-446655440000", response.ServiceID.String())
	assert.Equal(t, domain.ServiceActionCreate, response.Action)
	assert.Equal(t, domain.JobProcessing, response.State)
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
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}
