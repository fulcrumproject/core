package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// CreateUpdateMetricTypeRequest represents the request body for creating/updating a metric type
type CreateUpdateMetricTypeRequest struct {
	Name       string                  `json:"name"`
	EntityType domain.MetricEntityType `json:"entityType"`
}

// MetricTypeResponse represents the response body for metric type operations
type MetricTypeResponse struct {
	ID         string                  `json:"id"`
	Name       string                  `json:"name"`
	EntityType domain.MetricEntityType `json:"entityType"`
	CreatedAt  JSONUTCTime             `json:"createdAt"`
	UpdatedAt  JSONUTCTime             `json:"updatedAt"`
}

// metricTypeToResponse converts a domain.MetricType to a MetricTypeResponse
func metricTypeToResponse(mt *domain.MetricType) *MetricTypeResponse {
	return &MetricTypeResponse{
		ID:         mt.ID.String(),
		Name:       mt.Name,
		EntityType: mt.EntityType,
		CreatedAt:  JSONUTCTime(mt.CreatedAt),
		UpdatedAt:  JSONUTCTime(mt.UpdatedAt),
	}
}

type MetricTypeHandler struct {
	repo domain.MetricTypeRepository
}

func NewMetricTypeHandler(repo domain.MetricTypeRepository) *MetricTypeHandler {
	return &MetricTypeHandler{repo: repo}
}

// Routes returns the router with all metric type routes registered
func (h *MetricTypeHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware for all metric type routes
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", h.handleList)
	r.Post("/", h.handleCreate)
	r.Get("/{id}", h.handleGet)
	r.Put("/{id}", h.handleUpdate)
	r.Delete("/{id}", h.handleDelete)

	return r
}

func (h *MetricTypeHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateUpdateMetricTypeRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	metricType := &domain.MetricType{
		Name:       req.Name,
		EntityType: req.EntityType,
	}

	if err := h.repo.Create(r.Context(), metricType); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	metricType, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	filter := parseSimpleFilter(r)
	sorting := parseSorting(r)
	pagination := parsePagination(r)

	result, err := h.repo.List(r.Context(), filter, sorting, pagination)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPaginatedResponse(result, metricTypeToResponse))
}

func (h *MetricTypeHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	var req CreateUpdateMetricTypeRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	metricType, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	// Update fields
	metricType.Name = req.Name
	metricType.EntityType = req.EntityType

	if err := h.repo.Save(r.Context(), metricType); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.JSON(w, r, metricTypeToResponse(metricType))
}

func (h *MetricTypeHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	_, err = h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
