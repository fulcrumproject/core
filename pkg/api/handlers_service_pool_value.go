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

type CreateServicePoolValueReq struct {
	ServicePoolID properties.UUID `json:"servicePoolId"`
	Name          string          `json:"name"`
	Value         any             `json:"value"`
}

type ServicePoolValueHandler struct {
	querier   domain.ServicePoolValueQuerier
	commander domain.ServicePoolValueCommander
	authz     auth.Authorizer
}

func NewServicePoolValueHandler(
	querier domain.ServicePoolValueQuerier,
	commander domain.ServicePoolValueCommander,
	authz auth.Authorizer,
) *ServicePoolValueHandler {
	return &ServicePoolValueHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all service pool value routes registered
func (h *ServicePoolValueHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List service pool values - admin, participants, and agents can read
		// Supports filters: servicePoolId, serviceId, allocated (true/false)
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeServicePoolValue, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, ServicePoolValueToRes))

		// Create service pool value - admin and participants (for list generators only)
		r.With(
			middlewares.DecodeBody[CreateServicePoolValueReq](),
			middlewares.AuthzSimple(authz.ObjectTypeServicePoolValue, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, ServicePoolValueToRes))

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get service pool value - scope-checked
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServicePoolValue, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, ServicePoolValueToRes))

			// Delete service pool value - scope-checked (only if not allocated)
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServicePoolValue, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.DeleteServicePoolValue))
		})
	}
}

// Adapter functions that convert request structs to commander method calls

func (h *ServicePoolValueHandler) Create(ctx context.Context, req *CreateServicePoolValueReq) (*domain.ServicePoolValue, error) {
	params := domain.CreateServicePoolValueParams{
		ServicePoolID: req.ServicePoolID,
		Name:          req.Name,
		Value:         req.Value,
	}
	return h.commander.CreateServicePoolValue(ctx, params)
}

// ServicePoolValueRes represents the response body for service pool value operations
type ServicePoolValueRes struct {
	ID            properties.UUID  `json:"id"`
	Name          string           `json:"name"`
	Value         any              `json:"value"`
	ServicePoolID properties.UUID  `json:"servicePoolId"`
	ServiceID     *properties.UUID `json:"serviceId,omitempty"`
	PropertyName  *string          `json:"propertyName,omitempty"`
	AllocatedAt   *JSONUTCTime     `json:"allocatedAt,omitempty"`
	CreatedAt     JSONUTCTime      `json:"createdAt"`
	UpdatedAt     JSONUTCTime      `json:"updatedAt"`
}

// ServicePoolValueToRes converts a domain.ServicePoolValue to a ServicePoolValueRes
func ServicePoolValueToRes(v *domain.ServicePoolValue) *ServicePoolValueRes {
	res := &ServicePoolValueRes{
		ID:            v.ID,
		Name:          v.Name,
		Value:         v.Value,
		ServicePoolID: v.ServicePoolID,
		ServiceID:     v.ServiceID,
		PropertyName:  v.PropertyName,
		CreatedAt:     JSONUTCTime(v.CreatedAt),
		UpdatedAt:     JSONUTCTime(v.UpdatedAt),
	}

	if v.AllocatedAt != nil {
		allocated := JSONUTCTime(*v.AllocatedAt)
		res.AllocatedAt = &allocated
	}

	return res
}
