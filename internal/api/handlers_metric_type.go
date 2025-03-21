package api

import (
	"context"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type MetricTypeHandler struct {
	querier   domain.MetricTypeQuerier
	commander domain.MetricTypeCommander
	authz     domain.Authorizer
}

func NewMetricTypeHandler(
	querier domain.MetricTypeQuerier,
	commander domain.MetricTypeCommander,
	authz domain.Authorizer,
) *MetricTypeHandler {
	return &MetricTypeHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all metric type routes registered
func (h *MetricTypeHandler) Routes() func(r chi.Router) {

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

func (h *MetricTypeHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string                  `json:"name"`
		EntityType domain.MetricEntityType `json:"entityType"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	if err := h.authz.AuthorizeCtx(r.Context(), domain.SubjectMetricType, domain.ActionCreate, &domain.EmptyAuthScope); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	metricType, err := h.commander.Create(r.Context(), req.Name, req.EntityType)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionRead)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	metricType, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
	if err := h.authz.Authorize(id, domain.SubjectMetricType, domain.ActionRead, &domain.EmptyAuthScope); err != nil {
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
	render.JSON(w, r, NewPageResponse(result, metricTypeToResponse))
}

func (h *MetricTypeHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionUpdate)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	var p struct {
		Name *string `json:"name"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	metricType, err := h.commander.Update(r.Context(), id, p.Name)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionDelete)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	_, err = h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// MetricTypeResponse represents the response body for metric type operations
type MetricTypeResponse struct {
	ID         domain.UUID             `json:"id"`
	Name       string                  `json:"name"`
	EntityType domain.MetricEntityType `json:"entityType"`
	CreatedAt  JSONUTCTime             `json:"createdAt"`
	UpdatedAt  JSONUTCTime             `json:"updatedAt"`
}

// metricTypeToResponse converts a domain.MetricType to a MetricTypeResponse
func metricTypeToResponse(mt *domain.MetricType) *MetricTypeResponse {
	return &MetricTypeResponse{
		ID:         mt.ID,
		Name:       mt.Name,
		EntityType: mt.EntityType,
		CreatedAt:  JSONUTCTime(mt.CreatedAt),
		UpdatedAt:  JSONUTCTime(mt.UpdatedAt),
	}
}

func (h *MetricTypeHandler) authorize(ctx context.Context, id domain.UUID, action domain.AuthAction) (*domain.AuthScope, error) {
	scope, err := h.querier.AuthScope(ctx, id)
	if err != nil {
		return nil, err
	}
	err = h.authz.AuthorizeCtx(ctx, domain.SubjectMetricType, action, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}
