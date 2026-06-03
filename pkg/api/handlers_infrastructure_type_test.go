package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewInfrastructureTypeHandler(t *testing.T) {
	querier := domain.NewMockInfrastructureTypeQuerier(t)
	commander := domain.NewMockInfrastructureTypeCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewInfrastructureTypeHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

func TestInfrastructureTypeHandlerRoutes(t *testing.T) {
	querier := domain.NewMockInfrastructureTypeQuerier(t)
	commander := domain.NewMockInfrastructureTypeCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewInfrastructureTypeHandler(querier, commander, authz)
	routeFunc := handler.Routes()
	assert.NotNil(t, routeFunc)

	r := chi.NewRouter()
	routeFunc(r)

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
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
	assert.NoError(t, chi.Walk(r, walkFunc))
}

func TestInfrastructureTypeHandlerCreate(t *testing.T) {
	commander := domain.NewMockInfrastructureTypeCommander(t)
	handler := &InfrastructureTypeHandler{commander: commander}

	req := &CreateInfrastructureTypeReq{
		Name: "Test Infra Type",
		ConfigurationSchema: schema.Schema{
			Properties: map[string]schema.PropertyDefinition{
				"endpoint": {Type: "string", Required: true},
			},
		},
		ConfigContentType: "text/yaml",
	}

	commander.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(params domain.CreateInfrastructureTypeParams) bool {
			return params.Name == "Test Infra Type" &&
				params.ConfigContentType == "text/yaml" &&
				len(params.ConfigurationSchema.Properties) == 1
		})).
		Return(&domain.InfrastructureType{
			BaseEntity: domain.BaseEntity{ID: uuid.New()},
			Name:       "Test Infra Type",
		}, nil)

	ctx := auth.WithIdentity(context.Background(), &auth.Identity{
		ID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Name: "test-admin",
		Role: auth.RoleAdmin,
	})
	result, err := handler.Create(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInfrastructureTypeHandlerUpdate(t *testing.T) {
	commander := domain.NewMockInfrastructureTypeCommander(t)
	handler := &InfrastructureTypeHandler{commander: commander}

	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	name := "Renamed"
	req := &UpdateInfrastructureTypeReq{Name: &name}

	commander.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(params domain.UpdateInfrastructureTypeParams) bool {
			return params.ID == id && params.Name != nil && *params.Name == "Renamed"
		})).
		Return(&domain.InfrastructureType{
			BaseEntity: domain.BaseEntity{ID: id},
			Name:       "Renamed",
		}, nil)

	ctx := auth.WithIdentity(context.Background(), &auth.Identity{
		ID:   uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
		Name: "test-admin",
		Role: auth.RoleAdmin,
	})
	result, err := handler.Update(ctx, id, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInfrastructureTypeToRes(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	it := &domain.InfrastructureType{
		BaseEntity: domain.BaseEntity{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name: "InfraType",
		TemplateValidation: domain.TemplateValidation{
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint": {Type: "string"},
				},
			},
			ConfigTemplate:    "endpoint={{.endpoint}}",
			CmdTemplate:       "curl {{.configUrl}}",
			ConfigContentType: "text/yaml",
		},
	}

	res := InfrastructureTypeToRes(it)
	assert.Equal(t, it.ID, res.ID)
	assert.Equal(t, it.Name, res.Name)
	assert.Equal(t, JSONUTCTime(it.CreatedAt), res.CreatedAt)
	assert.Equal(t, JSONUTCTime(it.UpdatedAt), res.UpdatedAt)
	assert.Equal(t, it.ConfigTemplate, res.ConfigTemplate)
	assert.Equal(t, it.CmdTemplate, res.CmdTemplate)
	assert.Equal(t, it.ConfigContentType, res.ConfigContentType)
	assert.Len(t, res.ConfigurationSchema.Properties, 1)
}
