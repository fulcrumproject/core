package api

import (
	"fmt"
	"net/http"
	"testing"

	authmocks "github.com/fulcrumproject/core/pkg/auth/mocks"
	"github.com/fulcrumproject/core/pkg/domain/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// TestNewParticipantHandler tests the constructor
func TestNewParticipantHandler(t *testing.T) {
	querier := mocks.NewMockParticipantQuerier(t)
	commander := mocks.NewMockParticipantCommander(t)
	authz := authmocks.NewMockAuthorizer(t)

	handler := NewParticipantHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestParticipantHandlerRoutes tests that routes are properly registered
func TestParticipantHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := mocks.NewMockParticipantQuerier(t)
	commander := mocks.NewMockParticipantCommander(t)
	authz := authmocks.NewMockAuthorizer(t)

	// Create the handler
	handler := NewParticipantHandler(querier, commander, authz)

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
