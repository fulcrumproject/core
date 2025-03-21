package api

import (
	"context"
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
		r.Get("/", h.handleList)
		r.Group(func(r chi.Router) {
			r.Use(IDMiddleware)
			r.Get("/{id}", h.handleGet)
		})
	}
}

func (h *ServiceTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionRead)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	serviceType, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, serviceTypeToResponse(serviceType))
}

func (h *ServiceTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
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

func (h *ServiceTypeHandler) authorize(ctx context.Context, id domain.UUID, action domain.AuthAction) (*domain.AuthScope, error) {
	scope, err := h.querier.AuthScope(ctx, id)
	if err != nil {
		return nil, err
	}
	err = h.authz.AuthorizeCtx(ctx, domain.SubjectServiceType, action, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}
