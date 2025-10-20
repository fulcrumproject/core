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

// TestNewServicePoolValueHandler tests the constructor
func TestNewServicePoolValueHandler(t *testing.T) {
	querier := domain.NewMockServicePoolValueQuerier(t)
	commander := domain.NewMockServicePoolValueCommander(t)
	authz := auth.NewMockAuthorizer(t)

	handler := NewServicePoolValueHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestServicePoolValueHandlerRoutes tests that routes are properly registered
func TestServicePoolValueHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := domain.NewMockServicePoolValueQuerier(t)
	commander := domain.NewMockServicePoolValueCommander(t)
	authz := auth.NewMockAuthorizer(t)

	// Create the handler
	handler := NewServicePoolValueHandler(querier, commander, authz)

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
		case method == "DELETE" && route == "/{id}":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

// TestServicePoolValueToRes tests the ServicePoolValueToRes function for unallocated value
func TestServicePoolValueToRes(t *testing.T) {
	// Create a service pool value (unallocated)
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	poolID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	value := &domain.ServicePoolValue{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:          "IP-001",
		Value:         "192.168.1.10",
		ServicePoolID: properties.UUID(poolID),
		ServiceID:     nil,
		PropertyName:  nil,
		AllocatedAt:   nil,
	}

	// Convert to response
	res := ServicePoolValueToRes(value)

	// Verify
	assert.Equal(t, properties.UUID(id), res.ID)
	assert.Equal(t, "IP-001", res.Name)
	assert.Equal(t, "192.168.1.10", res.Value)
	assert.Equal(t, properties.UUID(poolID), res.ServicePoolID)
	assert.Nil(t, res.ServiceID)
	assert.Nil(t, res.PropertyName)
	assert.Nil(t, res.AllocatedAt)
	assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
}

// TestServicePoolValueToRes_Allocated tests ServicePoolValueToRes for allocated value
func TestServicePoolValueToRes_Allocated(t *testing.T) {
	// Create a service pool value (allocated)
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	poolID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	serviceID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	propertyName := "publicIp"
	allocatedAt := time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC)
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	value := &domain.ServicePoolValue{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:          "IP-001",
		Value:         "192.168.1.10",
		ServicePoolID: properties.UUID(poolID),
		ServiceID:     (*properties.UUID)(&serviceID),
		PropertyName:  &propertyName,
		AllocatedAt:   &allocatedAt,
	}

	// Convert to response
	res := ServicePoolValueToRes(value)

	// Verify
	assert.Equal(t, properties.UUID(id), res.ID)
	assert.Equal(t, "IP-001", res.Name)
	assert.Equal(t, "192.168.1.10", res.Value)
	assert.Equal(t, properties.UUID(poolID), res.ServicePoolID)
	assert.NotNil(t, res.ServiceID)
	assert.Equal(t, properties.UUID(serviceID), *res.ServiceID)
	assert.NotNil(t, res.PropertyName)
	assert.Equal(t, "publicIp", *res.PropertyName)
	assert.NotNil(t, res.AllocatedAt)
	assert.Equal(t, JSONUTCTime(allocatedAt), *res.AllocatedAt)
	assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
}

// TestServicePoolValueToRes_ComplexValue tests ServicePoolValueToRes with complex JSON value
func TestServicePoolValueToRes_ComplexValue(t *testing.T) {
	// Create a service pool value with complex JSON value
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	poolID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	complexValue := map[string]any{
		"ip":      "192.168.1.10",
		"gateway": "192.168.1.1",
		"netmask": "255.255.255.0",
	}

	value := &domain.ServicePoolValue{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:          "IP-001",
		Value:         complexValue,
		ServicePoolID: properties.UUID(poolID),
		ServiceID:     nil,
		PropertyName:  nil,
		AllocatedAt:   nil,
	}

	// Convert to response
	res := ServicePoolValueToRes(value)

	// Verify
	assert.Equal(t, properties.UUID(id), res.ID)
	assert.Equal(t, "IP-001", res.Name)
	assert.Equal(t, complexValue, res.Value)
	assert.Equal(t, properties.UUID(poolID), res.ServicePoolID)
	assert.Nil(t, res.ServiceID)
	assert.Nil(t, res.PropertyName)
	assert.Nil(t, res.AllocatedAt)
	assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
}
