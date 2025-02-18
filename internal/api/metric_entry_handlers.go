package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// CreateMetricEntryRequest represents the request body for creating a metric entry
type CreateMetricEntryRequest struct {
	AgentID    string  `json:"agentId"`
	ServiceID  string  `json:"serviceId"`
	ResourceID string  `json:"resourceId"`
	Value      float64 `json:"value"`
	TypeID     string  `json:"typeId"`
}

// MetricEntryResponse represents the response body for metric entry operations
type MetricEntryResponse struct {
	ID         string              `json:"id"`
	AgentID    string              `json:"agentId"`
	ServiceID  string              `json:"serviceId"`
	ResourceID string              `json:"resourceId"`
	Value      float64             `json:"value"`
	TypeID     string              `json:"typeId"`
	CreatedAt  JSONUTCTime         `json:"createdAt"`
	UpdatedAt  JSONUTCTime         `json:"updatedAt"`
	Agent      *AgentResponse      `json:"agent,omitempty"`
	Service    *ServiceResponse    `json:"service,omitempty"`
	Type       *MetricTypeResponse `json:"type,omitempty"`
}

// metricEntryToResponse converts a domain.MetricEntry to a MetricEntryResponse
func metricEntryToResponse(me *domain.MetricEntry) *MetricEntryResponse {
	resp := &MetricEntryResponse{
		ID:         me.ID.String(),
		AgentID:    me.AgentID.String(),
		ServiceID:  me.ServiceID.String(),
		ResourceID: me.ResourceID,
		Value:      me.Value,
		TypeID:     me.TypeID.String(),
		CreatedAt:  JSONUTCTime(me.CreatedAt),
		UpdatedAt:  JSONUTCTime(me.UpdatedAt),
	}

	if me.Agent != nil {
		resp.Agent = agentToResponse(me.Agent)
	}
	if me.Service != nil {
		resp.Service = serviceToResponse(me.Service)
	}
	if me.Type != nil {
		resp.Type = metricTypeToResponse(me.Type)
	}

	return resp
}

type MetricEntryHandler struct {
	repo domain.MetricEntryRepository
}

func NewMetricEntryHandler(repo domain.MetricEntryRepository) *MetricEntryHandler {
	return &MetricEntryHandler{repo: repo}
}

// Routes returns the router with all metric entry routes registered
func (h *MetricEntryHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware for all metric entry routes
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", h.handleList)
	r.Post("/", h.handleCreate)

	return r
}

func (h *MetricEntryHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateMetricEntryRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agentID, err := domain.ParseUUID(req.AgentID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	serviceID, err := domain.ParseUUID(req.ServiceID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	typeID, err := domain.ParseUUID(req.TypeID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	metricEntry := &domain.MetricEntry{
		AgentID:    agentID,
		ServiceID:  serviceID,
		ResourceID: req.ResourceID,
		Value:      req.Value,
		TypeID:     typeID,
	}

	if err := h.repo.Create(r.Context(), metricEntry); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, metricEntryToResponse(metricEntry))
}

func (h *MetricEntryHandler) handleList(w http.ResponseWriter, r *http.Request) {
	filter := parseSimpleFilter(r)
	sorting := parseSorting(r)
	pagination := parsePagination(r)

	result, err := h.repo.List(r.Context(), filter, sorting, pagination)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPaginatedResponse(result, metricEntryToResponse))
}
