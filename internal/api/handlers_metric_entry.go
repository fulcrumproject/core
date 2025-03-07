package api

import (
	"errors"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type MetricEntryHandler struct {
	querier   domain.MetricEntryQuerier
	commander *domain.MetricEntryCommander
}

func NewMetricEntryHandler(
	querier domain.MetricEntryQuerier,
	commander *domain.MetricEntryCommander,
) *MetricEntryHandler {
	return &MetricEntryHandler{
		querier:   querier,
		commander: commander,
	}
}

// Routes returns the router with all metric entry routes registered
func (h *MetricEntryHandler) Routes(agentAuthMw func(http.Handler) http.Handler) func(r chi.Router) {
	return func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Get("/", h.handleList)
		})
		// Agent authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(agentAuthMw)
			r.Post("/", h.handleCreate)
		})
	}
}

func (h *MetricEntryHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var p struct {
		ServiceID  *domain.UUID `json:"serviceId"`
		ExternalID *string      `json:"externalId"`
		ResourceID string       `json:"resourceId"`
		Value      float64      `json:"value"`
		TypeName   string       `json:"typeName"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	// Get agent ID from the authenticated agent in the context
	agent := GetAuthenticatedAgent(r)
	if agent == nil {
		render.Render(w, r, ErrUnauthorized())
		return
	}
	var (
		metricEntry *domain.MetricEntry
		err         error
	)
	if p.ServiceID != nil {
		metricEntry, err = h.commander.Create(r.Context(), p.TypeName, agent.ID, *p.ServiceID, p.ResourceID, p.Value)
	} else if p.ExternalID != nil {
		metricEntry, err = h.commander.CreateWithExternalID(r.Context(), p.TypeName, agent.ID, *p.ExternalID, p.ResourceID, p.Value)
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("at least one of serviceId or externalId must be specified")))
		return
	}
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, metricEntryToResponse(metricEntry))
}

func (h *MetricEntryHandler) handleList(w http.ResponseWriter, r *http.Request) {
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
	render.JSON(w, r, NewPageResponse(result, metricEntryToResponse))
}

// MetricEntryResponse represents the response body for metric entry operations
type MetricEntryResponse struct {
	ID         domain.UUID         `json:"id"`
	AgentID    domain.UUID         `json:"agentId"`
	ServiceID  domain.UUID         `json:"serviceId"`
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
		ID:         me.ID,
		AgentID:    me.AgentID,
		ServiceID:  me.ServiceID,
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
