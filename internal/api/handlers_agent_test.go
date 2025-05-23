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

// TestHandleCreate tests the handleCreate method
func TestHandleCreate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		requestBody    string
		mockSetup      func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			requestBody: `{
				"name": "TestAgent",
				"countryCode": "US",
				"attributes": {"test": ["value1", "value2"]},
				"providerId": "550e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "660e8400-e29b-41d4-a716-446655440000"
			}`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return a test agent
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				commander.createFunc = func(ctx context.Context, name string, countryCode domain.CountryCode, attributes domain.Attributes, providerID domain.UUID, agentTypeID domain.UUID) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:            name,
						CountryCode:     countryCode,
						Attributes:      attributes,
						State:           domain.AgentDisconnected,
						LastStateUpdate: createdAt,
						ProviderID:      providerID,
						AgentTypeID:     agentTypeID,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"id":          "550e8400-e29b-41d4-a716-446655440001",
				"name":        "TestAgent",
				"countryCode": "US",
				"attributes": map[string]interface{}{
					"test": []interface{}{"value1", "value2"},
				},
				"state":       "Disconnected",
				"providerId":  "550e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "660e8400-e29b-41d4-a716-446655440000",
				"createdAt":   "2023-01-01T00:00:00Z",
				"updatedAt":   "2023-01-01T00:00:00Z",
			},
		},
		{
			name:        "InvalidRequest",
			requestBody: `{invalid json`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Create mock for the commander even though it won't reach it
				// This prevents the "create not mocked" error from showing up in test failures
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				commander.createFunc = func(ctx context.Context, name string, countryCode domain.CountryCode, attributes domain.Attributes, providerID domain.UUID, agentTypeID domain.UUID) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID: uuid.New(),
						},
						Name:            name,
						State:           domain.AgentDisconnected,
						LastStateUpdate: createdAt,
					}, nil
				}
				// Authorization should not be called for invalid requests
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "UnauthorizedCreate",
			requestBody: `{
				"name": "TestAgent",
				"providerId": "550e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "660e8400-e29b-41d4-a716-446655440000"
			}`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "CommanderError",
			requestBody: `{
				"name": "TestAgent",
				"providerId": "550e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "660e8400-e29b-41d4-a716-446655440000"
			}`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return an error
				commander.createFunc = func(ctx context.Context, name string, countryCode domain.CountryCode, attributes domain.Attributes, providerID domain.UUID, agentTypeID domain.UUID) (*domain.Agent, error) {
					return nil, domain.NewInvalidInputErrorf("provider not found")
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockAgentQuerier{}
			commander := &mockAgentCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("POST", "/agents", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Add auth identity to context for authorization
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
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

// TestAgentHandleGet tests the handleGet method
func TestAgentHandleGet(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return a test agent
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:            "TestAgent",
						CountryCode:     "US",
						Attributes:      domain.Attributes{"test": []string{"value1", "value2"}},
						State:           domain.AgentConnected,
						LastStateUpdate: createdAt,
						ProviderID:      uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
						AgentTypeID:     uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
					}, nil
				}

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"id":          "550e8400-e29b-41d4-a716-446655440000",
				"name":        "TestAgent",
				"countryCode": "US",
				"attributes": map[string]interface{}{
					"test": []interface{}{"value1", "value2"},
				},
				"state":       "Connected",
				"providerId":  "660e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"createdAt":   "2023-01-01T00:00:00Z",
				"updatedAt":   "2023-01-01T00:00:00Z",
			},
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
					return nil, domain.NewNotFoundErrorf("agent not found")
				}

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Unauthorized",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "AuthScopeError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return nil, domain.NewNotFoundErrorf("scope not found")
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockAgentQuerier{}
			commander := &mockAgentCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/agents/"+tc.id, nil)
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
			handler.handleGet(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

// TestHandleGetMe tests the handleGetMe method
func TestHandleGetMe(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		agentID        string
		mockSetup      func(querier *mockAgentQuerier, commander *mockAgentCommander)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:    "Success",
			agentID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander) {
				// Setup the mock to return a test agent
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:            "TestAgent",
						CountryCode:     "US",
						Attributes:      domain.Attributes{"test": []string{"value1", "value2"}},
						State:           domain.AgentConnected,
						LastStateUpdate: createdAt,
						ProviderID:      uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
						AgentTypeID:     uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"id":          "550e8400-e29b-41d4-a716-446655440000",
				"name":        "TestAgent",
				"countryCode": "US",
				"attributes": map[string]interface{}{
					"test": []interface{}{"value1", "value2"},
				},
				"state":       "Connected",
				"providerId":  "660e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"createdAt":   "2023-01-01T00:00:00Z",
				"updatedAt":   "2023-01-01T00:00:00Z",
			},
		},
		{
			name:    "NotFound",
			agentID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander) {
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
					return nil, domain.NewNotFoundErrorf("agent not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockAgentQuerier{}
			commander := &mockAgentCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/agents/me", nil)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleGetMe(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

// TestAgentHandleList tests the handleList method
func TestAgentHandleList(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				// Setup the mock to return test agents
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Agent], error) {
					return &domain.PageResponse[domain.Agent]{
						Items: []domain.Agent{
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:            "TestAgent1",
								CountryCode:     "US",
								Attributes:      domain.Attributes{"test": []string{"value1", "value2"}},
								State:           domain.AgentConnected,
								LastStateUpdate: createdAt,
								ProviderID:      uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
								AgentTypeID:     uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
							},
							{
								BaseEntity: domain.BaseEntity{
									ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
									CreatedAt: createdAt,
									UpdatedAt: updatedAt,
								},
								Name:            "TestAgent2",
								CountryCode:     "CA",
								Attributes:      domain.Attributes{"test": []string{"value3", "value4"}},
								State:           domain.AgentDisconnected,
								LastStateUpdate: createdAt,
								ProviderID:      uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
								AgentTypeID:     uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
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
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "ListError",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.listFunc = func(ctx context.Context, authScope *domain.AuthIdentityScope, req *domain.PageRequest) (*domain.PageResponse[domain.Agent], error) {
					return nil, fmt.Errorf("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockAgentQuerier{}
			commander := &mockAgentCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/agents?page=1&pageSize=10", nil)

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
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

				items := response["items"].([]interface{})
				assert.Equal(t, 2, len(items))
			}
		})
	}
}

// TestHandleUpdate tests the handleUpdate method
func TestHandleUpdate(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		requestBody    string
		mockSetup      func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"name": "UpdatedAgentName",
				"countryCode": "CA",
				"attributes": {"test": ["value3", "value4"]}
			}`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Setup the mock to return a test agent
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, countryCode *domain.CountryCode, attributes *domain.Attributes, state *domain.AgentState) (*domain.Agent, error) {
					nameVal := "UpdatedAgentName"
					ccVal := domain.CountryCode("CA")
					attrVal := domain.Attributes{"test": []string{"value3", "value4"}}
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:            nameVal,
						CountryCode:     ccVal,
						Attributes:      attrVal,
						State:           domain.AgentConnected,
						LastStateUpdate: createdAt,
						ProviderID:      uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
						AgentTypeID:     uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"id":          "550e8400-e29b-41d4-a716-446655440000",
				"name":        "UpdatedAgentName",
				"countryCode": "CA",
				"attributes": map[string]interface{}{
					"test": []interface{}{"value3", "value4"},
				},
				"state":       "Connected",
				"providerId":  "660e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"createdAt":   "2023-01-01T00:00:00Z",
				"updatedAt":   "2023-01-02T00:00:00Z",
			},
		},
		{
			name:        "InvalidRequest",
			id:          "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{invalid json`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
				// Authorization and commander should not be called for invalid requests
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"name": "UpdatedAgentName"
			}`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "UpdateError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{
				"name": "UpdatedAgentName"
			}`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Setup the mock to return an error
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, countryCode *domain.CountryCode, attributes *domain.Attributes, state *domain.AgentState) (*domain.Agent, error) {
					return nil, domain.NewNotFoundErrorf("agent not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockAgentQuerier{}
			commander := &mockAgentCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("PUT", "/agents/"+tc.id, strings.NewReader(tc.requestBody))
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

			// Execute request
			w := httptest.NewRecorder()
			handler.handleUpdate(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

// TestHandleUpdateStatusMe tests the handleUpdateStatusMe method
func TestHandleUpdateStatusMe(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		agentID        string
		requestBody    string
		mockSetup      func(querier *mockAgentQuerier, commander *mockAgentCommander)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:        "Success",
			agentID:     "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"state": "Connected"}`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander) {
				// Setup the mock to return a test agent
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

				commander.updateStateFunc = func(ctx context.Context, id domain.UUID, state domain.AgentState) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:            "TestAgent",
						CountryCode:     "US",
						Attributes:      domain.Attributes{"test": []string{"value1", "value2"}},
						State:           state,
						LastStateUpdate: updatedAt,
						ProviderID:      uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
						AgentTypeID:     uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"id":          "550e8400-e29b-41d4-a716-446655440000",
				"name":        "TestAgent",
				"countryCode": "US",
				"attributes": map[string]interface{}{
					"test": []interface{}{"value1", "value2"},
				},
				"state":       "Connected",
				"providerId":  "660e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"createdAt":   "2023-01-01T00:00:00Z",
				"updatedAt":   "2023-01-02T00:00:00Z",
			},
		},
		{
			name:        "InvalidRequest",
			agentID:     "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{invalid json`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander) {
				// No setup needed for invalid request
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Invalid State",
			agentID:     "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"state": "Invalid"}`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander) {
				// Mock to return a proper error for invalid state
				commander.updateStateFunc = func(ctx context.Context, id domain.UUID, state domain.AgentState) (*domain.Agent, error) {
					return nil, domain.NewInvalidInputErrorf("invalid state: %s", state)
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "UpdateError",
			agentID:     "550e8400-e29b-41d4-a716-446655440000",
			requestBody: `{"state": "Connected"}`,
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander) {
				commander.updateStateFunc = func(ctx context.Context, id domain.UUID, state domain.AgentState) (*domain.Agent, error) {
					return nil, domain.NewNotFoundErrorf("agent not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockAgentQuerier{}
			commander := &mockAgentCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("PATCH", "/agents/me/state", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Add auth identity to context for authorization
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleUpdateStatusMe(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tc.expectedBody, response)
			}
		})
	}
}

// TestHandleDelete tests the handleDelete method
func TestHandleDelete(t *testing.T) {
	// Setup test cases
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Mock findByID to avoid 404 error
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID: id,
						},
					}, nil
				}

				// Setup the mock to not return an error
				commander.deleteFunc = func(ctx context.Context, id domain.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "Unauthorized",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return an unsuccessful auth
				authz.ShouldSucceed = false

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "DeleteError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Mock findByID to avoid 404 error
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID: id,
						},
					}, nil
				}

				// Setup the mock to return an error
				commander.deleteFunc = func(ctx context.Context, id domain.UUID) error {
					return domain.NewInvalidInputErrorf("cannot delete agent with active services")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier, commander *mockAgentCommander, authz *MockAuthorizer) {
				// Return a successful auth
				authz.ShouldSucceed = true

				querier.authScopeFunc = func(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
					return &domain.EmptyAuthScope, nil
				}

				// Return not found for the agent
				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
					return nil, domain.NewNotFoundErrorf("agent not found")
				}
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := &mockAgentQuerier{}
			commander := &mockAgentCommander{}
			authz := &MockAuthorizer{ShouldSucceed: true}
			tc.mockSetup(querier, commander, authz)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("DELETE", "/agents/"+tc.id, nil)
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
			handler.handleDelete(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestNewAgentHandler tests the constructor
func TestNewAgentHandler(t *testing.T) {
	querier := &mockAgentQuerier{}
	commander := &mockAgentCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewAgentHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestAgentHandlerRoutes tests that routes are properly registered
func TestAgentHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockAgentQuerier{}
	commander := &mockAgentCommander{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewAgentHandler(querier, commander, authz)

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
		case method == "GET" && route == "/me":
		case method == "PUT" && route == "/me/status":
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

// TestAgentToResponse tests the agentToResponse function
func TestAgentToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	agent := &domain.Agent{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:            "TestAgent",
		CountryCode:     "US",
		Attributes:      domain.Attributes{"test": []string{"value1", "value2"}},
		State:           domain.AgentConnected,
		LastStateUpdate: createdAt,
		ProviderID:      uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
		AgentTypeID:     uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
	}

	response := agentToResponse(agent)

	assert.Equal(t, agent.ID, response.ID)
	assert.Equal(t, agent.Name, response.Name)
	assert.Equal(t, agent.CountryCode, response.CountryCode)
	assert.Equal(t, agent.Attributes, response.Attributes)
	assert.Equal(t, agent.State, response.State)
	assert.Equal(t, agent.ProviderID, response.ProviderID)
	assert.Equal(t, agent.AgentTypeID, response.AgentTypeID)
	assert.Equal(t, JSONUTCTime(agent.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(agent.UpdatedAt), response.UpdatedAt)
}

// TestMustGetAgentID tests the MustGetAgentID function
func TestMustGetAgentID(t *testing.T) {
	// Test the happy path
	r := httptest.NewRequest("GET", "/test", nil)
	authIdentity := NewMockAuthAgent()
	r = r.WithContext(domain.WithAuthIdentity(r.Context(), authIdentity))

	// Call MustGetAgentID
	id, err := MustGetAgentID(r.Context())
	assert.NoError(t, err)
	assert.Equal(t, authIdentity.id, id)

	// Test the error case by creating a sub-test
	t.Run("Error case", func(t *testing.T) {
		// Create a request with a non-agent auth identity
		r := httptest.NewRequest("GET", "/test", nil)
		adminIdentity := NewMockAuthParticipant()
		r = r.WithContext(domain.WithAuthIdentity(r.Context(), adminIdentity))

		// This should return an error
		_, err := MustGetAgentID(r.Context())
		assert.Error(t, err)
	})
}
