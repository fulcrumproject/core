package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type AgentTypeHandler struct {
	querier domain.AgentTypeQuerier
}

func NewAgentTypeHandler(repo domain.AgentTypeRepository) *AgentTypeHandler {
	return &AgentTypeHandler{querier: repo}
}

// Routes returns the router with all agent type routes registered
func (h *AgentTypeHandler) Routes(authzMW AuthzMiddlewareFunc) func(r chi.Router) {
	return func(r chi.Router) {
		r.With(authzMW(domain.SubjectAgentType, domain.ActionList)).Get("/", h.handleList)
		r.Group(func(r chi.Router) {
			r.Use(UUIDMiddleware)
			r.With(authzMW(domain.SubjectAgentType, domain.ActionRead)).Get("/{id}", h.handleGet)
		})

	}
}

func (h *AgentTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
	agentType, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, agentTypeToResponse(agentType))
}

func (h *AgentTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	result, err := h.querier.List(r.Context(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, NewPageResponse(result, agentTypeToResponse))
}

// AgentTypeResponse represents the response body for agent type operations
type AgentTypeResponse struct {
	ID           domain.UUID            `json:"id"`
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
