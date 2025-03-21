package api

import (
	"context"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type AgentTypeHandler struct {
	querier domain.AgentTypeQuerier
	authz   domain.Authorizer
}

func NewAgentTypeHandler(
	querier domain.AgentTypeQuerier,
	authz domain.Authorizer,
) *AgentTypeHandler {
	return &AgentTypeHandler{
		querier: querier,
		authz:   authz,
	}
}

// Routes returns the router with all agent type routes registered
func (h *AgentTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Group(func(r chi.Router) {
			r.Use(IDMiddleware)
			r.Get("/{id}", h.handleGet)
		})

	}
}

func (h *AgentTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionRead)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	agentType, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, agentTypeToResponse(agentType))
}

func (h *AgentTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
	if err := h.authz.Authorize(id, domain.SubjectServiceType, domain.ActionRead, &domain.EmptyAuthScope); err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	result, err := h.querier.List(r.Context(), id.Scope(), pag)
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

func (h *AgentTypeHandler) authorize(ctx context.Context, id domain.UUID, action domain.AuthAction) (*domain.AuthScope, error) {
	scope, err := h.querier.AuthScope(ctx, id)
	if err != nil {
		return nil, err
	}
	err = h.authz.AuthorizeCtx(ctx, domain.SubjectAgentType, action, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}
