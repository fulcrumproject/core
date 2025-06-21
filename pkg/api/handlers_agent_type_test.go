package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewAgentTypeHandler tests the constructor
func TestNewAgentTypeHandler(t *testing.T) {
	querier := &mockAgentTypeQuerier{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewAgentTypeHandler(querier, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, authz, handler.authz)
}

// TestAgentTypeHandlerRoutes tests that routes are properly registered
func TestAgentTypeHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockAgentTypeQuerier{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewAgentTypeHandler(querier, authz)

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
		case method == "GET" && route == "/{id}":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestAgentTypeToResponse tests the agentTypeToResponse function
func TestAgentTypeToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	agentType := &domain.AgentType{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name: "TestAgentType",
		ServiceTypes: []domain.ServiceType{
			{
				BaseEntity: domain.BaseEntity{
					ID:        uuid.MustParse("650e8400-e29b-41d4-a716-446655440000"),
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Name: "TestServiceType",
			},
		},
	}

	response := AgentTypeToRes(agentType)

	assert.Equal(t, agentType.ID, response.ID)
	assert.Equal(t, agentType.Name, response.Name)
	assert.Equal(t, JSONUTCTime(agentType.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(agentType.UpdatedAt), response.UpdatedAt)
	assert.Len(t, response.ServiceTypes, 1)
	assert.Equal(t, agentType.ServiceTypes[0].ID, response.ServiceTypes[0].ID)
	assert.Equal(t, agentType.ServiceTypes[0].Name, response.ServiceTypes[0].Name)
}
