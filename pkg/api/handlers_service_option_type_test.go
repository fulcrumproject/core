package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewServiceOptionTypeHandler tests the constructor
func TestNewServiceOptionTypeHandler(t *testing.T) {
	querier := domain.NewMockServiceOptionTypeQuerier(t)
	commander := domain.NewMockServiceOptionTypeCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewServiceOptionTypeHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestServiceOptionTypeHandlerRoutes tests that routes are properly registered
func TestServiceOptionTypeHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := domain.NewMockServiceOptionTypeQuerier(t)
	commander := domain.NewMockServiceOptionTypeCommander(t)
	authz := authz.NewMockAuthorizer(t)

	// Create the handler
	handler := NewServiceOptionTypeHandler(querier, commander, authz)

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

// TestServiceOptionTypeToRes tests the ServiceOptionTypeToRes function
func TestServiceOptionTypeToRes(t *testing.T) {
	// Create a service option type
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	optionType := &domain.ServiceOptionType{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:        "Operating System",
		Type:        "operating_system",
		Description: "Available operating systems",
	}

	// Convert to response
	res := ServiceOptionTypeToRes(optionType)

	// Verify
	assert.Equal(t, id, res.ID)
	assert.Equal(t, "Operating System", res.Name)
	assert.Equal(t, "operating_system", res.Type)
	assert.Equal(t, "Available operating systems", res.Description)
	assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
}
