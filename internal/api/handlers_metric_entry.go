package api

import (
	"errors"
	"net/http"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateMetricEntryRequest struct {
	ServiceID    *domain.UUID `json:"serviceId,omitempty"`
	ExternalID   *string      `json:"externalId,omitempty"`
	ResourceID   string       `json:"resourceId"`
	Value        float64      `json:"value"`
	TypeName     string       `json:"typeName"`
	MetricTypeID domain.UUID  `json:"metricTypeId"`
	EntityType   string       `json:"entityType"`
	EntityID     domain.UUID  `json:"entityId"`
	Timestamp    time.Time    `json:"timestamp"`
}

type MetricEntryHandler struct {
	querier        domain.MetricEntryQuerier
	serviceQuerier domain.ServiceQuerier
	commander      domain.MetricEntryCommander
	authz          domain.Authorizer
}

func NewMetricEntryHandler(
	querier domain.MetricEntryQuerier,
	serviceQuerier domain.ServiceQuerier,
	commander domain.MetricEntryCommander,
	authz domain.Authorizer,
) *MetricEntryHandler {
	return &MetricEntryHandler{
		querier:        querier,
		commander:      commander,
		serviceQuerier: serviceQuerier,
		authz:          authz,
	}
}

// Routes returns the router with all metric entry routes registered
func (h *MetricEntryHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List metrics
		r.With(
			AuthzSimple(domain.SubjectMetricEntry, domain.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create metric entry
		r.With(
			DecodeBody[CreateMetricEntryRequest](),
			AuthzSimple(domain.SubjectMetricEntry, domain.ActionCreate, h.authz),
		).Post("/", h.handleCreate)
	}
}

func (h *MetricEntryHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	p := MustGetBody[CreateMetricEntryRequest](r.Context())
	id := domain.MustGetAuthIdentity(r.Context())

	var (
		service     *domain.Service
		err         error
		metricEntry *domain.MetricEntry
	)
	if p.ServiceID != nil {
		service, err = h.serviceQuerier.FindByID(r.Context(), *p.ServiceID)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
		metricEntry, err = h.commander.Create(r.Context(), p.TypeName, service.AgentID, *p.ServiceID, p.ResourceID, p.Value)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
	} else if p.ExternalID != nil && id.IsRole(domain.RoleAgent) && id.Scope().AgentID != nil {
		service, err = h.serviceQuerier.FindByExternalID(r.Context(), *id.Scope().AgentID, *p.ExternalID)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
		metricEntry, err = h.commander.CreateWithExternalID(r.Context(), p.TypeName, service.AgentID, *p.ExternalID, p.ResourceID, p.Value)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("serviceId or agent role and externalId are required")))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, metricEntryToResponse(metricEntry))
}

func (h *MetricEntryHandler) handleList(w http.ResponseWriter, r *http.Request) {
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
	render.JSON(w, r, NewPageResponse(result, metricEntryToResponse))
}

// MetricEntryResponse represents the response body for metric entry operations
type MetricEntryResponse struct {
	ID         domain.UUID         `json:"id"`
	ProviderID domain.UUID         `json:"providerId"`
	ConsumerID domain.UUID         `json:"consumerId"`
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
		ProviderID: me.ProviderID,
		ConsumerID: me.ConsumerID,
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
