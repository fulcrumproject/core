package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestHandleGetMe tests the handleGetMe method
func TestHandleGetMe(t *testing.T) {
	testCases := []struct {
		name           string
		agentID        string
		mockSetup      func(querier *mockAgentQuerier)
		expectedStatus int
	}{
		{
			name:    "Success",
			agentID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.GetFunc = func(ctx context.Context, id properties.UUID) (*domain.Agent, error) {
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
		},
		{
			name:    "NotFound",
			agentID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *mockAgentQuerier) {
				querier.GetFunc = func(ctx context.Context, id properties.UUID) (*domain.Agent, error) {
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
			req = req.WithContext(auth.WithIdentity(req.Context(), authIdentity))

			// Execute request
			w := httptest.NewRecorder()
			handler.GetMe(w, req)

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

// TestAgentToResponse tests the agentToResponse function
func TestAgentToResponse(t *testing.T) {
	// Create test agent
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	agent := &domain.Agent{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:        "TestAgent",
		Status:      domain.AgentConnected,
		ProviderID:  uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
		AgentTypeID: uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
		Tags:        []string{"tag1", "tag2"},
	}

	// Convert to response
	response := AgentToRes(agent)

	// Verify response
	assert.Equal(t, agent.ID, response.ID)
	assert.Equal(t, agent.Name, response.Name)
	assert.Equal(t, agent.Status, response.Status)
	assert.Equal(t, agent.ProviderID, response.ProviderID)
	assert.Equal(t, agent.AgentTypeID, response.AgentTypeID)
	assert.Equal(t, []string{"tag1", "tag2"}, response.Tags)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
}
