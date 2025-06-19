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

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/middlewares"
	"github.com/fulcrumproject/commons/properties"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTokenHandler tests the constructor
func TestNewTokenHandler(t *testing.T) {
	tokenQuerier := &mockTokenQuerier{}
	agentQuerier := &mockAgentQuerier{}
	commander := &mockTokenCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewTokenHandler(tokenQuerier, commander, agentQuerier, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, tokenQuerier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, agentQuerier, handler.agentQuerier)
	assert.Equal(t, authz, handler.authz)
}

// TestTokenHandlerRoutes tests that routes are properly registered
func TestTokenHandlerRoutes(t *testing.T) {
	// Create mocks
	tokenQuerier := &mockTokenQuerier{}
	agentQuerier := &mockAgentQuerier{}
	commander := &mockTokenCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewTokenHandler(tokenQuerier, commander, agentQuerier, authz)

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
		case method == "POST" && route == "/{id}/regenerate":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestTokenHandleCreate tests the handleCreate method
func TestTokenHandleCreate(t *testing.T) {
	now := time.Now().UTC()
	wantedExpireAt := now.Add(24 * time.Hour)

	// Setup test cases
	testCases := []struct {
		name           string
		requestBody    string
		mockSetup      func(tokenQuerier *mockTokenQuerier, agentQuerier *mockAgentQuerier, commander *mockTokenCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success - Create Admin Token",
			requestBody: fmt.Sprintf(`{
				"name": "Test Admin Token",
				"role": "admin",
				"expireAt": "%s"
			}`, wantedExpireAt.Format(time.RFC3339)),
			mockSetup: func(tokenQuerier *mockTokenQuerier, agentQuerier *mockAgentQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the commander
				commander.createFunc = func(ctx context.Context, name string, role auth.Role, expireAt *time.Time, scopeID *properties.UUID) (*domain.Token, error) {
					assert.Equal(t, "Test Admin Token", name)
					assert.Equal(t, auth.RoleAdmin, role)
					assert.WithinDuration(t, wantedExpireAt, *expireAt, time.Second) // Compare with some tolerance
					assert.Nil(t, scopeID)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.Token{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:        name,
						Role:        role,
						ExpireAt:    *expireAt,
						HashedValue: "hashed_value",
						PlainValue:  "plain_value",
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Success - Create Provider Admin Token",
			requestBody: fmt.Sprintf(`{
				"name": "Test Provider Token",
				"role": "participant",
				"expireAt": "%s",
				"scopeId": "550e8400-e29b-41d4-a716-446655440000"
			}`, wantedExpireAt.Format(time.RFC3339)),
			mockSetup: func(tokenQuerier *mockTokenQuerier, agentQuerier *mockAgentQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the commander
				commander.createFunc = func(ctx context.Context, name string, role auth.Role, expireAt *time.Time, scopeID *properties.UUID) (*domain.Token, error) {
					assert.Equal(t, "Test Provider Token", name)
					assert.Equal(t, auth.RoleParticipant, role)
					assert.WithinDuration(t, wantedExpireAt, *expireAt, time.Second)
					require.NotNil(t, scopeID)
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), *scopeID)

					participantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.Token{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:          name,
						Role:          role,
						ParticipantID: &participantID,
						ExpireAt:      *expireAt,
						HashedValue:   "hashed_value",
						PlainValue:    "plain_value",
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Success - Create Admin Token",
			requestBody: fmt.Sprintf(`{
				"name": "Test Admin Token",
				"role": "admin",
				"expireAt": "%s"
			}`, wantedExpireAt.Format(time.RFC3339)),
			mockSetup: func(tokenQuerier *mockTokenQuerier, agentQuerier *mockAgentQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the commander for successful creation
				commander.createFunc = func(ctx context.Context, name string, role auth.Role, expireAt *time.Time, scopeID *properties.UUID) (*domain.Token, error) {
					assert.Equal(t, "Test Admin Token", name)
					assert.Equal(t, auth.RoleAdmin, role)
					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					return &domain.Token{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("aa0e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:       name,
						Role:       role,
						ExpireAt:   *expireAt,
						PlainValue: "token-value-123",
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "CommanderError",
			requestBody: fmt.Sprintf(`{
				"name": "Test Admin Token",
				"role": "admin",
				"expireAt": "%s"
			}`, wantedExpireAt.Format(time.RFC3339)),
			mockSetup: func(tokenQuerier *mockTokenQuerier, agentQuerier *mockAgentQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the commander to return an error
				commander.createFunc = func(ctx context.Context, name string, role auth.Role, expireAt *time.Time, scopeID *properties.UUID) (*domain.Token, error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			tokenQuerier := &mockTokenQuerier{}
			agentQuerier := &mockAgentQuerier{}
			commander := &mockTokenCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true} // Not used in handler tests
			tc.mockSetup(tokenQuerier, agentQuerier, commander, authz)

			// Create the handler
			handler := NewTokenHandler(tokenQuerier, commander, agentQuerier, authz)

			// Create request with properties.JSON body
			req := httptest.NewRequest("POST", "/tokens", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.DecodeBody[CreateTokenRequest]()(http.HandlerFunc(handler.handleCreate))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusCreated {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, "aa0e8400-e29b-41d4-a716-446655440000", response["id"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
				assert.NotEmpty(t, response["value"]) // Token value should be returned
			}
		})
	}
}

// TestTokenHandleGet tests the handleGet method
func TestTokenHandleGet(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				tokenQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				tokenQuerier.findByIDFunc = func(ctx context.Context, id properties.UUID) (*domain.Token, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					expireAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.Token{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:        "Test Token",
						Role:        auth.RoleAdmin,
						ExpireAt:    expireAt,
						HashedValue: "hashed_value",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				tokenQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				// Setup the querier to return not found
				tokenQuerier.findByIDFunc = func(ctx context.Context, id properties.UUID) (*domain.Token, error) {
					return nil, domain.NotFoundError{Err: fmt.Errorf("token not found")}
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			tokenQuerier := &mockTokenQuerier{}
			agentQuerier := &mockAgentQuerier{}
			commander := &mockTokenCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(tokenQuerier, commander, authz)

			// Create the handler
			handler := NewTokenHandler(tokenQuerier, commander, agentQuerier, authz)

			// Create request
			req := httptest.NewRequest("GET", "/tokens/"+tc.id, nil)

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context (always required)
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
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
				assert.Equal(t, "Test Token", response["name"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
				// Plain value should not be returned in get request
				assert.Nil(t, response["value"])
			}
		})
	}
}

// TestTokenHandleList tests the handleList method
func TestTokenHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(tokenQuerier *mockTokenQuerier, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(tokenQuerier *mockTokenQuerier, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return tokens
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				expireAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				participantID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")

				tokenQuerier.listFunc = func(ctx context.Context, scope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Token], error) {
					return &domain.PageResponse[domain.Token]{
						Items: []domain.Token{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:        "Token 1",
								Role:        auth.RoleAdmin,
								ExpireAt:    expireAt,
								HashedValue: "hashed_value_1",
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("bb0e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:          "Token 2",
								Role:          auth.RoleParticipant,
								ParticipantID: &participantID,
								ExpireAt:      expireAt,
								HashedValue:   "hashed_value_2",
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
			mockSetup: func(tokenQuerier *mockTokenQuerier, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				tokenQuerier.listFunc = func(ctx context.Context, scope *auth.IdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Token], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			tokenQuerier := &mockTokenQuerier{}
			agentQuerier := &mockAgentQuerier{}
			commander := &mockTokenCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(tokenQuerier, authz)

			// Create the handler
			handler := NewTokenHandler(tokenQuerier, commander, agentQuerier, authz)

			// Create request
			var req *http.Request
			if tc.name == "InvalidPageRequest" {
				// Create invalid page request
				req = httptest.NewRequest("GET", "/tokens?page=invalid", nil)
			} else {
				req = httptest.NewRequest("GET", "/tokens?page=1&pageSize=10", nil)
			}

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAdmin()
			ctx := auth.WithIdentity(req.Context(), authIdentity)
			req = req.WithContext(ctx)

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
				assert.Equal(t, "Token 1", firstItem["name"])

				secondItem := items[1].(map[string]any)
				assert.Equal(t, "bb0e8400-e29b-41d4-a716-446655440000", secondItem["id"])
				assert.Equal(t, "Token 2", secondItem["name"])
			}
		})
	}
}

// TestTokenHandleUpdate tests the handleUpdate method
func TestTokenHandleUpdate(t *testing.T) {
	// Setup test cases
	now := time.Now().UTC()
	newExpiration := now.Add(48 * time.Hour)

	testCases := []struct {
		name           string
		id             string
		requestBody    string
		mockSetup      func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name:        "Success - Update Name",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"name": "Updated Token"}`,
			mockSetup: func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				tokenQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				// Setup the commander to update
				commander.updateFunc = func(ctx context.Context, id properties.UUID, name *string, expireAt *time.Time) (*domain.Token, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					require.NotNil(t, name)
					assert.Equal(t, "Updated Token", *name)
					assert.Nil(t, expireAt)

					newName := "Updated Token"
					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
					expireDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.Token{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:        newName,
						Role:        auth.RoleAdmin,
						ExpireAt:    expireDate,
						HashedValue: "hashed_value",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Success - Update Expiration",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: fmt.Sprintf(`{"expireAt": "%s"}`, newExpiration.Format(time.RFC3339)),
			mockSetup: func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				tokenQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				// Setup the commander to update
				commander.updateFunc = func(ctx context.Context, id properties.UUID, name *string, expireAt *time.Time) (*domain.Token, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					assert.Nil(t, name)
					require.NotNil(t, expireAt)
					assert.WithinDuration(t, newExpiration, *expireAt, time.Second)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

					return &domain.Token{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:        "Test Token",
						Role:        auth.RoleAdmin,
						ExpireAt:    *expireAt,
						HashedValue: "hashed_value",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "CommanderError",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"name": "Updated Token"}`,
			mockSetup: func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				tokenQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				// Setup the commander to return an error
				commander.updateFunc = func(ctx context.Context, id properties.UUID, name *string, expireAt *time.Time) (*domain.Token, error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			tokenQuerier := &mockTokenQuerier{}
			agentQuerier := &mockAgentQuerier{}
			commander := &mockTokenCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(tokenQuerier, commander, authz)

			// Create the handler
			handler := NewTokenHandler(tokenQuerier, commander, agentQuerier, authz)

			// Create request with properties.JSON body
			req := httptest.NewRequest("PATCH", "/tokens/"+tc.id, strings.NewReader(tc.requestBody))
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
			middlewareHandler := middlewares.DecodeBody[UpdateTokenRequest]()(middlewares.ID(http.HandlerFunc(handler.handleUpdate)))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, tc.id, response["id"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
			}
		})
	}
}

// TestTokenHandleDelete tests the handleDelete method
func TestTokenHandleDelete(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				tokenQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				// Setup the commander to delete
				commander.deleteFunc = func(ctx context.Context, id properties.UUID) error {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		// Remove authorization test case since auth is now handled by middleware
		{
			name: "CommanderError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				tokenQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				// Setup the commander to return an error
				commander.deleteFunc = func(ctx context.Context, id properties.UUID) error {
					return fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			tokenQuerier := &mockTokenQuerier{}
			agentQuerier := &mockAgentQuerier{}
			commander := &mockTokenCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(tokenQuerier, commander, authz)

			// Create the handler
			handler := NewTokenHandler(tokenQuerier, commander, agentQuerier, authz)

			// Create request
			req := httptest.NewRequest("DELETE", "/tokens/"+tc.id, nil)

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

// TestTokenHandleRegenerateValue tests the handleRegenerateValue method
func TestTokenHandleRegenerateValue(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				tokenQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				// Setup the commander to regenerate
				commander.regenerateFunc = func(ctx context.Context, id properties.UUID) (*domain.Token, error) {
					assert.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), id)

					createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
					expireAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

					return &domain.Token{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:        "Test Token",
						Role:        auth.RoleAdmin,
						ExpireAt:    expireAt,
						HashedValue: "new_hashed_value",
						PlainValue:  "new_plain_value",
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		// Remove authorization test case since auth is now handled by middleware
		{
			name: "CommanderError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(tokenQuerier *mockTokenQuerier, commander *mockTokenCommander, authz *MockAuthorizer) {
				// Setup the mock to authorize successfully
				authz.ShouldSucceed = true

				// Setup the querier to return auth scope
				tokenQuerier.authScopeFunc = func(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
					return &auth.AllwaysMatchObjectScope{}, nil
				}

				// Setup the commander to return an error
				commander.regenerateFunc = func(ctx context.Context, id properties.UUID) (*domain.Token, error) {
					return nil, fmt.Errorf("regeneration error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			tokenQuerier := &mockTokenQuerier{}
			agentQuerier := &mockAgentQuerier{}
			commander := &mockTokenCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(tokenQuerier, commander, authz)

			// Create the handler
			handler := NewTokenHandler(tokenQuerier, commander, agentQuerier, authz)

			// Create request
			req := httptest.NewRequest("POST", "/tokens/"+tc.id+"/regenerate", nil)

			// Set up chi router context for URL parameters FIRST
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAdmin()
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request with middleware
			w := httptest.NewRecorder()
			middlewareHandler := middlewares.ID(http.HandlerFunc(handler.handleRegenerateValue))
			middlewareHandler.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				// Verify response structure
				assert.Equal(t, tc.id, response["id"])
				assert.Equal(t, "Test Token", response["name"])
				assert.NotEmpty(t, response["createdAt"])
				assert.NotEmpty(t, response["updatedAt"])
				// The plain token value should be returned in the response
				assert.Equal(t, "new_plain_value", response["value"])
			}
		})
	}
}

// TestTokenToResponse tests the tokenToResponse function
func TestTokenToResponse(t *testing.T) {
	// Create a token
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	participantID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	expireAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	token := &domain.Token{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:          "Test Token",
		Role:          auth.RoleParticipant,
		ParticipantID: &participantID,
		ExpireAt:      expireAt,
		HashedValue:   "hashed_value",
		PlainValue:    "plain_value",
	}

	// Convert to response
	response := tokenToResponse(token)

	// Verify all fields are correctly mapped
	assert.Equal(t, id, response.ID)
	assert.Equal(t, "Test Token", response.Name)
	assert.Equal(t, auth.RoleParticipant, response.Role)
	assert.Equal(t, participantID, *response.ParticipantID)
	assert.Equal(t, "plain_value", response.Value)
	assert.Equal(t, JSONUTCTime(expireAt), response.ExpireAt)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
}
