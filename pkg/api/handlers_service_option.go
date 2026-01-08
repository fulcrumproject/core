package api

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
)

type CreateServiceOptionReq struct {
	ProviderID          properties.UUID `json:"providerId"`
	ServiceOptionTypeID properties.UUID `json:"serviceOptionTypeId"`
	Name                string          `json:"name"`
	Value               any             `json:"value"`
	Enabled             bool            `json:"enabled"`
	DisplayOrder        int             `json:"displayOrder"`
}

func (r CreateServiceOptionReq) ObjectScope() (authz.ObjectScope, error) {
	return &authz.DefaultObjectScope{
		ProviderID: &r.ProviderID,
	}, nil
}

type UpdateServiceOptionReq struct {
	Name         *string `json:"name"`
	Value        *any    `json:"value"`
	Enabled      *bool   `json:"enabled"`
	DisplayOrder *int    `json:"displayOrder"`
}

type ServiceOptionHandler struct {
	querier   domain.ServiceOptionQuerier
	commander domain.ServiceOptionCommander
	authz     authz.Authorizer
}

func NewServiceOptionHandler(
	querier domain.ServiceOptionQuerier,
	commander domain.ServiceOptionCommander,
	authz authz.Authorizer,
) *ServiceOptionHandler {
	return &ServiceOptionHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all service option routes registered
func (h *ServiceOptionHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List service options - scoped to provider
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeServiceOption, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, ServiceOptionToRes))

		// Create service option - admin, participant (own provider), agent (own provider)
		r.With(
			middlewares.DecodeBody[CreateServiceOptionReq](),
			middlewares.AuthzFromBody[CreateServiceOptionReq](authz.ObjectTypeServiceOption, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, ServiceOptionToRes))

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get service option
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceOption, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, ServiceOptionToRes))

			// Update service option
			r.With(
				middlewares.DecodeBody[UpdateServiceOptionReq](),
				middlewares.AuthzFromID(authz.ObjectTypeServiceOption, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, ServiceOptionToRes))

			// Delete service option
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceOption, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

// Adapter functions that convert request structs to commander method calls

func (h *ServiceOptionHandler) Create(ctx context.Context, req *CreateServiceOptionReq) (*domain.ServiceOption, error) {
	params := domain.CreateServiceOptionParams{
		ProviderID:          req.ProviderID,
		ServiceOptionTypeID: req.ServiceOptionTypeID,
		Name:                req.Name,
		Value:               req.Value,
		Enabled:             req.Enabled,
		DisplayOrder:        req.DisplayOrder,
	}
	return h.commander.Create(ctx, params)
}

func (h *ServiceOptionHandler) Update(ctx context.Context, id properties.UUID, req *UpdateServiceOptionReq) (*domain.ServiceOption, error) {
	params := domain.UpdateServiceOptionParams{
		ID:           id,
		Name:         req.Name,
		Value:        req.Value,
		Enabled:      req.Enabled,
		DisplayOrder: req.DisplayOrder,
	}
	return h.commander.Update(ctx, params)
}

// ServiceOptionRes represents the response body for service option operations
type ServiceOptionRes struct {
	ID                  properties.UUID `json:"id"`
	ProviderID          properties.UUID `json:"providerId"`
	ServiceOptionTypeID properties.UUID `json:"serviceOptionTypeId"`
	Name                string          `json:"name"`
	Value               any             `json:"value"`
	Enabled             bool            `json:"enabled"`
	DisplayOrder        int             `json:"displayOrder"`
	CreatedAt           JSONUTCTime     `json:"createdAt"`
	UpdatedAt           JSONUTCTime     `json:"updatedAt"`
}

// ServiceOptionToRes converts a domain.ServiceOption to a response
func ServiceOptionToRes(so *domain.ServiceOption) *ServiceOptionRes {
	return &ServiceOptionRes{
		ID:                  so.ID,
		ProviderID:          so.ProviderID,
		ServiceOptionTypeID: so.ServiceOptionTypeID,
		Name:                so.Name,
		Value:               so.Value,
		Enabled:             so.Enabled,
		DisplayOrder:        so.DisplayOrder,
		CreatedAt:           JSONUTCTime(so.CreatedAt),
		UpdatedAt:           JSONUTCTime(so.UpdatedAt),
	}
}
