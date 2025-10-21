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

// TestNewServiceGroupHandler tests the constructor
func TestNewServiceGroupHandler(t *testing.T) {
	querier := domain.NewMockServiceGroupQuerier(t)
	commander := domain.NewMockServiceGroupCommander(t)
	authz := auth.NewMockAuthorizer(t)

	handler := NewServiceGroupHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestServiceGroupHandlerRoutes tests that routes are properly registered
func TestServiceGroupHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := domain.NewMockServiceGroupQuerier(t)
	commander := domain.NewMockServiceGroupCommander(t)
	authz := auth.NewMockAuthorizer(t)

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

// TestServiceGroupToResponse tests the serviceGroupToResponse function
func TestServiceGroupToResponse(t *testing.T) {
	// Create a service group
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	consumerID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	serviceGroup := &domain.ServiceGroup{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:       "Test Group",
		ConsumerID: consumerID,
	}

	// Convert to response
	response := ServiceGroupToRes(serviceGroup)

	// Verify all fields are correctly mapped
	assert.Equal(t, id, response.ID)
	assert.Equal(t, "Test Group", response.Name)
	assert.Equal(t, consumerID, response.ConsumerID)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
}
