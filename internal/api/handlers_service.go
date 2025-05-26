package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ServiceHandler struct {
	querier             domain.ServiceQuerier
	agentQuerier        domain.AgentQuerier
	serviceGroupQuerier domain.ServiceGroupQuerier
	commander           domain.ServiceCommander
	authz               domain.Authorizer
}

func NewServiceHandler(
	querier domain.ServiceQuerier,
	agentQuerier domain.AgentQuerier,
	serviceGroupQuerier domain.ServiceGroupQuerier,
	commander domain.ServiceCommander,
	authz domain.Authorizer,
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
	GroupID       domain.UUID       `json:"groupId"`
	AgentID       domain.UUID       `json:"agentId"`
	ServiceTypeID domain.UUID       `json:"serviceTypeId"`
	Name          string            `json:"name"`
	Attributes    domain.Attributes `json:"attributes"`
	Properties    domain.JSON       `json:"properties"`
}

// UpdateServiceRequest represents the request to update a service
type UpdateServiceRequest struct {
	Name       *string      `json:"name,omitempty"`
	Properties *domain.JSON `json:"properties,omitempty"`
}

// ServiceActionRequest represents a state transition request
type ServiceActionRequest struct {
	Action string `json:"action"`
}

// CreateServiceScopeExtractor creates an extractor that gets a combined scope from the request body
// by retrieving scopes from both ServiceGroup and Agent
func CreateServiceScopeExtractor(
	serviceGroupQuerier domain.ServiceGroupQuerier,
	agentQuerier domain.AgentQuerier,
) AuthTargetScopeExtractor {
	return func(r *http.Request) (*domain.AuthTargetScope, error) {
		// Get decoded body from context
		body := MustGetBody[CreateServiceRequest](r.Context())

		// Get service group scope
		serviceGroupScope, err := serviceGroupQuerier.AuthScope(r.Context(), body.GroupID)
		if err != nil {
			return nil, err
		}

		// Get agent scope
		agentScope, err := agentQuerier.AuthScope(r.Context(), body.AgentID)
		if err != nil {
			return nil, err
		}

		// Combine the scopes
		scope := &domain.AuthTargetScope{
			ConsumerID:    serviceGroupScope.ConsumerID,
			ParticipantID: agentScope.ParticipantID,
			AgentID:       &body.AgentID,
		}

		return scope, nil
	}
}

// Routes returns the router with all service routes registered
func (h *ServiceHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List - simple authorization
		r.With(
			AuthzSimple(domain.SubjectService, domain.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create - decode body + specialized scope extractor for authorization
		r.With(
			DecodeBody[CreateServiceRequest](),
			AuthzFromExtractor(
				domain.SubjectService,
				domain.ActionCreate,
				h.authz,
				CreateServiceScopeExtractor(h.serviceGroupQuerier, h.agentQuerier),
			),
		).Post("/", h.handleCreate)

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(ID)

			// Get - authorize from resource ID
			r.With(
				AuthzFromID(domain.SubjectService, domain.ActionRead, h.authz, h.querier),
			).Get("/{id}", h.handleGet)

			// Update - decode body + authorize from resource ID
			r.With(
				DecodeBody[UpdateServiceRequest](),
				AuthzFromID(domain.SubjectService, domain.ActionUpdate, h.authz, h.querier),
			).Patch("/{id}", h.handleUpdate)

			// Start - authorize from resource ID
			r.With(
				AuthzFromID(domain.SubjectService, domain.ActionStart, h.authz, h.querier),
			).Post("/{id}/start", h.handleStart)

			// Stop - authorize from resource ID
			r.With(
				AuthzFromID(domain.SubjectService, domain.ActionStop, h.authz, h.querier),
			).Post("/{id}/stop", h.handleStop)

			// Delete - authorize from resource ID
			r.With(
				AuthzFromID(domain.SubjectService, domain.ActionDelete, h.authz, h.querier),
			).Delete("/{id}", h.handleDelete)

			// Retry - authorize from resource ID
			r.With(
				AuthzFromID(domain.SubjectService, domain.ActionUpdate, h.authz, h.querier),
			).Post("/{id}/retry", h.handleRetry)
		})
	}
}

func (h *ServiceHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	// Get decoded body from context
	body := MustGetBody[CreateServiceRequest](r.Context())

	service, err := h.commander.Create(
		r.Context(),
		body.AgentID,
		body.ServiceTypeID,
		body.GroupID,
		body.Name,
		body.Attributes,
		body.Properties,
	)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	service, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())

	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	result, err := h.querier.List(r.Context(), id.Scope(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, serviceToResponse))
}

func (h *ServiceHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())
	body := MustGetBody[UpdateServiceRequest](r.Context())

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

func (h *ServiceHandler) handleTransition(w http.ResponseWriter, r *http.Request, t domain.ServiceState) {
	id := MustGetID(r.Context())

	if _, err := h.commander.Transition(r.Context(), id, t); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ServiceHandler) handleRetry(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	if _, err := h.commander.Retry(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ServiceResponse represents the response body for service operations
type ServiceResponse struct {
	ID                domain.UUID           `json:"id"`
	ProviderID        domain.UUID           `json:"providerId"`
	ConsumerID        domain.UUID           `json:"consumerId"`
	AgentID           domain.UUID           `json:"agentId"`
	ServiceTypeID     domain.UUID           `json:"serviceTypeId"`
	GroupID           domain.UUID           `json:"groupId"`
	ExternalID        *string               `json:"externalId,omitempty"`
	Name              string                `json:"name"`
	Attributes        domain.Attributes     `json:"attributes"`
	CurrentState      domain.ServiceState   `json:"currentState"`
	TargetState       *domain.ServiceState  `json:"targetState,omitempty"`
	FailedAction      *domain.ServiceAction `json:"failedAction,omitempty"`
	ErrorMessage      *string               `json:"errorMessage,omitempty"`
	RetryCount        int                   `json:"retryCount,omitempty"`
	CurrentProperties *domain.JSON          `json:"currentProperties,omitempty"`
	TargetProperties  *domain.JSON          `json:"targetProperties,omitempty"`
	Resources         *domain.JSON          `json:"resources,omitempty"`
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
		Attributes:        s.Attributes,
		CurrentState:      s.CurrentState,
		TargetState:       s.TargetState,
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
