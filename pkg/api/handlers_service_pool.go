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

type CreateServicePoolReq struct {
	Name             string                   `json:"name"`
	Type             string                   `json:"type"`
	GeneratorType    domain.PoolGeneratorType `json:"generatorType"`
	GeneratorConfig  *properties.JSON         `json:"generatorConfig,omitempty"`
	ServicePoolSetID properties.UUID          `json:"servicePoolSetId"`
}

type UpdateServicePoolReq struct {
	Name            *string          `json:"name"`
	GeneratorConfig *properties.JSON `json:"generatorConfig,omitempty"`
}

type ServicePoolHandler struct {
	querier   domain.ServicePoolQuerier
	commander domain.ServicePoolCommander
	authz     auth.Authorizer
}

func NewServicePoolHandler(
	querier domain.ServicePoolQuerier,
	commander domain.ServicePoolCommander,
	authz auth.Authorizer,
) *ServicePoolHandler {
	return &ServicePoolHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all service pool routes registered
func (h *ServicePoolHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List service pools - admin, participants, and agents can read
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeServicePool, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, ServicePoolToRes))

		// Create service pool - admin and participants
		r.With(
			middlewares.DecodeBody[CreateServicePoolReq](),
			middlewares.AuthzSimple(authz.ObjectTypeServicePool, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, ServicePoolToRes))

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get service pool - scope-checked
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServicePool, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, ServicePoolToRes))

			// Update service pool - scope-checked
			r.With(
				middlewares.DecodeBody[UpdateServicePoolReq](),
				middlewares.AuthzFromID(authz.ObjectTypeServicePool, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, ServicePoolToRes))

			// Delete service pool - scope-checked
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServicePool, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

// Adapter functions that convert request structs to commander method calls

func (h *ServicePoolHandler) Create(ctx context.Context, req *CreateServicePoolReq) (*domain.ServicePool, error) {
	params := domain.CreateServicePoolParams{
		Name:             req.Name,
		Type:             req.Type,
		GeneratorType:    req.GeneratorType,
		GeneratorConfig:  req.GeneratorConfig,
		ServicePoolSetID: req.ServicePoolSetID,
	}
	return h.commander.Create(ctx, params)
}

func (h *ServicePoolHandler) Update(ctx context.Context, id properties.UUID, req *UpdateServicePoolReq) (*domain.ServicePool, error) {
	params := domain.UpdateServicePoolParams{
		Name:            req.Name,
		GeneratorConfig: req.GeneratorConfig,
	}
	return h.commander.Update(ctx, id, params)
}

// ServicePoolRes represents the response body for service pool operations
type ServicePoolRes struct {
	ID               properties.UUID          `json:"id"`
	Name             string                   `json:"name"`
	Type             string                   `json:"type"`
	GeneratorType    domain.PoolGeneratorType `json:"generatorType"`
	GeneratorConfig  *properties.JSON         `json:"generatorConfig,omitempty"`
	ServicePoolSetID properties.UUID          `json:"servicePoolSetId"`
	CreatedAt        JSONUTCTime              `json:"createdAt"`
	UpdatedAt        JSONUTCTime              `json:"updatedAt"`
}

// ServicePoolToRes converts a domain.ServicePool to a ServicePoolRes
func ServicePoolToRes(p *domain.ServicePool) *ServicePoolRes {
	return &ServicePoolRes{
		ID:               p.ID,
		Name:             p.Name,
		Type:             p.Type,
		GeneratorType:    p.GeneratorType,
		GeneratorConfig:  p.GeneratorConfig,
		ServicePoolSetID: p.ServicePoolSetID,
		CreatedAt:        JSONUTCTime(p.CreatedAt),
		UpdatedAt:        JSONUTCTime(p.UpdatedAt),
	}
}
