package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	authmocks "github.com/fulcrumproject/core/pkg/auth/mocks"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/domain/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestNewServiceTypeHandler tests the constructor
func TestNewServiceTypeHandler(t *testing.T) {
	querier := mocks.NewMockServiceTypeQuerier(t)
	commander := mocks.NewMockServiceTypeCommander(t)
	authz := authmocks.NewMockAuthorizer(t)

	handler := NewServiceTypeHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

// TestServiceTypeHandlerRoutes tests that routes are properly registered
func TestServiceTypeHandlerRoutes(t *testing.T) {
	// Create mocks
	querier := mocks.NewMockServiceTypeQuerier(t)
	commander := mocks.NewMockServiceTypeCommander(t)
	authz := authmocks.NewMockAuthorizer(t)

	// Create the handler
	handler := NewServiceTypeHandler(querier, commander, authz)

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

	response := ServiceTypeToRes(serviceType)

	// Verify all fields are correctly mapped
	assert.Equal(t, serviceType.ID, response.ID)
	assert.Equal(t, serviceType.Name, response.Name)
	assert.Equal(t, JSONUTCTime(serviceType.CreatedAt), response.CreatedAt)
	assert.Equal(t, JSONUTCTime(serviceType.UpdatedAt), response.UpdatedAt)
}

// TestServiceTypeHandlerCreate tests the Create adapter function
func TestServiceTypeHandlerCreate(t *testing.T) {
	commander := mocks.NewMockServiceTypeCommander(t)
	handler := &ServiceTypeHandler{commander: commander}

	req := &CreateServiceTypeReq{
		Name: "Test Service Type",
	}

	// Set up mock expectation
	commander.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(params domain.CreateServiceTypeParams) bool {
			return params.Name == "Test Service Type"
		})).
		Return(&domain.ServiceType{
			BaseEntity: domain.BaseEntity{ID: uuid.New()},
			Name:       "Test Service Type",
		}, nil)

	ctx := auth.WithIdentity(context.Background(), &auth.Identity{
		ID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name: "test-admin",
		Role: auth.RoleAdmin,
	})
	result, err := handler.Create(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestServiceTypeHandlerUpdate tests the Update adapter function
func TestServiceTypeHandlerUpdate(t *testing.T) {
	commander := mocks.NewMockServiceTypeCommander(t)
	handler := &ServiceTypeHandler{commander: commander}

	serviceTypeID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	name := "Updated Service Type"
	req := &UpdateServiceTypeReq{
		Name: &name,
	}

	// Set up mock expectation
	commander.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(params domain.UpdateServiceTypeParams) bool {
			return params.ID == serviceTypeID && params.Name != nil && *params.Name == "Updated Service Type"
		})).
		Return(&domain.ServiceType{
			BaseEntity: domain.BaseEntity{ID: serviceTypeID},
			Name:       "Updated Service Type",
		}, nil)

	ctx := auth.WithIdentity(context.Background(), &auth.Identity{
		ID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name: "test-admin",
		Role: auth.RoleAdmin,
	})
	result, err := handler.Update(ctx, serviceTypeID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}
