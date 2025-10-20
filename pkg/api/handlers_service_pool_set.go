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

type CreateServicePoolSetReq struct {
	Name       string          `json:"name"`
	ProviderID properties.UUID `json:"providerId"`
}

type UpdateServicePoolSetReq struct {
	Name *string `json:"name"`
}

type ServicePoolSetHandler struct {
	querier   domain.ServicePoolSetQuerier
	commander domain.ServicePoolSetCommander
	authz     auth.Authorizer
}

func NewServicePoolSetHandler(
	querier domain.ServicePoolSetQuerier,
	commander domain.ServicePoolSetCommander,
	authz auth.Authorizer,
) *ServicePoolSetHandler {
	return &ServicePoolSetHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all service pool set routes registered
func (h *ServicePoolSetHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List service pool sets - admin and participants can read their own
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeServicePoolSet, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, ServicePoolSetToRes))

		// Create service pool set - admin and participants
		r.With(
			middlewares.DecodeBody[CreateServicePoolSetReq](),
			middlewares.AuthzSimple(authz.ObjectTypeServicePoolSet, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, ServicePoolSetToRes))

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get service pool set - scope-checked
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServicePoolSet, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, ServicePoolSetToRes))

			// Update service pool set - scope-checked
			r.With(
				middlewares.DecodeBody[UpdateServicePoolSetReq](),
				middlewares.AuthzFromID(authz.ObjectTypeServicePoolSet, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, ServicePoolSetToRes))

			// Delete service pool set - scope-checked
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServicePoolSet, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.DeleteServicePoolSet))
		})
	}
}

// Adapter functions that convert request structs to commander method calls

func (h *ServicePoolSetHandler) Create(ctx context.Context, req *CreateServicePoolSetReq) (*domain.ServicePoolSet, error) {
	params := domain.CreateServicePoolSetParams{
		Name:       req.Name,
		ProviderID: req.ProviderID,
	}
	return h.commander.CreateServicePoolSet(ctx, params)
}

func (h *ServicePoolSetHandler) Update(ctx context.Context, id properties.UUID, req *UpdateServicePoolSetReq) (*domain.ServicePoolSet, error) {
	params := domain.UpdateServicePoolSetParams{
		Name: req.Name,
	}
	return h.commander.UpdateServicePoolSet(ctx, id, params)
}

// ServicePoolSetRes represents the response body for service pool set operations
type ServicePoolSetRes struct {
	ID         properties.UUID `json:"id"`
	Name       string          `json:"name"`
	ProviderID properties.UUID `json:"providerId"`
	CreatedAt  JSONUTCTime     `json:"createdAt"`
	UpdatedAt  JSONUTCTime     `json:"updatedAt"`
}

// ServicePoolSetToRes converts a domain.ServicePoolSet to a ServicePoolSetRes
func ServicePoolSetToRes(ps *domain.ServicePoolSet) *ServicePoolSetRes {
	return &ServicePoolSetRes{
		ID:         ps.ID,
		Name:       ps.Name,
		ProviderID: ps.ProviderID,
		CreatedAt:  JSONUTCTime(ps.CreatedAt),
		UpdatedAt:  JSONUTCTime(ps.UpdatedAt),
	}
}
