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

func TestNewAgentPoolHandler(t *testing.T) {
	querier := domain.NewMockAgentPoolQuerier(t)
	commander := domain.NewMockAgentPoolCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewAgentPoolHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

func TestAgentPoolHandlerRoutes(t *testing.T) {
	querier := domain.NewMockAgentPoolQuerier(t)
	commander := domain.NewMockAgentPoolCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewAgentPoolHandler(querier, commander, authz)

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

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

func TestAgentPoolToRes(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	config := properties.JSON{"cidr": "192.168.1.0/24"}

	tests := []struct {
		name            string
		pool            *domain.AgentPool
		expectedName    string
		expectedType    string
		expectedGenType domain.PoolGeneratorType
		expectedConfig  *properties.JSON
	}{
		{
			name: "with config",
			pool: &domain.AgentPool{
				BaseEntity:      domain.BaseEntity{ID: id, CreatedAt: createdAt, UpdatedAt: updatedAt},
				Name:            "Public IPs",
				Type:            "public_ip",
				PropertyType:    "string",
				GeneratorType:   domain.PoolGeneratorSubnet,
				GeneratorConfig: &config,
			},
			expectedName:    "Public IPs",
			expectedType:    "public_ip",
			expectedGenType: domain.PoolGeneratorSubnet,
			expectedConfig:  &config,
		},
		{
			name: "nil config",
			pool: &domain.AgentPool{
				BaseEntity:    domain.BaseEntity{ID: id, CreatedAt: createdAt, UpdatedAt: updatedAt},
				Name:          "Operating Systems",
				Type:          "operating_system",
				PropertyType:  "string",
				GeneratorType: domain.PoolGeneratorList,
			},
			expectedName:    "Operating Systems",
			expectedType:    "operating_system",
			expectedGenType: domain.PoolGeneratorList,
			expectedConfig:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := AgentPoolToRes(tt.pool)

			assert.Equal(t, properties.UUID(id), res.ID)
			assert.Equal(t, tt.expectedName, res.Name)
			assert.Equal(t, tt.expectedType, res.Type)
			assert.Equal(t, "string", res.PropertyType)
			assert.Equal(t, tt.expectedGenType, res.GeneratorType)
			assert.Equal(t, tt.expectedConfig, res.GeneratorConfig)
			assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
			assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
		})
	}
}
