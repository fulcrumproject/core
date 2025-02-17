package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// AgentTypeGetResponse represents the detailed response for single agent type operations
type AgentTypeGetResponse struct {
	AgentTypeListResponse
	ServiceTypes []AgentTypeServiceTypeResponse `json:"serviceTypes"`
}

type AgentTypeServiceTypeResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// AgentTypeListResponse represents the response body for agent type operations
type AgentTypeListResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// agentTypeToResponse converts a domain.AgentType to an AgentTypeResponse
func agentTypeToResponse(at *domain.AgentType) *AgentTypeListResponse {
	return &AgentTypeListResponse{
		ID:        uuid.UUID(at.ID).String(),
		Name:      at.Name,
		CreatedAt: at.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: at.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// agentTypeToDetailResponse converts a domain.AgentType to an AgentTypeDetailResponse
func agentTypeToDetailResponse(at *domain.AgentType) *AgentTypeGetResponse {
	response := &AgentTypeGetResponse{
		AgentTypeListResponse: *agentTypeToResponse(at),
		ServiceTypes:          make([]AgentTypeServiceTypeResponse, len(at.ServiceTypes)),
	}

	for i, st := range at.ServiceTypes {
		response.ServiceTypes[i] = AgentTypeServiceTypeResponse{
			ID:        uuid.UUID(st.ID).String(),
			Name:      st.Name,
			CreatedAt: st.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: st.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
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
	id, err := domain.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agentType, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, agentTypeToDetailResponse(agentType))
}

func (h *AgentTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	// Parse request parameters using shared utilities
	filters := ParseFilters(r, []FilterConfig{
		{
			Param: "name",
			Query: "name ? LIKE ",
		},
	})
	sorting := ParseSorting(r)
	pagination := ParsePagination(r)

	result, err := h.repo.List(r.Context(), filters, sorting, pagination)
	if err != nil {
		render.Render(w, r, ErrInternalServer(err))
		return
	}

	response := make([]*AgentTypeListResponse, len(result.Items))
	for i, agentType := range result.Items {
		response[i] = agentTypeToResponse(&agentType)
	}

	render.JSON(w, r, &PaginatedResponse[*AgentTypeListResponse]{
		Items:       response,
		TotalItems:  result.TotalItems,
		TotalPages:  result.TotalPages,
		CurrentPage: result.CurrentPage,
		HasNext:     result.HasNext,
		HasPrev:     result.HasPrev,
	})
}
