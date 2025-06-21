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

type ServiceHandler struct {
	querier             domain.ServiceQuerier
	agentQuerier        domain.AgentQuerier
	serviceGroupQuerier domain.ServiceGroupQuerier
	commander           domain.ServiceCommander
	authz               auth.Authorizer
}

func NewServiceHandler(
	querier domain.ServiceQuerier,
	agentQuerier domain.AgentQuerier,
	serviceGroupQuerier domain.ServiceGroupQuerier,
	commander domain.ServiceCommander,
	authz auth.Authorizer,
) *ServiceHandler {
	return &ServiceHandler{
		querier:             querier,
		agentQuerier:        agentQuerier,
		serviceGroupQuerier: serviceGroupQuerier,
		commander:           commander,
		authz:               authz,
	}
}

// Request types

// CreateServiceReq represents the request to create a service
type CreateServiceReq struct {
	GroupID       properties.UUID  `json:"groupId"`
	AgentID       *properties.UUID `json:"agentId,omitempty"`
	ServiceTypeID properties.UUID  `json:"serviceTypeId"`
	AgentTags     []string         `json:"agentTags,omitempty"`
	Name          string           `json:"name"`
	Properties    properties.JSON  `json:"properties"`
}

// UpdateServiceReq represents the request to update a service
type UpdateServiceReq struct {
	Name       *string          `json:"name,omitempty"`
	Properties *properties.JSON `json:"properties,omitempty"`
}

// ServiceActionReq represents a status transition request
type ServiceActionReq struct {
	Action string `json:"action"`
}

// CreateServiceScopeExtractor creates an extractor that gets a combined scope from the request body
// by retrieving scopes from both ServiceGroup and Agent
func CreateServiceScopeExtractor(
	serviceGroupQuerier domain.ServiceGroupQuerier,
	agentQuerier domain.AgentQuerier,
) middlewares.ObjectScopeExtractor {
	return func(r *http.Request) (auth.ObjectScope, error) {
		// Get decoded body from context
		body := middlewares.MustGetBody[CreateServiceReq](r.Context())

		// Get service group scope
		return serviceGroupQuerier.AuthScope(r.Context(), body.GroupID)
	}
}

// Routes returns the router with all service routes registered
func (h *ServiceHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeService, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, ServiceToRes))

		// Create - decode body + specialized scope extractor for authorization
		r.With(
			middlewares.DecodeBody[CreateServiceReq](),
			middlewares.AuthzFromExtractor(
				authz.ObjectTypeService,
				authz.ActionCreate,
				h.authz,
				CreateServiceScopeExtractor(h.serviceGroupQuerier, h.agentQuerier),
			),
		).Post("/", h.Create)

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier, ServiceToRes))

			// Update - decode body + authorize from resource ID
			r.With(
				middlewares.DecodeBody[UpdateServiceReq](),
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, ServiceToRes))

			// Start - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionStart, h.authz, h.querier.AuthScope),
			).Post("/{id}/start", CommandWithoutBody(h.Start))

			// Stop - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionStop, h.authz, h.querier.AuthScope),
			).Post("/{id}/stop", CommandWithoutBody(h.Stop))

			// Delete - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", CommandWithoutBody(h.Delete))

			// Retry - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Post("/{id}/retry", CommandWithoutBody(h.Retry))
		})
	}
}

// Create handles service creation with custom logic for agent selection
func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get decoded body from context
	body := middlewares.MustGetBody[CreateServiceReq](r.Context())

	var service *domain.Service
	var err error

	if body.AgentID != nil {
		// Direct agent specification
		service, err = h.commander.Create(
			r.Context(),
			*body.AgentID,
			body.ServiceTypeID,
			body.GroupID,
			body.Name,
			body.Properties,
		)
	} else {
		// Agent discovery using service type and tags
		service, err = h.commander.CreateWithTags(
			r.Context(),
			body.ServiceTypeID,
			body.GroupID,
			body.Name,
			body.Properties,
			body.AgentTags,
		)
	}

	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, ServiceToRes(service))
}

// Adapter functions for standard handlers
func (h *ServiceHandler) Update(ctx context.Context, id properties.UUID, req *UpdateServiceReq) (*domain.Service, error) {
	return h.commander.Update(ctx, id, req.Name, req.Properties)
}

func (h *ServiceHandler) Start(ctx context.Context, id properties.UUID) error {
	_, err := h.commander.Transition(ctx, id, domain.ServiceStarted)
	return err
}

func (h *ServiceHandler) Stop(ctx context.Context, id properties.UUID) error {
	_, err := h.commander.Transition(ctx, id, domain.ServiceStopped)
	return err
}

func (h *ServiceHandler) Delete(ctx context.Context, id properties.UUID) error {
	_, err := h.commander.Transition(ctx, id, domain.ServiceDeleted)
	return err
}

func (h *ServiceHandler) Retry(ctx context.Context, id properties.UUID) error {
	_, err := h.commander.Retry(ctx, id)
	return err
}

// ServiceRes represents the response body for service operations
type ServiceRes struct {
	ID                properties.UUID       `json:"id"`
	ProviderID        properties.UUID       `json:"providerId"`
	ConsumerID        properties.UUID       `json:"consumerId"`
	AgentID           properties.UUID       `json:"agentId"`
	ServiceTypeID     properties.UUID       `json:"serviceTypeId"`
	GroupID           properties.UUID       `json:"groupId"`
	ExternalID        *string               `json:"externalId,omitempty"`
	Name              string                `json:"name"`
	CurrentStatus     domain.ServiceStatus  `json:"currentStatus"`
	TargetStatus      *domain.ServiceStatus `json:"targetStatus,omitempty"`
	FailedAction      *domain.ServiceAction `json:"failedAction,omitempty"`
	ErrorMessage      *string               `json:"errorMessage,omitempty"`
	RetryCount        int                   `json:"retryCount,omitempty"`
	CurrentProperties *properties.JSON      `json:"currentProperties,omitempty"`
	TargetProperties  *properties.JSON      `json:"targetProperties,omitempty"`
	Resources         *properties.JSON      `json:"resources,omitempty"`
	CreatedAt         JSONUTCTime           `json:"createdAt"`
	UpdatedAt         JSONUTCTime           `json:"updatedAt"`
}

// ServiceToRes converts a domain.Service to a ServiceResponse
func ServiceToRes(s *domain.Service) *ServiceRes {
	resp := &ServiceRes{
		ID:                s.ID,
		ProviderID:        s.ProviderID,
		ConsumerID:        s.ConsumerID,
		AgentID:           s.AgentID,
		ServiceTypeID:     s.ServiceTypeID,
		GroupID:           s.GroupID,
		ExternalID:        s.ExternalID,
		Name:              s.Name,
		CurrentStatus:     s.CurrentStatus,
		TargetStatus:      s.TargetStatus,
		FailedAction:      s.FailedAction,
		ErrorMessage:      s.ErrorMessage,
		RetryCount:        s.RetryCount,
		CurrentProperties: s.CurrentProperties,
		TargetProperties:  s.TargetProperties,
		Resources:         s.Resources,
		CreatedAt:         JSONUTCTime(s.CreatedAt),
		UpdatedAt:         JSONUTCTime(s.UpdatedAt),
	}
	return resp
}
