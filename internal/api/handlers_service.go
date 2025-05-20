package api

import (
	"context"
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

// Routes returns the router with all service routes registered
func (h *ServiceHandler) Routes() func(r chi.Router) {

	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(IDMiddleware)
			r.Get("/{id}", h.handleGet)
			r.Patch("/{id}", h.handleUpdate)
			r.Post("/{id}/start", h.handleStart)
			r.Post("/{id}/stop", h.handleStop)
			r.Delete("/{id}", h.handleDelete)
			r.Post("/{id}/retry", h.handleRetry)
		})
	}
}

func (h *ServiceHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var p struct {
		GroupID       domain.UUID       `json:"groupId"`
		AgentID       domain.UUID       `json:"agentId"`
		ServiceTypeID domain.UUID       `json:"serviceTypeId"`
		Name          string            `json:"name"`
		Attributes    domain.Attributes `json:"attributes"`
		Properties    domain.JSON       `json:"properties"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	serviceGroupScope, err := h.serviceGroupQuerier.AuthScope(r.Context(), p.GroupID)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	agentScope, err := h.agentQuerier.AuthScope(r.Context(), p.AgentID)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	scope := &domain.AuthScope{ConsumerID: serviceGroupScope.ConsumerID, ParticipantID: agentScope.ParticipantID, AgentID: &p.AgentID}
	if err := h.authz.AuthorizeCtx(r.Context(), domain.SubjectService, domain.ActionCreate, scope); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	service, err := h.commander.Create(
		r.Context(),
		p.AgentID,
		p.ServiceTypeID,
		p.GroupID,
		p.Name,
		p.Attributes,
		p.Properties,
	)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)

	_, err := h.authorize(r.Context(), id, domain.ActionRead)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}

	service, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())

	if err := h.authz.Authorize(id, domain.SubjectService, domain.ActionRead, &domain.EmptyAuthScope); err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}

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
	id := MustGetID(r)

	_, err := h.authorize(r.Context(), id, domain.ActionUpdate)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}

	var p struct {
		Name       *string              `json:"name"`
		State      *domain.ServiceState `json:"state"`
		Properties *domain.JSON         `json:"properties"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	service, err := h.commander.Update(
		r.Context(),
		id,
		p.Name,
		p.Properties,
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
	id := MustGetID(r)

	var action domain.AuthAction
	switch t {
	case domain.ServiceStarted:
		action = domain.ActionStart
	case domain.ServiceStopped:
		action = domain.ActionStop
	case domain.ServiceDeleted:
		action = domain.ActionDelete
	default:
		action = domain.ActionUpdate
	}

	if _, err := h.authorize(r.Context(), id, action); err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}

	if _, err := h.commander.Transition(r.Context(), id, t); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ServiceHandler) handleRetry(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionUpdate)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
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
	BrokerID          domain.UUID           `json:"brokerId"`
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
		BrokerID:          s.ConsumerID,
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

func (h *ServiceHandler) authorize(ctx context.Context, id domain.UUID, action domain.AuthAction) (*domain.AuthScope, error) {
	scope, err := h.querier.AuthScope(ctx, id)
	if err != nil {
		return nil, err
	}
	err = h.authz.AuthorizeCtx(ctx, domain.SubjectService, action, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}
