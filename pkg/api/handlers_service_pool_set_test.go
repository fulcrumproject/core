package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewServicePoolSetHandler tests the constructor
func TestNewServicePoolSetHandler(t *testing.T) {
	querier := domain.NewMockServicePoolSetQuerier(t)
	commander := domain.NewMockServicePoolSetCommander(t)
	authz := auth.NewMockAuthorizer(t)

	handler := NewServicePoolSetHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestServicePoolSetHandlerRoutes tests that routes are properly registered
func TestServicePoolSetHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := domain.NewMockServicePoolSetQuerier(t)
	commander := domain.NewMockServicePoolSetCommander(t)
	authz := auth.NewMockAuthorizer(t)

	// Create the handler
	handler := NewServicePoolSetHandler(querier, commander, authz)

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

// TestServicePoolSetToRes tests the ServicePoolSetToRes function
func TestServicePoolSetToRes(t *testing.T) {
	// Create a service pool set
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	providerID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	poolSet := &domain.ServicePoolSet{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:       "Production Pools",
		ProviderID: properties.UUID(providerID),
	}

	// Convert to response
	res := ServicePoolSetToRes(poolSet)

	// Verify
	assert.Equal(t, properties.UUID(id), res.ID)
	assert.Equal(t, "Production Pools", res.Name)
	assert.Equal(t, properties.UUID(providerID), res.ProviderID)
	assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
}
