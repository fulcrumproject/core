package api

import (
	"context"
	"net/http"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateInfrastructureReq struct {
	Name                 string           `json:"name"`
	ProviderID           properties.UUID  `json:"providerId"`
	InfrastructureTypeID properties.UUID  `json:"infrastructureTypeId"`
	Tags                 []string         `json:"tags"`
	Configuration        *properties.JSON `json:"configuration,omitempty"`
}

// ObjectScope implements authz.ObjectScopeProvider — scopes create authorization
// against the provider participant.
func (r CreateInfrastructureReq) ObjectScope() (authz.ObjectScope, error) {
	return &authz.DefaultObjectScope{ParticipantID: &r.ProviderID}, nil
}

type UpdateInfrastructureReq struct {
	Name          *string          `json:"name,omitempty"`
	Tags          *[]string        `json:"tags,omitempty"`
	Configuration *properties.JSON `json:"configuration,omitempty"`
}

type InfrastructureHandler struct {
	querier   domain.InfrastructureQuerier
	commander domain.InfrastructureCommander
	authz     authz.Authorizer
}

func NewInfrastructureHandler(
	querier domain.InfrastructureQuerier,
	commander domain.InfrastructureCommander,
	authz authz.Authorizer,
) *InfrastructureHandler {
	return &InfrastructureHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

func (h *InfrastructureHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeInfrastructure, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, InfrastructureToRes))

		// Create endpoint
		r.With(
			middlewares.DecodeBody[CreateInfrastructureReq](),
			middlewares.AuthzFromBody[CreateInfrastructureReq](authz.ObjectTypeInfrastructure, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, InfrastructureToRes))

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeInfrastructure, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, InfrastructureToRes))

			r.With(
				middlewares.DecodeBody[UpdateInfrastructureReq](),
				middlewares.AuthzFromID(authz.ObjectTypeInfrastructure, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, InfrastructureToRes))

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeInfrastructure, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})

		// Self endpoint — the install-token flow (Phase 3) will hit this with a
		// RoleAgent bootstrap identity whose Scope.AgentID carries the
		// infrastructure id (AgentID coordinate doubles as the self-reference).
		r.With(
			middlewares.MustHaveRoles(auth.RoleAgent),
		).Get("/me", h.GetMe)
	}
}

func (h *InfrastructureHandler) Create(ctx context.Context, req *CreateInfrastructureReq) (*domain.Infrastructure, error) {
	params := domain.CreateInfrastructureParams{
		Name:                 req.Name,
		ProviderID:           req.ProviderID,
		InfrastructureTypeID: req.InfrastructureTypeID,
		Tags:                 req.Tags,
		Configuration:        req.Configuration,
	}
	return h.commander.Create(ctx, params)
}

func (h *InfrastructureHandler) Update(ctx context.Context, id properties.UUID, req *UpdateInfrastructureReq) (*domain.Infrastructure, error) {
	params := domain.UpdateInfrastructureParams{
		ID:            id,
		Name:          req.Name,
		Tags:          req.Tags,
		Configuration: req.Configuration,
	}
	return h.commander.Update(ctx, params)
}

// GetMe returns the infrastructure row referenced by the identity's AgentID
// coordinate (set when the install flow issues a bootstrap token).
func (h *InfrastructureHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	infraID := auth.MustGetIdentity(r.Context()).Scope.AgentID
	infra, err := h.querier.Get(r.Context(), *infraID)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, InfrastructureToRes(infra))
}

type InfrastructureRes struct {
	ID                   properties.UUID        `json:"id"`
	Name                 string                 `json:"name"`
	ProviderID           properties.UUID        `json:"providerId"`
	InfrastructureTypeID properties.UUID        `json:"infrastructureTypeId"`
	Tags                 []string               `json:"tags"`
	Configuration        *properties.JSON       `json:"configuration,omitempty"`
	Participant          *ParticipantRes        `json:"participant,omitempty"`
	InfrastructureType   *InfrastructureTypeRes `json:"infrastructureType,omitempty"`
	CreatedAt            JSONUTCTime            `json:"createdAt"`
	UpdatedAt            JSONUTCTime            `json:"updatedAt"`
}

func InfrastructureToRes(i *domain.Infrastructure) *InfrastructureRes {
	res := &InfrastructureRes{
		ID:                   i.ID,
		Name:                 i.Name,
		ProviderID:           i.ProviderID,
		InfrastructureTypeID: i.InfrastructureTypeID,
		Tags:                 []string(i.Tags),
		Configuration:        i.Configuration,
		CreatedAt:            JSONUTCTime(i.CreatedAt),
		UpdatedAt:            JSONUTCTime(i.UpdatedAt),
	}
	if i.Provider != nil {
		res.Participant = ParticipantToRes(i.Provider)
	}
	if i.InfrastructureType != nil {
		res.InfrastructureType = InfrastructureTypeToRes(i.InfrastructureType)
	}
	return res
}
