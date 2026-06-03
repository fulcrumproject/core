package api

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/chi/v5"
)

type InfrastructureTypeHandler struct {
	querier   domain.InfrastructureTypeQuerier
	commander domain.InfrastructureTypeCommander
	authz     authz.Authorizer
}

func NewInfrastructureTypeHandler(
	querier domain.InfrastructureTypeQuerier,
	commander domain.InfrastructureTypeCommander,
	authz authz.Authorizer,
) *InfrastructureTypeHandler {
	return &InfrastructureTypeHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all infrastructure type routes registered.
func (h *InfrastructureTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeInfrastructureType, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, InfrastructureTypeToRes))

		// Create endpoint - admin only
		r.With(
			middlewares.DecodeBody[CreateInfrastructureTypeReq](),
			middlewares.AuthzSimple(authz.ObjectTypeInfrastructureType, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, InfrastructureTypeToRes))

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeInfrastructureType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, InfrastructureTypeToRes))

			r.With(
				middlewares.DecodeBody[UpdateInfrastructureTypeReq](),
				middlewares.AuthzFromID(authz.ObjectTypeInfrastructureType, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, InfrastructureTypeToRes))

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeInfrastructureType, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

// CreateInfrastructureTypeReq is the request body for creating infrastructure types.
type CreateInfrastructureTypeReq struct {
	Name                string        `json:"name"`
	ConfigurationSchema schema.Schema `json:"configurationSchema"`
	ConfigTemplate      string        `json:"configTemplate,omitempty"`
	CmdTemplate         string        `json:"cmdTemplate,omitempty"`
	ConfigContentType   string        `json:"configContentType,omitempty"`
}

// UpdateInfrastructureTypeReq is the request body for updating infrastructure types.
type UpdateInfrastructureTypeReq struct {
	Name                *string        `json:"name"`
	ConfigurationSchema *schema.Schema `json:"configurationSchema,omitempty"`
	ConfigTemplate      *string        `json:"configTemplate,omitempty"`
	CmdTemplate         *string        `json:"cmdTemplate,omitempty"`
	ConfigContentType   *string        `json:"configContentType,omitempty"`
}

// InfrastructureTypeRes is the response body for infrastructure type operations.
type InfrastructureTypeRes struct {
	ID                  properties.UUID `json:"id"`
	Name                string          `json:"name"`
	CreatedAt           JSONUTCTime     `json:"createdAt"`
	UpdatedAt           JSONUTCTime     `json:"updatedAt"`
	ConfigurationSchema schema.Schema   `json:"configurationSchema"`
	ConfigTemplate      string          `json:"configTemplate"`
	CmdTemplate         string          `json:"cmdTemplate"`
	ConfigContentType   string          `json:"configContentType"`
}

// InfrastructureTypeToRes converts a domain.InfrastructureType to an InfrastructureTypeRes.
func InfrastructureTypeToRes(it *domain.InfrastructureType) *InfrastructureTypeRes {
	return &InfrastructureTypeRes{
		ID:                  it.ID,
		Name:                it.Name,
		CreatedAt:           JSONUTCTime(it.CreatedAt),
		UpdatedAt:           JSONUTCTime(it.UpdatedAt),
		ConfigurationSchema: it.ConfigurationSchema,
		ConfigTemplate:      it.ConfigTemplate,
		CmdTemplate:         it.CmdTemplate,
		ConfigContentType:   it.ConfigContentType,
	}
}

// Adapter functions converting request structs to commander method calls.

func (h *InfrastructureTypeHandler) Create(ctx context.Context, req *CreateInfrastructureTypeReq) (*domain.InfrastructureType, error) {
	params := domain.CreateInfrastructureTypeParams{
		Name:                req.Name,
		ConfigurationSchema: req.ConfigurationSchema,
		ConfigTemplate:      req.ConfigTemplate,
		CmdTemplate:         req.CmdTemplate,
		ConfigContentType:   req.ConfigContentType,
	}
	return h.commander.Create(ctx, params)
}

func (h *InfrastructureTypeHandler) Update(ctx context.Context, id properties.UUID, req *UpdateInfrastructureTypeReq) (*domain.InfrastructureType, error) {
	params := domain.UpdateInfrastructureTypeParams{
		ID:                  id,
		Name:                req.Name,
		ConfigurationSchema: req.ConfigurationSchema,
		ConfigTemplate:      req.ConfigTemplate,
		CmdTemplate:         req.CmdTemplate,
		ConfigContentType:   req.ConfigContentType,
	}
	return h.commander.Update(ctx, params)
}
