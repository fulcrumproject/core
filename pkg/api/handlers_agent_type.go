package api

import (
	"net/http"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/middlewares"
	"github.com/fulcrumproject/commons/properties"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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
		).Get("/", h.handleList)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using agent type's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgentType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", h.handleGet)
		})
	}
}

func (h *AgentTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	agentType, err := h.querier.Get(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, agentTypeToResponse(agentType))
}

func (h *AgentTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := auth.MustGetIdentity(r.Context())
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	result, err := h.querier.List(r.Context(), &id.Scope, pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, agentTypeToResponse))
}

// AgentTypeResponse represents the response body for agent type operations
type AgentTypeResponse struct {
	ID           properties.UUID        `json:"id"`
	Name         string                 `json:"name"`
	CreatedAt    JSONUTCTime            `json:"createdAt"`
	UpdatedAt    JSONUTCTime            `json:"updatedAt"`
	ServiceTypes []*ServiceTypeResponse `json:"serviceTypes"`
}

// agentTypeToResponse converts a domain.AgentType to an AgentTypeResponse
func agentTypeToResponse(at *domain.AgentType) *AgentTypeResponse {
	response := &AgentTypeResponse{
		ID:           at.ID,
		Name:         at.Name,
		CreatedAt:    JSONUTCTime(at.CreatedAt),
		UpdatedAt:    JSONUTCTime(at.UpdatedAt),
		ServiceTypes: make([]*ServiceTypeResponse, 0),
	}
	for _, st := range at.ServiceTypes {
		response.ServiceTypes = append(response.ServiceTypes, serviceTypeToResponse(&st))
	}
	return response
}
