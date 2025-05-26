package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ServiceTypeHandler struct {
	querier domain.ServiceTypeQuerier
	authz   domain.Authorizer
}

func NewServiceTypeHandler(
	querier domain.ServiceTypeQuerier,
	authz domain.Authorizer,
) *ServiceTypeHandler {
	return &ServiceTypeHandler{
		querier: querier,
		authz:   authz,
	}
}

// Routes returns the router with all service type routes registered
func (h *ServiceTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			AuthzSimple(domain.SubjectServiceType, domain.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(ID)

			// Get endpoint - authorize using service type's scope
			r.With(
				AuthzFromID(domain.SubjectServiceType, domain.ActionRead, h.authz, h.querier),
			).Get("/{id}", h.handleGet)
		})
	}
}

func (h *ServiceTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	serviceType, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, serviceTypeToResponse(serviceType))
}

func (h *ServiceTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
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
