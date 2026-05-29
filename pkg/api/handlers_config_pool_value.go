package api

import (
	"context"
	"net/http"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
)

type CreateConfigPoolValueReq struct {
	Name         string          `json:"name"`
	Value        any             `json:"value"`
	ConfigPoolID properties.UUID `json:"configPoolId"`
}

type ConfigPoolValueHandler struct {
	querier     domain.ConfigPoolValueQuerier
	poolQuerier domain.ConfigPoolQuerier
	commander   domain.ConfigPoolValueCommander
	authz       authz.Authorizer
}

func NewConfigPoolValueHandler(
	querier domain.ConfigPoolValueQuerier,
	poolQuerier domain.ConfigPoolQuerier,
	commander domain.ConfigPoolValueCommander,
	authz authz.Authorizer,
) *ConfigPoolValueHandler {
	return &ConfigPoolValueHandler{
		querier:     querier,
		poolQuerier: poolQuerier,
		commander:   commander,
		authz:       authz,
	}
}

// createConfigPoolValueScopeExtractor inherits the parent pool's auth scope. The body
// only carries ConfigPoolID; we fetch the pool's AuthScope (which already handles the
// global/participant branch via AdminOnlyObjectScope).
func createConfigPoolValueScopeExtractor(poolQuerier domain.ConfigPoolQuerier) middlewares.ObjectScopeExtractor {
	return func(r *http.Request) (authz.ObjectScope, error) {
		body := middlewares.MustGetBody[CreateConfigPoolValueReq](r.Context())
		return poolQuerier.AuthScope(r.Context(), body.ConfigPoolID)
	}
}

func (h *ConfigPoolValueHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeConfigPoolValue, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, ConfigPoolValueToRes))

		r.With(
			middlewares.DecodeBody[CreateConfigPoolValueReq](),
			middlewares.AuthzFromExtractor(
				authz.ObjectTypeConfigPoolValue,
				authz.ActionCreate,
				h.authz,
				createConfigPoolValueScopeExtractor(h.poolQuerier),
			),
		).Post("/", Create(h.Create, ConfigPoolValueToRes))

		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeConfigPoolValue, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, ConfigPoolValueToRes))

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeConfigPoolValue, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

func (h *ConfigPoolValueHandler) Create(ctx context.Context, req *CreateConfigPoolValueReq) (*domain.ConfigPoolValue, error) {
	return h.commander.Create(ctx, domain.CreateConfigPoolValueParams{
		Name:         req.Name,
		Value:        req.Value,
		ConfigPoolID: req.ConfigPoolID,
	})
}

type ConfigPoolValueRes struct {
	ID               properties.UUID    `json:"id"`
	Name             string             `json:"name"`
	Value            any                `json:"value"`
	ConfigPoolID     properties.UUID    `json:"configPoolId"`
	ConfigPool       *ConfigPoolRes     `json:"configPool,omitempty"`
	AgentID          *properties.UUID   `json:"agentId,omitempty"`
	Agent            *AgentRes          `json:"agent,omitempty"`
	InfrastructureID *properties.UUID   `json:"infrastructureId,omitempty"`
	Infrastructure   *InfrastructureRes `json:"infrastructure,omitempty"`
	PropertyName     *string            `json:"propertyName,omitempty"`
	AllocatedAt      *JSONUTCTime       `json:"allocatedAt,omitempty"`
	CreatedAt        JSONUTCTime        `json:"createdAt"`
	UpdatedAt        JSONUTCTime        `json:"updatedAt"`
}

func ConfigPoolValueToRes(a *domain.ConfigPoolValue) *ConfigPoolValueRes {
	res := &ConfigPoolValueRes{
		ID:               a.ID,
		Name:             a.Name,
		Value:            a.Value,
		ConfigPoolID:     a.ConfigPoolID,
		AgentID:          a.AgentID,
		InfrastructureID: a.InfrastructureID,
		PropertyName:     a.PropertyName,
		CreatedAt:        JSONUTCTime(a.CreatedAt),
		UpdatedAt:        JSONUTCTime(a.UpdatedAt),
	}
	if a.AllocatedAt != nil {
		allocatedAt := JSONUTCTime(*a.AllocatedAt)
		res.AllocatedAt = &allocatedAt
	}

	if a.ConfigPool != nil {
		res.ConfigPool = ConfigPoolToRes(a.ConfigPool)
	}

	if a.Agent != nil {
		res.Agent = AgentToRes(a.Agent)
	}

	if a.Infrastructure != nil {
		res.Infrastructure = InfrastructureToRes(a.Infrastructure)
	}

	return res
}
