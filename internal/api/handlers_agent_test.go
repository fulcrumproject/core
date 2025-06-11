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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleCreate tests the handleCreate method (pure business logic)
func TestHandleCreate(t *testing.T) {
	testCases := []struct {
		name           string
		requestBody    CreateAgentRequest
		mockSetup      func(commander *mockAgentCommander)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			requestBody: CreateAgentRequest{
				Name:        "TestAgent",
				ProviderID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				AgentTypeID: uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
			},
			mockSetup: func(commander *mockAgentCommander) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				commander.createFunc = func(ctx context.Context, name string, providerID domain.UUID, agentTypeID domain.UUID, tags []string) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:             name,
						Status:           domain.AgentDisconnected,
						LastStatusUpdate: createdAt,
						ProviderID:       providerID,
						AgentTypeID:      agentTypeID,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"id":          "550e8400-e29b-41d4-a716-446655440001",
				"name":        "TestAgent",
				"status":      "Disconnected",
				"providerId":  "550e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "660e8400-e29b-41d4-a716-446655440000",
				"tags":        interface{}(nil),
				"createdAt":   "2023-01-01T00:00:00Z",
				"updatedAt":   "2023-01-01T00:00:00Z",
			},
		},
		{
			name: "CommanderError",
			requestBody: CreateAgentRequest{
				Name:        "TestAgent",
				ProviderID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
				AgentTypeID: uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
			},
			mockSetup: func(commander *mockAgentCommander) {
				commander.createFunc = func(ctx context.Context, name string, providerID domain.UUID, agentTypeID domain.UUID, tags []string) (*domain.Agent, error) {
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
			tc.mockSetup(commander)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request with decoded body in context (simulating DecodeBody middleware)
			req := httptest.NewRequest("POST", "/agents", nil)
			req = req.WithContext(context.WithValue(req.Context(), decodedBodyContextKey, tc.requestBody))

			// Add auth identity to context
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

// TestAgentHandleGet tests the handleGet method (pure business logic)
func TestAgentHandleGet(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(querier *mockAgentQuerier)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:             "TestAgent",
						Status:           domain.AgentConnected,
						LastStatusUpdate: createdAt,
						ProviderID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
						AgentTypeID:      uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"id":          "550e8400-e29b-41d4-a716-446655440000",
				"name":        "TestAgent",
				"status":      "Connected",
				"providerId":  "660e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"tags":        interface{}(nil),
				"createdAt":   "2023-01-01T00:00:00Z",
				"updatedAt":   "2023-01-01T00:00:00Z",
			},
		},
		{
			name: "NotFound",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier) {
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
			tc.mockSetup(querier)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request with ID in context (simulating ID middleware)
			req := httptest.NewRequest("GET", "/agents/"+tc.id, nil)
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))

			// Add auth identity to context
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
	testCases := []struct {
		name           string
		agentID        string
		mockSetup      func(querier *mockAgentQuerier)
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:    "Success",
			agentID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.findByIDFunc = func(ctx context.Context, id domain.UUID) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:             "TestAgent",
						Status:           domain.AgentConnected,
						LastStatusUpdate: createdAt,
						ProviderID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
						AgentTypeID:      uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"id":          "550e8400-e29b-41d4-a716-446655440000",
				"name":        "TestAgent",
				"status":      "Connected",
				"providerId":  "660e8400-e29b-41d4-a716-446655440000",
				"agentTypeId": "770e8400-e29b-41d4-a716-446655440000",
				"tags":        interface{}(nil),
				"createdAt":   "2023-01-01T00:00:00Z",
				"updatedAt":   "2023-01-01T00:00:00Z",
			},
		},
		{
			name:    "NotFound",
			agentID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier) {
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
			tc.mockSetup(querier)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/agents/me", nil)

			// Add agent auth identity to context (simulating RequireAgentIdentity middleware)
			agentUUID := uuid.MustParse(tc.agentID)
			authIdentity := NewMockAuthAgentWithID(agentUUID)
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
	testCases := []struct {
		name           string
		mockSetup      func(querier *mockAgentQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			mockSetup: func(querier *mockAgentQuerier) {
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
								Name:             "TestAgent1",
								Status:           domain.AgentConnected,
								LastStatusUpdate: createdAt,
								ProviderID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
								AgentTypeID:      uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
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
		},
		{
			name: "ListError",
			mockSetup: func(querier *mockAgentQuerier) {
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
			tc.mockSetup(querier)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request
			req := httptest.NewRequest("GET", "/agents", nil)

			// Add auth identity to context
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleList(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestHandleUpdate tests the handleUpdate method
func TestHandleUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		requestBody    UpdateAgentRequest
		mockSetup      func(commander *mockAgentCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: UpdateAgentRequest{
				Name: stringPtr("UpdatedAgent"),
			},
			mockSetup: func(commander *mockAgentCommander) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, status *domain.AgentStatus, tags *[]string) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:             *name,
						Status:           domain.AgentConnected,
						LastStatusUpdate: createdAt,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "UpdateError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			requestBody: UpdateAgentRequest{
				Name: stringPtr("UpdatedAgent"),
			},
			mockSetup: func(commander *mockAgentCommander) {
				commander.updateFunc = func(ctx context.Context, id domain.UUID, name *string, status *domain.AgentStatus, tags *[]string) (*domain.Agent, error) {
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
			tc.mockSetup(commander)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request with decoded body and ID in context
			req := httptest.NewRequest("PATCH", "/agents/"+tc.id, nil)
			req = req.WithContext(context.WithValue(req.Context(), decodedBodyContextKey, tc.requestBody))

			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))

			// Add auth identity to context
			authIdentity := NewMockAuthAgent()
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleUpdate(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestHandleUpdateStatusMe tests the handleUpdateStatusMe method
func TestHandleUpdateStatusMe(t *testing.T) {
	testCases := []struct {
		name           string
		agentID        string
		requestBody    UpdateAgentStatusRequest
		mockSetup      func(commander *mockAgentCommander)
		expectedStatus int
	}{
		{
			name:    "Success",
			agentID: "550e8400-e29b-41d4-a716-446655440000",
			requestBody: UpdateAgentStatusRequest{
				Status: domain.AgentConnected,
			},
			mockSetup: func(commander *mockAgentCommander) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				commander.updateStatusFunc = func(ctx context.Context, id domain.UUID, status domain.AgentStatus) (*domain.Agent, error) {
					return &domain.Agent{
						BaseEntity: domain.BaseEntity{
							ID:        id,
							CreatedAt: createdAt,
							UpdatedAt: updatedAt,
						},
						Name:             "TestAgent",
						Status:           status,
						LastStatusUpdate: createdAt,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "UpdateError",
			agentID: "550e8400-e29b-41d4-a716-446655440000",
			requestBody: UpdateAgentStatusRequest{
				Status: domain.AgentConnected,
			},
			mockSetup: func(commander *mockAgentCommander) {
				commander.updateStatusFunc = func(ctx context.Context, id domain.UUID, status domain.AgentStatus) (*domain.Agent, error) {
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
			tc.mockSetup(commander)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request with decoded body
			req := httptest.NewRequest("PUT", "/agents/me/status", nil)
			req = req.WithContext(context.WithValue(req.Context(), decodedBodyContextKey, tc.requestBody))

			// Add agent auth identity to context
			agentUUID := uuid.MustParse(tc.agentID)
			authIdentity := NewMockAuthAgentWithID(agentUUID)
			req = req.WithContext(domain.WithAuthIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.handleUpdateStatusMe(w, req)

			// Assert response
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

// TestHandleDelete tests the handleDelete method
func TestHandleDelete(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(commander *mockAgentCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(commander *mockAgentCommander) {
				commander.deleteFunc = func(ctx context.Context, id domain.UUID) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "DeleteError",
			id:   "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(commander *mockAgentCommander) {
				commander.deleteFunc = func(ctx context.Context, id domain.UUID) error {
					return domain.NewNotFoundErrorf("agent not found")
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
			tc.mockSetup(commander)

			// Create the handler
			handler := NewAgentHandler(querier, commander, authz)

			// Create request with ID in context (simulating ID middleware)
			req := httptest.NewRequest("DELETE", "/agents/"+tc.id, nil)
			parsedUUID, _ := domain.ParseUUID(tc.id)
			req = req.WithContext(context.WithValue(req.Context(), uuidContextKey, parsedUUID))

			// Add auth identity to context
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

// TestNewAgentHandler tests the NewAgentHandler constructor
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
		Name:             "TestAgent",
		Status:           domain.AgentConnected,
		LastStatusUpdate: createdAt,
		ProviderID:       uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
		AgentTypeID:      uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
	}

	response := agentToResponse(agent)
	assert.Equal(t, agent.ID, response.ID)
	assert.Equal(t, agent.Name, response.Name)
	assert.Equal(t, agent.Status, response.Status)
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}
