package api

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
)

type CreateAgentPoolReq struct {
	Name            string                   `json:"name"`
	Type            string                   `json:"type"`
	PropertyType    string                   `json:"propertyType"`
	GeneratorType   domain.PoolGeneratorType `json:"generatorType"`
	GeneratorConfig *properties.JSON         `json:"generatorConfig,omitempty"`
}

type UpdateAgentPoolReq struct {
	Name            *string          `json:"name"`
	GeneratorConfig *properties.JSON `json:"generatorConfig,omitempty"`
}

type AgentPoolHandler struct {
	querier   domain.AgentPoolQuerier
	commander domain.AgentPoolCommander
	authz     authz.Authorizer
}

func NewAgentPoolHandler(
	querier domain.AgentPoolQuerier,
	commander domain.AgentPoolCommander,
	authz authz.Authorizer,
) *AgentPoolHandler {
	return &AgentPoolHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

func (h *AgentPoolHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeAgentPool, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, AgentPoolToRes))

		r.With(
			middlewares.DecodeBody[CreateAgentPoolReq](),
			middlewares.AuthzSimple(authz.ObjectTypeAgentPool, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, AgentPoolToRes))

		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgentPool, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, AgentPoolToRes))

			r.With(
				middlewares.DecodeBody[UpdateAgentPoolReq](),
				middlewares.AuthzFromID(authz.ObjectTypeAgentPool, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, AgentPoolToRes))

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgentPool, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

func (h *AgentPoolHandler) Create(ctx context.Context, req *CreateAgentPoolReq) (*domain.AgentPool, error) {
	params := domain.CreateAgentPoolParams{
		Name:            req.Name,
		Type:            req.Type,
		PropertyType:    req.PropertyType,
		GeneratorType:   req.GeneratorType,
		GeneratorConfig: req.GeneratorConfig,
	}
	return h.commander.Create(ctx, params)
}

func (h *AgentPoolHandler) Update(ctx context.Context, id properties.UUID, req *UpdateAgentPoolReq) (*domain.AgentPool, error) {
	params := domain.UpdateAgentPoolParams{
		Name:            req.Name,
		GeneratorConfig: req.GeneratorConfig,
	}
	return h.commander.Update(ctx, id, params)
}

type AgentPoolRes struct {
	ID              properties.UUID          `json:"id"`
	Name            string                   `json:"name"`
	Type            string                   `json:"type"`
	PropertyType    string                   `json:"propertyType"`
	GeneratorType   domain.PoolGeneratorType `json:"generatorType"`
	GeneratorConfig *properties.JSON         `json:"generatorConfig,omitempty"`
	CreatedAt       JSONUTCTime              `json:"createdAt"`
	UpdatedAt       JSONUTCTime              `json:"updatedAt"`
}

func AgentPoolToRes(a *domain.AgentPool) *AgentPoolRes {
	return &AgentPoolRes{
		ID:              a.ID,
		Name:            a.Name,
		Type:            a.Type,
		PropertyType:    a.PropertyType,
		GeneratorType:   a.GeneratorType,
		GeneratorConfig: a.GeneratorConfig,
		CreatedAt:       JSONUTCTime(a.CreatedAt),
		UpdatedAt:       JSONUTCTime(a.UpdatedAt),
	}
}
