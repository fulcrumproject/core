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

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/middlewares"
	"github.com/fulcrumproject/commons/properties"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewServiceGroupHandler tests the constructor
func TestNewServiceGroupHandler(t *testing.T) {
	querier := &mockServiceGroupQuerier{}
	commander := &mockServiceGroupCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewServiceGroupHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestServiceGroupHandlerRoutes tests that routes are properly registered
func TestServiceGroupHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockServiceGroupQuerier{}
	commander := &mockServiceGroupCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewServiceGroupHandler(querier, commander, authz)

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

// TestServiceGroupHandleCreate tests the handleCreate method
func TestServiceGroupHandleCreate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		requestBody    CreateServiceGroupRequest
		mockSetup      func(commander *mockServiceGroupCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			requestBody: CreateServiceGroupRequest{
				Name:       "Test Group",
				ConsumerID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			},
			mockSetup: func(commander *mockServiceGroupCommander) {
				consumerID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

				// Setup the commander
				commander.createFunc = func(ctx context.Context, name string, bID properties.UUID) (*domain.ServiceGroup, error) {
					assert.Equal(t, "Test Group", name)
					assert.Equal(t, consumerID, bID)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.ServiceGroup{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       name,
						ConsumerID: bID,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "CommanderError",
			requestBody: CreateServiceGroupRequest{
				Name:       "Test Group",
				ConsumerID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			},
			mockSetup: func(commander *mockServiceGroupCommander) {
				// Setup the commander to return an error
				commander.createFunc = func(ctx context.Context, name string, consumerID properties.UUID) (*domain.ServiceGroup, error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockServiceGroupQuerier{}
			commander := &mockServiceGroupCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(commander)

			// Create the handler
			handler := NewServiceGroupHandler(querier, commander, authz)

			// Create request with body
			bodyBytes, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/service-groups", bytes.NewReader(bodyBytes))
			req = req.WithContext(auth.WithIdentity(req.Context(), NewMockAuthAdmin()))
			req.Header.Set("Content-Type", "application/json")

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[CreateServiceGroupRequest]()(http.HandlerFunc(handler.handleCreate))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusCreated {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, "660e8400-e29b-41d4-a716-446655440000", response["id"])
				assert.Equal(t, "Test Group", response["name"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
			}
		})
	}
}

// TestServiceGroupHandleGet tests the handleGet method
func TestServiceGroupHandleGet(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockServiceGroupQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockServiceGroupQuerier) {
				querier.findByIDFunc = func(ctx context.Context, id properties.UUID) (*domain.ServiceGroup, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					consumerID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")

					return &domain.ServiceGroup{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       "Test Group",
						ConsumerID: consumerID,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockServiceGroupQuerier) {
				// Setup the querier to return not found
				querier.findByIDFunc = func(ctx context.Context, id properties.UUID) (*domain.ServiceGroup, error) {
					return nil, domain.NotFoundError{Err: fmt.Errorf("service group not found")}
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockServiceGroupQuerier{}
			commander := &mockServiceGroupCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier)

			// Create the handler
			handler := NewServiceGroupHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/service-groups/"+tc.id, nil)

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()

			middlewareHandler := middlewares.ID(http.HandlerFunc(handler.handleGet))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, tc.id, response["id"])
				assert.Equal(t, "Test Group", response["name"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
			}
		})
	}
}

// TestServiceGroupHandleList tests the handleList method
func TestServiceGroupHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockServiceGroupQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockServiceGroupQuerier) {
				// Setup the mock to return service groups
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				consumerID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")

				querier.listFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceGroup], error) {
					return &domain.PageResponse[domain.ServiceGroup]{
						Items: []domain.ServiceGroup{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:       "Group 1",
								ConsumerID: consumerID,
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:       "Group 2",
								ConsumerID: consumerID,
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
			mockSetup: func(querier *mockServiceGroupQuerier) {
				querier.listFunc = func(ctx context.Context, authScope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.ServiceGroup], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockServiceGroupQuerier{}
			commander := &mockServiceGroupCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier)

			// Create the handler
			handler := NewServiceGroupHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/service-groups?page=1&pageSize=10", nil)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleList(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
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
				assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", firstItem["id"])
				assert.Equal(t, "Group 1", firstItem["name"])

				secondItem := items[1].(map[string]any)
				assert.Equal(t, "660e8400-e29b-41d4-a716-446655440000", secondItem["id"])
				assert.Equal(t, "Group 2", secondItem["name"])
			}
		})
	}
}

