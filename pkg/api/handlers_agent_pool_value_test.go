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

func TestNewAgentPoolValueHandler(t *testing.T) {
	querier := domain.NewMockAgentPoolValueQuerier(t)
	commander := domain.NewMockAgentPoolValueCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewAgentPoolValueHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

func TestAgentPoolValueHandlerRoutes(t *testing.T) {
	querier := domain.NewMockAgentPoolValueQuerier(t)
	commander := domain.NewMockAgentPoolValueCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewAgentPoolValueHandler(querier, commander, authz)

	routeFunc := handler.Routes()
	assert.NotNil(t, routeFunc)

	r := chi.NewRouter()
	routeFunc(r)

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
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

func TestAgentPoolValueToRes(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	poolID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440000")
	agentID := uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	allocatedAt := time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)
	propName := "ip_address"

	tests := []struct {
		name             string
		value            *domain.AgentPoolValue
		expectedAgentID  *properties.UUID
		expectedAllocAt  *JSONUTCTime
		expectedPropName *string
	}{
		{
			name: "not allocated",
			value: &domain.AgentPoolValue{
				BaseEntity:  domain.BaseEntity{ID: properties.UUID(id), CreatedAt: createdAt, UpdatedAt: updatedAt},
				Name:        "value-1",
				Value:       "192.168.1.1",
				AgentPoolID: properties.UUID(poolID),
			},
			expectedAgentID:  nil,
			expectedAllocAt:  nil,
			expectedPropName: nil,
		},
		{
			name: "allocated",
			value: &domain.AgentPoolValue{
				BaseEntity:   domain.BaseEntity{ID: properties.UUID(id), CreatedAt: createdAt, UpdatedAt: updatedAt},
				Name:         "value-2",
				Value:        "192.168.1.2",
				AgentPoolID:  properties.UUID(poolID),
				AgentID:      (*properties.UUID)(&agentID),
				PropertyName: &propName,
				AllocatedAt:  &allocatedAt,
			},
			expectedAgentID:  (*properties.UUID)(&agentID),
			expectedAllocAt:  jsonUTCTimePtr(allocatedAt),
			expectedPropName: &propName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := AgentPoolValueToRes(tt.value)

			assert.Equal(t, properties.UUID(id), res.ID)
			assert.Equal(t, tt.value.Name, res.Name)
			assert.Equal(t, tt.value.Value, res.Value)
			assert.Equal(t, properties.UUID(poolID), res.AgentPoolID)
			assert.Equal(t, tt.expectedAgentID, res.AgentID)
			assert.Equal(t, tt.expectedPropName, res.PropertyName)
			assert.Equal(t, tt.expectedAllocAt, res.AllocatedAt)
			assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
			assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
		})
	}
}

func jsonUTCTimePtr(t time.Time) *JSONUTCTime {
	j := JSONUTCTime(t)
	return &j
}
