package api

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
)

type CreateAgentPoolValueReq struct {
	Name        string          `json:"name"`
	Value       any             `json:"value"`
	AgentPoolID properties.UUID `json:"agentPoolId"`
}

type AgentPoolValueHandler struct {
	querier   domain.AgentPoolValueQuerier
	commander domain.AgentPoolValueCommander
	authz     authz.Authorizer
}

func NewAgentPoolValueHandler(
	querier domain.AgentPoolValueQuerier,
	commander domain.AgentPoolValueCommander,
	authz authz.Authorizer,
) *AgentPoolValueHandler {
	return &AgentPoolValueHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

func (h *AgentPoolValueHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeAgentPoolValue, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, AgentPoolValueToRes))

		r.With(
			middlewares.DecodeBody[CreateAgentPoolValueReq](),
			middlewares.AuthzSimple(authz.ObjectTypeAgentPoolValue, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, AgentPoolValueToRes))

		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgentPoolValue, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, AgentPoolValueToRes))

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgentPoolValue, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

func (h *AgentPoolValueHandler) Create(ctx context.Context, req *CreateAgentPoolValueReq) (*domain.AgentPoolValue, error) {
	return h.commander.Create(ctx, domain.CreateAgentPoolValueParams{
		Name:        req.Name,
		Value:       req.Value,
		AgentPoolID: req.AgentPoolID,
	})
}

type AgentPoolValueRes struct {
	ID           properties.UUID  `json:"id"`
	Name         string           `json:"name"`
	Value        any              `json:"value"`
	AgentPoolID  properties.UUID  `json:"agentPoolId"`
	AgentPool    *AgentPoolRes    `json:"agentPool,omitempty"`
	AgentID      *properties.UUID `json:"agentId,omitempty"`
	Agent        *AgentRes        `json:"agent,omitempty"`
	PropertyName *string          `json:"propertyName,omitempty"`
	AllocatedAt  *JSONUTCTime     `json:"allocatedAt,omitempty"`
	CreatedAt    JSONUTCTime      `json:"createdAt"`
	UpdatedAt    JSONUTCTime      `json:"updatedAt"`
}

func AgentPoolValueToRes(a *domain.AgentPoolValue) *AgentPoolValueRes {
	res := &AgentPoolValueRes{
		ID:           a.ID,
		Name:         a.Name,
		Value:        a.Value,
		AgentPoolID:  a.AgentPoolID,
		AgentID:      a.AgentID,
		PropertyName: a.PropertyName,
		CreatedAt:    JSONUTCTime(a.CreatedAt),
		UpdatedAt:    JSONUTCTime(a.UpdatedAt),
	}
	if a.AllocatedAt != nil {
		allocatedAt := JSONUTCTime(*a.AllocatedAt)
		res.AllocatedAt = &allocatedAt
	}

	if a.AgentPool != nil {
		res.AgentPool = AgentPoolToRes(a.AgentPool)
	}

	if a.Agent != nil {
		res.Agent = AgentToRes(a.Agent)
	}

	return res
}
