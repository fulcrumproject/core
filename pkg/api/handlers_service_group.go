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

type CreateServiceGroupReq struct {
	Name       string          `json:"name"`
	ConsumerID properties.UUID `json:"consumerId"`
}

func (r CreateServiceGroupReq) ObjectScope() (auth.ObjectScope, error) {
	return &auth.DefaultObjectScope{ConsumerID: &r.ConsumerID}, nil
}

type UpdateServiceGroupReq struct {
	Name *string `json:"name"`
}

type ServiceGroupHandler struct {
	querier   domain.ServiceGroupQuerier
	commander domain.ServiceGroupCommander
	authz     auth.Authorizer
}

func NewServiceGroupHandler(
	querier domain.ServiceGroupQuerier,
	commander domain.ServiceGroupCommander,
	authz auth.Authorizer,
) *ServiceGroupHandler {
	return &ServiceGroupHandler{
		commander: commander,
		querier:   querier,
		authz:     authz,
	}
}

// Routes returns the router with all service group routes registered
func (h *ServiceGroupHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeServiceGroup, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, ServiceGroupToRes))

		// Create endpoint - using standard Create handler
		r.With(
			middlewares.DecodeBody[CreateServiceGroupReq](),
			middlewares.AuthzFromBody[CreateServiceGroupReq](authz.ObjectTypeServiceGroup, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, ServiceGroupToRes))

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using service group's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceGroup, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier, ServiceGroupToRes))

			// Update endpoint - using standard Update handler
			r.With(
				middlewares.DecodeBody[UpdateServiceGroupReq](),
				middlewares.AuthzFromID(authz.ObjectTypeServiceGroup, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, ServiceGroupToRes))

			// Delete endpoint - authorize using service group's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceGroup, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

// Adapter functions that convert request structs to commander method calls
func (h *ServiceGroupHandler) Create(ctx context.Context, req *CreateServiceGroupReq) (*domain.ServiceGroup, error) {
	return h.commander.Create(ctx, req.Name, req.ConsumerID)
}

// Adapter functions that convert request structs to commander method calls
func (h *ServiceGroupHandler) Update(ctx context.Context, id properties.UUID, req *UpdateServiceGroupReq) (*domain.ServiceGroup, error) {
	return h.commander.Update(ctx, id, req.Name)
}

// ServiceGroupRes represents the response body for service group operations
type ServiceGroupRes struct {
	ID         properties.UUID `json:"id"`
	Name       string          `json:"name"`
	ConsumerID properties.UUID `json:"consumerId"`
	CreatedAt  JSONUTCTime     `json:"createdAt"`
	UpdatedAt  JSONUTCTime     `json:"updatedAt"`
}

// ServiceGroupToRes converts a domain.ServiceGroup to a ServiceGroupResponse
func ServiceGroupToRes(sg *domain.ServiceGroup) *ServiceGroupRes {
	return &ServiceGroupRes{
		ID:         sg.ID,
		Name:       sg.Name,
		ConsumerID: sg.ConsumerID,
		CreatedAt:  JSONUTCTime(sg.CreatedAt),
		UpdatedAt:  JSONUTCTime(sg.UpdatedAt),
	}
}
