package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ServiceTypeHandler struct {
	querier domain.ServiceTypeQuerier
}

func NewServiceTypeHandler(repo domain.ServiceTypeRepository) *ServiceTypeHandler {
	return &ServiceTypeHandler{querier: repo}
}

// Routes returns the router with all service type routes registered
func (h *ServiceTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Group(func(r chi.Router) {
			r.Use(UUIDMiddleware)
			r.Get("/{id}", h.handleGet)
		})
	}
}

func (h *ServiceTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
	serviceType, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, serviceTypeToResponse(serviceType))
}

func (h *ServiceTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
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
	render.JSON(w, r, NewPageResponse(result, serviceTypeToResponse))
}

// ServiceTypeResponse represents the response body for service type operations
type ServiceTypeResponse struct {
	ID        domain.UUID `json:"id"`
	Name      string      `json:"name"`
	CreatedAt JSONUTCTime `json:"createdAt"`
	UpdatedAt JSONUTCTime `json:"updatedAt"`
}

// serviceTypeToResponse converts a domain.ServiceType to a ServiceTypeResponse
func serviceTypeToResponse(st *domain.ServiceType) *ServiceTypeResponse {
	return &ServiceTypeResponse{
		ID:        st.ID,
		Name:      st.Name,
		CreatedAt: JSONUTCTime(st.CreatedAt),
		UpdatedAt: JSONUTCTime(st.UpdatedAt),
	}
}
