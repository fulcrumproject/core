package api

import (
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
)

type AgentTypeHandler struct {
	querier domain.AgentTypeQuerier
	authz   auth.Authorizer
}

func NewAgentTypeHandler(
	querier domain.AgentTypeQuerier,
	authz auth.Authorizer,
) *AgentTypeHandler {
	return &AgentTypeHandler{
		querier: querier,
		authz:   authz,
	}
}

// Routes returns the router with all agent type routes registered
func (h *AgentTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeAgentType, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, AgentTypeToRes))

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using agent type's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgentType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier, AgentTypeToRes))
		})
	}
}

// AgentTypeRes represents the response body for agent type operations
type AgentTypeRes struct {
	ID           properties.UUID   `json:"id"`
	Name         string            `json:"name"`
	CreatedAt    JSONUTCTime       `json:"createdAt"`
	UpdatedAt    JSONUTCTime       `json:"updatedAt"`
	ServiceTypes []*ServiceTypeRes `json:"serviceTypes"`
}

// AgentTypeToRes converts a domain.AgentType to an AgentTypeResponse
func AgentTypeToRes(at *domain.AgentType) *AgentTypeRes {
	response := &AgentTypeRes{
		ID:           at.ID,
		Name:         at.Name,
		CreatedAt:    JSONUTCTime(at.CreatedAt),
		UpdatedAt:    JSONUTCTime(at.UpdatedAt),
		ServiceTypes: make([]*ServiceTypeRes, 0),
	}
	for _, st := range at.ServiceTypes {
		response.ServiceTypes = append(response.ServiceTypes, ServiceTypeToRes(&st))
	}
	return response
}
