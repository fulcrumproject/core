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

type CreateMetricTypeReq struct {
	Name       string                  `json:"name"`
	EntityType domain.MetricEntityType `json:"entityType"`
}

type UpdateMetricTypeReq struct {
	Name *string `json:"name"`
}

type MetricTypeHandler struct {
	querier   domain.MetricTypeQuerier
	commander domain.MetricTypeCommander
	authz     auth.Authorizer
}

func NewMetricTypeHandler(
	querier domain.MetricTypeQuerier,
	commander domain.MetricTypeCommander,
	authz auth.Authorizer,
) *MetricTypeHandler {
	return &MetricTypeHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all metric type routes registered
func (h *MetricTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List metric types
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeMetricType, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, MetricTypeToRes))

		// Create metric type - using standard Create handler
		r.With(
			middlewares.DecodeBody[CreateMetricTypeReq](),
			middlewares.AuthzSimple(authz.ObjectTypeMetricType, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, MetricTypeToRes))

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get metric type
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeMetricType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, MetricTypeToRes))

			// Update metric type - using standard Update handler
			r.With(
				middlewares.DecodeBody[UpdateMetricTypeReq](),
				middlewares.AuthzFromID(authz.ObjectTypeMetricType, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, MetricTypeToRes))

			// Delete metric type
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeMetricType, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

// Adapter functions that convert request structs to commander method calls

func (h *MetricTypeHandler) Create(ctx context.Context, req *CreateMetricTypeReq) (*domain.MetricType, error) {
	return h.commander.Create(ctx, req.Name, req.EntityType)
}

func (h *MetricTypeHandler) Update(ctx context.Context, id properties.UUID, req *UpdateMetricTypeReq) (*domain.MetricType, error) {
	return h.commander.Update(ctx, id, req.Name)
}

// MetricTypeRes represents the response body for metric type operations
type MetricTypeRes struct {
	ID         properties.UUID         `json:"id"`
	Name       string                  `json:"name"`
	EntityType domain.MetricEntityType `json:"entityType"`
	CreatedAt  JSONUTCTime             `json:"createdAt"`
	UpdatedAt  JSONUTCTime             `json:"updatedAt"`
}

// MetricTypeToRes converts a domain.MetricType to a MetricTypeResponse
func MetricTypeToRes(mt *domain.MetricType) *MetricTypeRes {
	return &MetricTypeRes{
		ID:         mt.ID,
		Name:       mt.Name,
		EntityType: mt.EntityType,
		CreatedAt:  JSONUTCTime(mt.CreatedAt),
		UpdatedAt:  JSONUTCTime(mt.UpdatedAt),
	}
}
