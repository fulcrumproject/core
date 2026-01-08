package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestHandleGetMe tests the handleGetMe method
func TestHandleGetMe(t *testing.T) {
	testCases := []struct {
		name           string
		agentID        string
		mockSetup      func(querier *domain.MockAgentQuerier)
		expectedStatus int
	}{
		{
			name:    "Success",
			agentID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *domain.MockAgentQuerier) {
				createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
				updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				querier.EXPECT().
					Get(mock.Anything, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(&domain.Agent{
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
						Configuration: &properties.JSON{
							"timeout": 60,
							"debug":   true,
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "NotFound",
			agentID: "550e8400-e29b-41d4-a716-446655440000",
			mockSetup: func(querier *domain.MockAgentQuerier) {
				querier.EXPECT().
					Get(mock.Anything, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")).
					Return(nil, domain.NewNotFoundErrorf("agent not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			querier := domain.NewMockAgentQuerier(t)
			commander := domain.NewMockAgentCommander(t)
			mockAuthz := authz.NewMockAuthorizer(t)
			tc.mockSetup(querier)

			// Create the handler
			handler := NewAgentHandler(querier, commander, mockAuthz)

			// Create request
			req := httptest.NewRequest("GET", "/agents/me", nil)

			// Add agent auth identity to context (simulating RequireAgentIdentity middleware)
			agentUUID := uuid.MustParse(tc.agentID)
			authIdentity := newMockAuthAgentWithID(agentUUID)
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
	querier := domain.NewMockAgentQuerier(t)
	commander := domain.NewMockAgentCommander(t)
	authz := authz.NewMockAuthorizer(t)

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
		Configuration: &properties.JSON{
			"timeout": 30,
			"retries": 3,
		},
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
	assert.Equal(t, agent.Configuration, response.Configuration)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
}

// TestAgentToResponse_NilConfiguration tests the agentToResponse function with nil configuration
func TestAgentToResponse_NilConfiguration(t *testing.T) {
	// Create test agent with nil configuration
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	agent := &domain.Agent{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:          "TestAgent",
		Status:        domain.AgentConnected,
		ProviderID:    uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
		AgentTypeID:   uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
		Tags:          []string{"tag1", "tag2"},
		Configuration: nil,
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
	assert.Nil(t, response.Configuration)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
}
