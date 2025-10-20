package api

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
)

type CreateServiceOptionTypeReq struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type UpdateServiceOptionTypeReq struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type ServiceOptionTypeHandler struct {
	querier   domain.ServiceOptionTypeQuerier
	commander domain.ServiceOptionTypeCommander
	authz     auth.Authorizer
}

func NewServiceOptionTypeHandler(
	querier domain.ServiceOptionTypeQuerier,
	commander domain.ServiceOptionTypeCommander,
	authz auth.Authorizer,
) *ServiceOptionTypeHandler {
	return &ServiceOptionTypeHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all service option type routes registered
func (h *ServiceOptionTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List service option types - all roles can read
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeServiceOptionType, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, ServiceOptionTypeToRes))

		// Create service option type - admin only
		r.With(
			middlewares.DecodeBody[CreateServiceOptionTypeReq](),
			middlewares.AuthzSimple(authz.ObjectTypeServiceOptionType, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, ServiceOptionTypeToRes))

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get service option type - all roles can read
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceOptionType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, ServiceOptionTypeToRes))

			// Update service option type - admin only
			r.With(
				middlewares.DecodeBody[UpdateServiceOptionTypeReq](),
				middlewares.AuthzFromID(authz.ObjectTypeServiceOptionType, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, ServiceOptionTypeToRes))

			// Delete service option type - admin only
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceOptionType, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

// Adapter functions that convert request structs to commander method calls

func (h *ServiceOptionTypeHandler) Create(ctx context.Context, req *CreateServiceOptionTypeReq) (*domain.ServiceOptionType, error) {
	params := domain.CreateServiceOptionTypeParams{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
	}
	return h.commander.Create(ctx, params)
}

func (h *ServiceOptionTypeHandler) Update(ctx context.Context, id properties.UUID, req *UpdateServiceOptionTypeReq) (*domain.ServiceOptionType, error) {
	params := domain.UpdateServiceOptionTypeParams{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	}
	return h.commander.Update(ctx, params)
}

// ServiceOptionTypeRes represents the response body for service option type operations
type ServiceOptionTypeRes struct {
	ID          properties.UUID `json:"id"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Description string          `json:"description"`
	CreatedAt   JSONUTCTime     `json:"createdAt"`
	UpdatedAt   JSONUTCTime     `json:"updatedAt"`
}

// ServiceOptionTypeToRes converts a domain.ServiceOptionType to a response
func ServiceOptionTypeToRes(sot *domain.ServiceOptionType) *ServiceOptionTypeRes {
	return &ServiceOptionTypeRes{
		ID:          sot.ID,
		Name:        sot.Name,
		Type:        sot.Type,
		Description: sot.Description,
		CreatedAt:   JSONUTCTime(sot.CreatedAt),
		UpdatedAt:   JSONUTCTime(sot.UpdatedAt),
	}
}
