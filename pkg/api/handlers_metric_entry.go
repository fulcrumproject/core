package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateMetricEntryReq struct {
	ServiceID       *properties.UUID `json:"serviceId,omitempty"`
	AgentInstanceID *string          `json:"agentInstanceId,omitempty"`
	ResourceID      string           `json:"resourceId"`
	Value           float64          `json:"value"`
	TypeName        string           `json:"typeName"`
	MetricTypeID    properties.UUID  `json:"metricTypeId"`
	EntityType      string           `json:"entityType"`
	EntityID        properties.UUID  `json:"entityId"`
	Timestamp       time.Time        `json:"timestamp"`
}

type MetricEntryHandler struct {
	querier        domain.MetricEntryQuerier
	serviceQuerier domain.ServiceQuerier
	commander      domain.MetricEntryCommander
	authz          authz.Authorizer
}

func NewMetricEntryHandler(
	querier domain.MetricEntryQuerier,
	serviceQuerier domain.ServiceQuerier,
	commander domain.MetricEntryCommander,
	authz authz.Authorizer,
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
			middlewares.AuthzSimple(authz.ObjectTypeMetricEntry, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, MetricEntryToRes))

		// Create metric entry
		r.With(
			middlewares.DecodeBody[CreateMetricEntryReq](),
			middlewares.AuthzSimple(authz.ObjectTypeMetricEntry, authz.ActionCreate, h.authz),
		).Post("/", h.Create)
	}
}

func (h *MetricEntryHandler) Create(w http.ResponseWriter, r *http.Request) {
	p := middlewares.MustGetBody[CreateMetricEntryReq](r.Context())
	id := auth.MustGetIdentity(r.Context())

	var (
		service     *domain.Service
		err         error
		metricEntry *domain.MetricEntry
	)
	if p.ServiceID != nil {
		service, err = h.serviceQuerier.Get(r.Context(), *p.ServiceID)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
		params := domain.CreateMetricEntryParams{
			TypeName:   p.TypeName,
			AgentID:    service.AgentID,
			ServiceID:  *p.ServiceID,
			ResourceID: p.ResourceID,
			Value:      p.Value,
		}
		metricEntry, err = h.commander.Create(r.Context(), params)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
	} else if p.AgentInstanceID != nil && id.HasRole(auth.RoleAgent) && id.Scope.AgentID != nil {
		service, err = h.serviceQuerier.FindByAgentInstanceID(r.Context(), *id.Scope.AgentID, *p.AgentInstanceID)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
		params := domain.CreateMetricEntryWithAgentInstanceIDParams{
			TypeName:        p.TypeName,
			AgentID:         service.AgentID,
			AgentInstanceID: *p.AgentInstanceID,
			ResourceID:      p.ResourceID,
			Value:           p.Value,
		}
		metricEntry, err = h.commander.CreateWithAgentInstanceID(r.Context(), params)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("serviceId or agent role and agentInstanceId are required")))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, MetricEntryToRes(metricEntry))
}

// MetricEntryRes represents the response body for metric entry operations
type MetricEntryRes struct {
	ID         properties.UUID `json:"id"`
	ProviderID properties.UUID `json:"providerId"`
	ConsumerID properties.UUID `json:"consumerId"`
	AgentID    properties.UUID `json:"agentId"`
	ServiceID  properties.UUID `json:"serviceId"`
	ResourceID string          `json:"resourceId"`
	Value      float64         `json:"value"`
	TypeID     string          `json:"typeId"`
	CreatedAt  JSONUTCTime     `json:"createdAt"`
	UpdatedAt  JSONUTCTime     `json:"updatedAt"`
	Agent      *AgentRes       `json:"agent,omitempty"`
	Service    *ServiceRes     `json:"service,omitempty"`
	Type       *MetricTypeRes  `json:"type,omitempty"`
}

// MetricEntryToRes converts a domain.MetricEntry to a MetricEntryResponse
func MetricEntryToRes(me *domain.MetricEntry) *MetricEntryRes {
	resp := &MetricEntryRes{
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
		resp.Agent = AgentToRes(me.Agent)
	}
	if me.Service != nil {
		resp.Service = ServiceToRes(me.Service)
	}
	if me.Type != nil {
		resp.Type = MetricTypeToRes(me.Type)
	}
	return resp
}
