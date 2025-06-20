package api

import (
	"net/http"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateMetricTypeRequest struct {
	Name       string                  `json:"name"`
	EntityType domain.MetricEntityType `json:"entityType"`
}

type UpdateMetricTypeRequest struct {
	Name *string `json:"name"`
}

type MetricTypeHandler struct {
	querier   domain.MetricTypeQuerier
	commander domain.MetricTypeCommander
	authz     auth.Authorizer
}

func NewMetricTypeHandler(
	querier domain.MetricTypeQuerier,
	commander domain.MetricTypeCommander,
	authz auth.Authorizer,
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
		// List metric types
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeMetricType, authz.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create metric type
		r.With(
			middlewares.DecodeBody[CreateMetricTypeRequest](),
			middlewares.AuthzSimple(authz.ObjectTypeMetricType, authz.ActionCreate, h.authz),
		).Post("/", h.handleCreate)

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get metric type
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeMetricType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", h.handleGet)

			// Update metric type
			r.With(
				middlewares.DecodeBody[UpdateMetricTypeRequest](),
				middlewares.AuthzFromID(authz.ObjectTypeMetricType, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", h.handleUpdate)

			// Delete metric type
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeMetricType, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *MetricTypeHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	p := middlewares.MustGetBody[CreateMetricTypeRequest](r.Context())

	metricType, err := h.commander.Create(r.Context(), p.Name, p.EntityType)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	metricType, err := h.querier.Get(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := auth.MustGetIdentity(r.Context())
	pag, err := ParsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	result, err := h.querier.List(r.Context(), &id.Scope, pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, metricTypeToResponse))
}

func (h *MetricTypeHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())
	p := middlewares.MustGetBody[UpdateMetricTypeRequest](r.Context())

	metricType, err := h.commander.Update(r.Context(), id, p.Name)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	_, err := h.querier.Get(r.Context(), id)
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
	ID         properties.UUID         `json:"id"`
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
