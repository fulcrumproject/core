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

// TestNewServiceTypeHandler tests the constructor
func TestNewServiceTypeHandler(t *testing.T) {
	querier := &mockServiceTypeQuerier{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	handler := NewServiceTypeHandler(querier, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, authz, handler.authz)
}

// TestServiceTypeHandlerRoutes tests that routes are properly registered
func TestServiceTypeHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := &mockServiceTypeQuerier{}
	authz := &MockAuthorizer{ShouldSucceed: true}

	// Create the handler
	handler := NewServiceTypeHandler(querier, authz)

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
		case method == "POST" && route == "/{id}/validate":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestServiceTypeToResponse tests the serviceTypeToResponse function
func TestServiceTypeToResponse(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	// Create a service type
	serviceType := &domain.ServiceType{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name: "VM Instance",
	}

	response := serviceTypeToResponse(serviceType)

	// Verify all fields are correctly mapped
	assert.Equal(t, serviceType.ID, response.ID)
	assert.Equal(t, serviceType.Name, response.Name)
	assert.Equal(t, JSONUTCTime(serviceType.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(serviceType.UpdatedAt), response.UpdatedAt)
}
