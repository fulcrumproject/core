package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

// ServiceTypeResponse represents the response body for service type operations
type ServiceTypeResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// serviceTypeToResponse converts a domain.ServiceType to a ServiceTypeResponse
func serviceTypeToResponse(st *domain.ServiceType) *ServiceTypeResponse {
	return &ServiceTypeResponse{
		ID:        uuid.UUID(st.ID).String(),
		Name:      st.Name,
		CreatedAt: st.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: st.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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
	id, err := domain.ParseID(chi.URLParam(r, "id"))
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
	// Parse request parameters using shared utilities
	filters := ParseFilters(r, []FilterConfig{
		{
			Field:      "name",
			ExactMatch: false,
		},
	})
	sorting := ParseSorting(r)
	pagination := ParsePagination(r)

	result, err := h.repo.List(r.Context(), filters, sorting, pagination)
	if err != nil {
		render.Render(w, r, ErrInternalServer(err))
		return
	}

	response := make([]*ServiceTypeResponse, len(result.Items))
	for i, serviceType := range result.Items {
		response[i] = serviceTypeToResponse(&serviceType)
	}

	render.JSON(w, r, &PaginatedResponse[*ServiceTypeResponse]{
		Items:       response,
		TotalItems:  result.TotalItems,
		TotalPages:  result.TotalPages,
		CurrentPage: result.CurrentPage,
		HasNext:     result.HasNext,
		HasPrev:     result.HasPrev,
	})
}
