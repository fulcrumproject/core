package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type MetricTypeHandler struct {
	querier   domain.MetricTypeQuerier
	commander *domain.MetricTypeCommander
}

func NewMetricTypeHandler(
	repo domain.MetricTypeRepository,
	commander *domain.MetricTypeCommander,
) *MetricTypeHandler {
	return &MetricTypeHandler{
		querier:   repo,
		commander: commander,
	}
}

// Routes returns the router with all metric type routes registered
func (h *MetricTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(UUIDMiddleware)
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
	metricType, err := h.commander.Create(r.Context(), req.Name, req.EntityType)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
	metricType, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
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
	render.JSON(w, r, NewPageResponse(result, metricTypeToResponse))
}

func (h *MetricTypeHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
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
	id := GetUUIDParam(r)
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
