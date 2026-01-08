package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewServicePoolHandler tests the constructor
func TestNewServicePoolHandler(t *testing.T) {
	querier := domain.NewMockServicePoolQuerier(t)
	commander := domain.NewMockServicePoolCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewServicePoolHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestServicePoolHandlerRoutes tests that routes are properly registered
func TestServicePoolHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := domain.NewMockServicePoolQuerier(t)
	commander := domain.NewMockServicePoolCommander(t)
	authz := authz.NewMockAuthorizer(t)

	// Create the handler
	handler := NewServicePoolHandler(querier, commander, authz)

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

// TestServicePoolToRes tests the ServicePoolToRes function
func TestServicePoolToRes(t *testing.T) {
	// Create a service pool
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	poolSetID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	config := properties.JSON{"cidr": "192.168.1.0/24", "excludeFirst": 1, "excludeLast": 1}

	pool := &domain.ServicePool{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:             "Public IPs",
		Type:             "public_ip",
		GeneratorType:    domain.PoolGeneratorSubnet,
		GeneratorConfig:  &config,
		ServicePoolSetID: properties.UUID(poolSetID),
	}

	// Convert to response
	res := ServicePoolToRes(pool)

	// Verify
	assert.Equal(t, properties.UUID(id), res.ID)
	assert.Equal(t, "Public IPs", res.Name)
	assert.Equal(t, "public_ip", res.Type)
	assert.Equal(t, domain.PoolGeneratorSubnet, res.GeneratorType)
	assert.Equal(t, &config, res.GeneratorConfig)
	assert.Equal(t, properties.UUID(poolSetID), res.ServicePoolSetID)
	assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
}

// TestServicePoolToRes_NilConfig tests ServicePoolToRes with nil generator config
func TestServicePoolToRes_NilConfig(t *testing.T) {
	// Create a service pool with nil config (list type)
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	poolSetID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	pool := &domain.ServicePool{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:             "Operating Systems",
		Type:             "operating_system",
		GeneratorType:    domain.PoolGeneratorList,
		GeneratorConfig:  nil,
		ServicePoolSetID: properties.UUID(poolSetID),
	}

	// Convert to response
	res := ServicePoolToRes(pool)

	// Verify
	assert.Equal(t, properties.UUID(id), res.ID)
	assert.Equal(t, "Operating Systems", res.Name)
	assert.Equal(t, "operating_system", res.Type)
	assert.Equal(t, domain.PoolGeneratorList, res.GeneratorType)
	assert.Nil(t, res.GeneratorConfig)
	assert.Equal(t, properties.UUID(poolSetID), res.ServicePoolSetID)
	assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
}
