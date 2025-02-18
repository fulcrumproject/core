package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ServiceTypeResponse represents the response body for service type operations
type ServiceTypeResponse struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	CreatedAt JSONUTCTime `json:"createdAt"`
	UpdatedAt JSONUTCTime `json:"updatedAt"`
}

// serviceTypeToResponse converts a domain.ServiceType to a ServiceTypeResponse
func serviceTypeToResponse(st *domain.ServiceType) *ServiceTypeResponse {
	return &ServiceTypeResponse{
		ID:        st.ID.String(),
		Name:      st.Name,
		CreatedAt: JSONUTCTime(st.CreatedAt),
		UpdatedAt: JSONUTCTime(st.UpdatedAt),
	}
}

type ServiceTypeHandler struct {
	repo domain.ServiceTypeRepository
}

func NewServiceTypeHandler(repo domain.ServiceTypeRepository) *ServiceTypeHandler {
	return &ServiceTypeHandler{repo: repo}
}

// Routes returns the router with all service type routes registered
func (h *ServiceTypeHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware for all service type routes
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", h.handleList)
	r.Get("/{id}", h.handleGet)

	return r
}

func (h *ServiceTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	serviceType, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, serviceTypeToResponse(serviceType))
}

func (h *ServiceTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	filter := parseSimpleFilter(r)
	sorting := parseSorting(r)
	pagination := parsePagination(r)

	result, err := h.repo.List(r.Context(), filter, sorting, pagination)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPaginatedResponse(result, serviceTypeToResponse))
}