// TestServiceGroupHandleUpdate tests the handleUpdate method
func TestServiceGroupHandleUpdate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		requestBody    UpdateServiceGroupRequest
		mockSetup      func(commander *mockServiceGroupCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: UpdateServiceGroupRequest{
				Name: stringPtr("Updated Group"),
			},
			mockSetup: func(commander *mockServiceGroupCommander) {
				// Setup the commander to update
				commander.updateFunc = func(ctx context.Context, id properties.UUID, name *string) (*domain.ServiceGroup, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					require.NotNil(t, name)
					assert.Equal(t, "Updated Group", *name)

					newName := "Updated Group"
					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
					consumerID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")

					return &domain.ServiceGroup{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       newName,
						ConsumerID: consumerID,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "CommanderError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: UpdateServiceGroupRequest{
				Name: stringPtr("Updated Group"),
			},
			mockSetup: func(commander *mockServiceGroupCommander) {
				// Setup the commander to return an error
				commander.updateFunc = func(ctx context.Context, id properties.UUID, name *string) (*domain.ServiceGroup, error) {
					return nil, fmt.Errorf("update error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockServiceGroupQuerier{}
			commander := &mockServiceGroupCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(commander)

			// Create the handler
			handler := NewServiceGroupHandler(querier, commander, authz)

			// Create request with body
			bodyBytes, err := json.Marshal(tc.requestBody)
			require.NoError(t, err)
			req := httptest.NewRequest("PATCH", "/service-groups/"+tc.id, bytes.NewReader(bodyBytes))
			req = req.WithContext(auth.WithIdentity(req.Context(), NewMockAuthAdmin()))
			req.Header.Set("Content-Type", "application/json")

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[UpdateServiceGroupRequest]()(middlewares.ID(http.HandlerFunc(handler.handleUpdate)))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, tc.id, response["id"])
				assert.Equal(t, "Updated Group", response["name"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
			}
		})
	}
}

// TestServiceGroupHandleDelete tests the handleDelete method
func TestServiceGroupHandleDelete(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(commander *mockServiceGroupCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(commander *mockServiceGroupCommander) {
				// Setup the commander to delete
				commander.deleteFunc = func(ctx context.Context, id properties.UUID) error {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "DeleteError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(commander *mockServiceGroupCommander) {
				// Setup the commander to return an error
				commander.deleteFunc = func(ctx context.Context, id properties.UUID) error {
					return fmt.Errorf("delete error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockServiceGroupQuerier{}
			commander := &mockServiceGroupCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(commander)

			// Create the handler
			handler := NewServiceGroupHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("DELETE", "/service-groups/"+tc.id, nil)

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.ID(http.HandlerFunc(handler.handleDelete))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestServiceGroupToResponse tests the serviceGroupToResponse function
func TestServiceGroupToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	consumerID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")

	// Create a service group
	serviceGroup := &domain.ServiceGroup{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:       "Test Group",
		ConsumerID: consumerID,
	}

	response := serviceGroupToResponse(serviceGroup)

	// Verify all fields are correctly mapped
	assert.Equal(t, serviceGroup.ID, response.ID)
	assert.Equal(t, serviceGroup.Name, response.Name)
	assert.Equal(t, JSONUTCTime(serviceGroup.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(serviceGroup.UpdatedAt), response.UpdatedAt)
}
