package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
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
		// List metric types
		r.With(
			AuthzSimple(domain.SubjectMetricType, domain.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create metric type
		r.With(
			DecodeBody[CreateMetricTypeRequest](),
			AuthzSimple(domain.SubjectMetricType, domain.ActionCreate, h.authz),
		).Post("/", h.handleCreate)

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(ID)

			// Get metric type
			r.With(
				AuthzFromID(domain.SubjectMetricType, domain.ActionRead, h.authz, h.querier),
			).Get("/{id}", h.handleGet)

			// Update metric type
			r.With(
				DecodeBody[UpdateMetricTypeRequest](),
				AuthzFromID(domain.SubjectMetricType, domain.ActionUpdate, h.authz, h.querier),
			).Patch("/{id}", h.handleUpdate)

			// Delete metric type
			r.With(
				AuthzFromID(domain.SubjectMetricType, domain.ActionDelete, h.authz, h.querier),
			).Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *MetricTypeHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	p := MustGetBody[CreateMetricTypeRequest](r.Context())

	metricType, err := h.commander.Create(r.Context(), p.Name, p.EntityType)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	metricType, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
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

	render.JSON(w, r, NewPageResponse(result, metricTypeToResponse))
}

func (h *MetricTypeHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())
	p := MustGetBody[UpdateMetricTypeRequest](r.Context())

	metricType, err := h.commander.Update(r.Context(), id, p.Name)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	_, err := h.querier.FindByID(r.Context(), id)
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
