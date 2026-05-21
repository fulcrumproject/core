package api

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
)

type CreateConfigPoolReq struct {
	Name            string                   `json:"name"`
	Type            string                   `json:"type"`
	PropertyType    string                   `json:"propertyType"`
	GeneratorType   domain.PoolGeneratorType `json:"generatorType"`
	GeneratorConfig *properties.JSON         `json:"generatorConfig,omitempty"`
	ParticipantID   *properties.UUID         `json:"participantId,omitempty"`
}

// ObjectScope implements middlewares.ObjectScopeProvider — a participant body falls into
// DefaultObjectScope (only owning participant + admin match); a nil ParticipantID means
// "global pool", which is admin-only via AdminOnlyObjectScope (otherwise the all-nil
// DefaultObjectScope branch would let any participant create/modify globals).
func (r CreateConfigPoolReq) ObjectScope() (authz.ObjectScope, error) {
	if r.ParticipantID == nil {
		return authz.AdminOnlyObjectScope{}, nil
	}
	return &authz.DefaultObjectScope{ParticipantID: r.ParticipantID}, nil
}

type UpdateConfigPoolReq struct {
	Name            *string          `json:"name"`
	GeneratorConfig *properties.JSON `json:"generatorConfig,omitempty"`
}

type ConfigPoolHandler struct {
	querier   domain.ConfigPoolQuerier
	commander domain.ConfigPoolCommander
	authz     authz.Authorizer
}

func NewConfigPoolHandler(
	querier domain.ConfigPoolQuerier,
	commander domain.ConfigPoolCommander,
	authz authz.Authorizer,
) *ConfigPoolHandler {
	return &ConfigPoolHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

func (h *ConfigPoolHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeConfigPool, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, ConfigPoolToRes))

		r.With(
			middlewares.DecodeBody[CreateConfigPoolReq](),
			middlewares.AuthzFromBody[CreateConfigPoolReq](authz.ObjectTypeConfigPool, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, ConfigPoolToRes))

		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeConfigPool, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, ConfigPoolToRes))

			r.With(
				middlewares.DecodeBody[UpdateConfigPoolReq](),
				middlewares.AuthzFromID(authz.ObjectTypeConfigPool, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, ConfigPoolToRes))

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeConfigPool, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

func (h *ConfigPoolHandler) Create(ctx context.Context, req *CreateConfigPoolReq) (*domain.ConfigPool, error) {
	params := domain.CreateConfigPoolParams{
		Name:            req.Name,
		Type:            req.Type,
		PropertyType:    req.PropertyType,
		GeneratorType:   req.GeneratorType,
		GeneratorConfig: req.GeneratorConfig,
		ParticipantID:   req.ParticipantID,
	}
	return h.commander.Create(ctx, params)
}

func (h *ConfigPoolHandler) Update(ctx context.Context, id properties.UUID, req *UpdateConfigPoolReq) (*domain.ConfigPool, error) {
	params := domain.UpdateConfigPoolParams{
		Name:            req.Name,
		GeneratorConfig: req.GeneratorConfig,
	}
	return h.commander.Update(ctx, id, params)
}

type ConfigPoolRes struct {
	ID              properties.UUID          `json:"id"`
	Name            string                   `json:"name"`
	Type            string                   `json:"type"`
	PropertyType    string                   `json:"propertyType"`
	GeneratorType   domain.PoolGeneratorType `json:"generatorType"`
	GeneratorConfig *properties.JSON         `json:"generatorConfig,omitempty"`
	ParticipantID   *properties.UUID         `json:"participantId,omitempty"`
	CreatedAt       JSONUTCTime              `json:"createdAt"`
	UpdatedAt       JSONUTCTime              `json:"updatedAt"`
}

func ConfigPoolToRes(a *domain.ConfigPool) *ConfigPoolRes {
	return &ConfigPoolRes{
		ID:              a.ID,
		Name:            a.Name,
		Type:            a.Type,
		PropertyType:    a.PropertyType,
		GeneratorType:   a.GeneratorType,
		GeneratorConfig: a.GeneratorConfig,
		ParticipantID:   a.ParticipantID,
		CreatedAt:       JSONUTCTime(a.CreatedAt),
		UpdatedAt:       JSONUTCTime(a.UpdatedAt),
	}
}
