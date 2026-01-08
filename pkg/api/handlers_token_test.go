package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewTokenHandler tests the constructor
func TestNewTokenHandler(t *testing.T) {
	tokenQuerier := domain.NewMockTokenQuerier(t)
	agentQuerier := domain.NewMockAgentQuerier(t)
	commander := domain.NewMockTokenCommander(t)
	store := domain.NewMockStore(t)
	authz := auth.NewMockAuthorizer(t)

	handler := NewTokenHandler(tokenQuerier, commander, agentQuerier, store, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, tokenQuerier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, agentQuerier, handler.agentQuerier)
	assert.Equal(t, authz, handler.authz)
}

// TestTokenHandlerRoutes tests that routes are properly registered
func TestTokenHandlerRoutes(t *testing.T) {
	// Create mocks
	tokenQuerier := domain.NewMockTokenQuerier(t)
	agentQuerier := domain.NewMockAgentQuerier(t)
	commander := domain.NewMockTokenCommander(t)
	store := domain.NewMockStore(t)
	authz := auth.NewMockAuthorizer(t)

	// Create the handler
	handler := NewTokenHandler(tokenQuerier, commander, agentQuerier, store, authz)

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

// TestTokenToResponse tests the tokenToResponse function
func TestTokenToResponse(t *testing.T) {
	now := time.Now()
	participantID := uuid.New()
	agentID := uuid.New()

	// Test token with all fields populated
	token := &domain.Token{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		Name:          "Test Token",
		Role:          auth.RoleParticipant,
		ExpireAt:      now.Add(time.Hour),
		ParticipantID: &participantID,
		AgentID:       &agentID,
		HashedValue:   "hashed_value",
		PlainValue:    "plain_value",
	}

	response := TokenToRes(token)

	assert.Equal(t, token.ID, response.ID)
	assert.Equal(t, token.Name, response.Name)
	assert.Equal(t, token.Role, response.Role)
	assert.Equal(t, JSONUTCTime(token.ExpireAt), response.ExpireAt)
	assert.Equal(t, &participantID, response.ParticipantID)
	assert.Equal(t, &agentID, response.AgentID)
	assert.Equal(t, JSONUTCTime(token.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(token.UpdatedAt), response.UpdatedAt)
	assert.Equal(t, "plain_value", response.Value)
}
