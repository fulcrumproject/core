package api

import (
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

// CreateServiceRequest represents the request to create a service
type CreateServiceRequest struct {
	GroupID       properties.UUID  `json:"groupId"`
	AgentID       *properties.UUID `json:"agentId,omitempty"`
	ServiceTypeID properties.UUID  `json:"serviceTypeId"`
	AgentTags     []string         `json:"agentTags,omitempty"`
	Name          string           `json:"name"`
	Properties    properties.JSON  `json:"properties"`
}

// UpdateServiceRequest represents the request to update a service
type UpdateServiceRequest struct {
	Name       *string          `json:"name,omitempty"`
	Properties *properties.JSON `json:"properties,omitempty"`
}

// ServiceActionRequest represents a status transition request
type ServiceActionRequest struct {
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
		body := middlewares.MustGetBody[CreateServiceRequest](r.Context())

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
		).Get("/", h.handleList)

		// Create - decode body + specialized scope extractor for authorization
		r.With(
			middlewares.DecodeBody[CreateServiceRequest](),
			middlewares.AuthzFromExtractor(
				authz.ObjectTypeService,
				authz.ActionCreate,
				h.authz,
				CreateServiceScopeExtractor(h.serviceGroupQuerier, h.agentQuerier),
			),
		).Post("/", h.handleCreate)

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", h.handleGet)

			// Update - decode body + authorize from resource ID
			r.With(
				middlewares.DecodeBody[UpdateServiceRequest](),
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", h.handleUpdate)

			// Start - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionStart, h.authz, h.querier.AuthScope),
			).Post("/{id}/start", h.handleStart)

			// Stop - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionStop, h.authz, h.querier.AuthScope),
			).Post("/{id}/stop", h.handleStop)

			// Delete - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", h.handleDelete)

			// Retry - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Post("/{id}/retry", h.handleRetry)
		})
	}
}

func (h *ServiceHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	// Get decoded body from context
	body := middlewares.MustGetBody[CreateServiceRequest](r.Context())

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
	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	service, err := h.querier.Get(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := auth.MustGetIdentity(r.Context())

	pag, err := ParsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	result, err := h.querier.List(r.Context(), &id.Scope, pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, serviceToResponse))
}

func (h *ServiceHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())
	body := middlewares.MustGetBody[UpdateServiceRequest](r.Context())

	service, err := h.commander.Update(
		r.Context(),
		id,
		body.Name,
		body.Properties,
	)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleStart(w http.ResponseWriter, r *http.Request) {
	h.handleTransition(w, r, domain.ServiceStarted)
}

func (h *ServiceHandler) handleStop(w http.ResponseWriter, r *http.Request) {
	h.handleTransition(w, r, domain.ServiceStopped)
}

func (h *ServiceHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	h.handleTransition(w, r, domain.ServiceDeleted)
}

func (h *ServiceHandler) handleTransition(w http.ResponseWriter, r *http.Request, t domain.ServiceStatus) {
	id := middlewares.MustGetID(r.Context())

	if _, err := h.commander.Transition(r.Context(), id, t); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ServiceHandler) handleRetry(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	if _, err := h.commander.Retry(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ServiceResponse represents the response body for service operations
type ServiceResponse struct {
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

// serviceToResponse converts a domain.Service to a ServiceResponse
func serviceToResponse(s *domain.Service) *ServiceResponse {
	resp := &ServiceResponse{
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
