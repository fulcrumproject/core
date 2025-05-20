package api

import (
	"context"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ServiceGroupHandler struct {
	querier   domain.ServiceGroupQuerier
	commander domain.ServiceGroupCommander
	authz     domain.Authorizer
}

func NewServiceGroupHandler(
	querier domain.ServiceGroupQuerier,
	commander domain.ServiceGroupCommander,
	authz domain.Authorizer,
) *ServiceGroupHandler {
	return &ServiceGroupHandler{
		commander: commander,
		querier:   querier,
		authz:     authz,
	}
}

// Routes returns the router with all service group routes registered
func (h *ServiceGroupHandler) Routes() func(r chi.Router) {

	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(IDMiddleware)
			r.Get("/{id}", h.handleGet)
			r.Patch("/{id}", h.handleUpdate)
			r.Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *ServiceGroupHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Name       string      `json:"name"`
		ConsumerID domain.UUID `json:"consumerId"`
	}
	if err := render.Decode(r, &q); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	scope := domain.AuthScope{ConsumerID: &q.ConsumerID}
	if err := h.authz.AuthorizeCtx(r.Context(), domain.SubjectServiceGroup, domain.ActionCreate, &scope); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	sg, err := h.commander.Create(r.Context(), q.Name, q.ConsumerID)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serviceGroupToResponse(sg))
}

func (h *ServiceGroupHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionRead)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	serviceGroup, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, serviceGroupToResponse(serviceGroup))
}

func (h *ServiceGroupHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
	if err := h.authz.Authorize(id, domain.SubjectServiceGroup, domain.ActionRead, &domain.EmptyAuthScope); err != nil {
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
	render.JSON(w, r, NewPageResponse(result, serviceGroupToResponse))
}

func (h *ServiceGroupHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionUpdate)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	var req struct {
		Name *string `json:"name"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	sg, err := h.commander.Update(r.Context(), id, req.Name)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, serviceGroupToResponse(sg))
}

func (h *ServiceGroupHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionDelete)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ServiceGroupResponse represents the response body for service group operations
type ServiceGroupResponse struct {
	ID        domain.UUID `json:"id"`
	Name      string      `json:"name"`
	CreatedAt JSONUTCTime `json:"createdAt"`
	UpdatedAt JSONUTCTime `json:"updatedAt"`
}

// serviceGroupToResponse converts a domain.ServiceGroup to a ServiceGroupResponse
func serviceGroupToResponse(sg *domain.ServiceGroup) *ServiceGroupResponse {
	return &ServiceGroupResponse{
		ID:        sg.ID,
		Name:      sg.Name,
		CreatedAt: JSONUTCTime(sg.CreatedAt),
		UpdatedAt: JSONUTCTime(sg.UpdatedAt),
	}
}

func (h *ServiceGroupHandler) authorize(ctx context.Context, id domain.UUID, action domain.AuthAction) (*domain.AuthScope, error) {
	scope, err := h.querier.AuthScope(ctx, id)
	if err != nil {
		return nil, err
	}
	err = h.authz.AuthorizeCtx(ctx, domain.SubjectServiceGroup, action, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}
