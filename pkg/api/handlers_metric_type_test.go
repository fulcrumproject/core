package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	authmocks "github.com/fulcrumproject/core/pkg/auth/mocks"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/domain/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewMetricTypeHandler tests the constructor
func TestNewMetricTypeHandler(t *testing.T) {
	querier := mocks.NewMockMetricTypeQuerier(t)
	commander := mocks.NewMockMetricTypeCommander(t)
	authz := authmocks.NewMockAuthorizer(t)

	handler := NewMetricTypeHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestMetricTypeHandlerRoutes tests that routes are properly registered
func TestMetricTypeHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := mocks.NewMockMetricTypeQuerier(t)
	commander := mocks.NewMockMetricTypeCommander(t)
	authz := authmocks.NewMockAuthorizer(t)

	// Create the handler
	handler := NewMetricTypeHandler(querier, commander, authz)

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

// TestMetricTypeToResponse tests the metricTypeToResponse function
func TestMetricTypeToResponse(t *testing.T) {
	// Create a metric type
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	metricType := &domain.MetricType{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:       "CPU Usage",
		EntityType: domain.MetricEntityType("service"),
	}

	// Convert to response
	response := MetricTypeToRes(metricType)

	// Verify all fields are correctly mapped
	assert.Equal(t, id, response.ID)
	assert.Equal(t, "CPU Usage", response.Name)
	assert.Equal(t, domain.MetricEntityType("service"), response.EntityType)
	assert.Equal(t, JSONUTCTime(createdAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), response.UpdatedAt)
}
