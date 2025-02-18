package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// AgentTypeResponse represents the response body for agent type operations
type AgentTypeResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	CreatedAt    JSONUTCTime            `json:"createdAt"`
	UpdatedAt    JSONUTCTime            `json:"updatedAt"`
	ServiceTypes []*ServiceTypeResponse `json:"serviceTypes"`
}

// agentTypeToResponse converts a domain.AgentType to an AgentTypeResponse
func agentTypeToResponse(at *domain.AgentType) *AgentTypeResponse {
	response := &AgentTypeResponse{
		ID:           at.ID.String(),
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

type AgentTypeHandler struct {
	repo domain.AgentTypeRepository
}

func NewAgentTypeHandler(repo domain.AgentTypeRepository) *AgentTypeHandler {
	return &AgentTypeHandler{repo: repo}
}

// Routes returns the router with all agent type routes registered
func (h *AgentTypeHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware for all agent type routes
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", h.handleList)
	r.Get("/{id}", h.handleGet)

	return r
}

func (h *AgentTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agentType, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, agentTypeToResponse(agentType))
}

func (h *AgentTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	filter := parseSimpleFilter(r) // name
	sorting := parseSorting(r)
	pagination := parsePagination(r)

	result, err := h.repo.List(r.Context(), filter, sorting, pagination)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPaginatedResponse(result, agentTypeToResponse))
}
