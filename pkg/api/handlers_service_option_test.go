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

// TestNewServiceOptionHandler tests the constructor
func TestNewServiceOptionHandler(t *testing.T) {
	querier := domain.NewMockServiceOptionQuerier(t)
	commander := domain.NewMockServiceOptionCommander(t)
	authz := auth.NewMockAuthorizer(t)

	handler := NewServiceOptionHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestServiceOptionHandlerRoutes tests that routes are properly registered
func TestServiceOptionHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := domain.NewMockServiceOptionQuerier(t)
	commander := domain.NewMockServiceOptionCommander(t)
	authz := auth.NewMockAuthorizer(t)

	// Create the handler
	handler := NewServiceOptionHandler(querier, commander, authz)

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

// TestServiceOptionToRes tests the ServiceOptionToRes function
func TestServiceOptionToRes(t *testing.T) {
	// Create a service option
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	providerID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	typeID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	option := &domain.ServiceOption{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		ProviderID:          properties.UUID(providerID),
		ServiceOptionTypeID: properties.UUID(typeID),
		Name:                "Ubuntu 20.04",
		Value:               map[string]any{"image": "ubuntu-20.04"},
		Enabled:             true,
		DisplayOrder:        1,
	}

	// Convert to response
	res := ServiceOptionToRes(option)

	// Verify
	assert.Equal(t, properties.UUID(id), res.ID)
	assert.Equal(t, properties.UUID(providerID), res.ProviderID)
	assert.Equal(t, properties.UUID(typeID), res.ServiceOptionTypeID)
	assert.Equal(t, "Ubuntu 20.04", res.Name)
	assert.Equal(t, map[string]any{"image": "ubuntu-20.04"}, res.Value)
	assert.True(t, res.Enabled)
	assert.Equal(t, 1, res.DisplayOrder)
	assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
}

// TestCreateServiceOptionReq_ObjectScope tests the ObjectScope method
func TestCreateServiceOptionReq_ObjectScope(t *testing.T) {
	providerID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")

	req := CreateServiceOptionReq{
		ProviderID: properties.UUID(providerID),
	}

	scope, err := req.ObjectScope()
	assert.NoError(t, err)
	assert.NotNil(t, scope)

	// Verify it's a DefaultObjectScope with provider ID
	defaultScope, ok := scope.(*auth.DefaultObjectScope)
	assert.True(t, ok, "Should return DefaultObjectScope")
	assert.NotNil(t, defaultScope.ProviderID)
	assert.Equal(t, properties.UUID(providerID), *defaultScope.ProviderID)
}
