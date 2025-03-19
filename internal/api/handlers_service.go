package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ServiceHandler struct {
	querier   domain.ServiceQuerier
	commander domain.ServiceCommander
}

func NewServiceHandler(
	querier domain.ServiceQuerier,
	commander domain.ServiceCommander,
) *ServiceHandler {
	return &ServiceHandler{
		querier:   querier,
		commander: commander,
	}
}

// Routes returns the router with all service routes registered
func (h *ServiceHandler) Routes(authzMW AuthzMiddlewareFunc) func(r chi.Router) {
	return func(r chi.Router) {
		r.With(authzMW(domain.SubjectService, domain.ActionList)).Get("/", h.handleList)
		r.With(authzMW(domain.SubjectService, domain.ActionCreate)).Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(UUIDMiddleware)
			r.With(authzMW(domain.SubjectService, domain.ActionRead)).Get("/{id}", h.handleGet)
			r.With(authzMW(domain.SubjectService, domain.ActionUpdate)).Patch("/{id}", h.handleUpdate)
			r.With(authzMW(domain.SubjectService, domain.ActionStart)).Post("/{id}/start", h.handleStart)
			r.With(authzMW(domain.SubjectService, domain.ActionStop)).Post("/{id}/stop", h.handleStop)
			r.With(authzMW(domain.SubjectService, domain.ActionDelete)).Delete("/{id}", h.handleDelete)
			r.With(authzMW(domain.SubjectService, domain.ActionUpdate)).Post("/{id}/retry", h.handleRetry)
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
	id := MustGetUUIDParam(r)
	service, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleList(w http.ResponseWriter, r *http.Request) {
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	result, err := h.querier.List(r.Context(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, NewPageResponse(result, serviceToResponse))
}

func (h *ServiceHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
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
	id := MustGetUUIDParam(r)
	if _, err := h.commander.Transition(r.Context(), id, t); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ServiceHandler) handleRetry(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
	if _, err := h.commander.Retry(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ServiceResponse represents the response body for service operations
type ServiceResponse struct {
	ID                domain.UUID           `json:"id"`
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

	// Relationships
	Agent       *AgentResponse        `json:"agent,omitempty"`
	ServiceType *ServiceTypeResponse  `json:"serviceType,omitempty"`
	Group       *ServiceGroupResponse `json:"group,omitempty"`
}

// serviceToResponse converts a domain.Service to a ServiceResponse
func serviceToResponse(s *domain.Service) *ServiceResponse {
	resp := &ServiceResponse{
		ID:                s.ID,
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

	if s.Agent != nil {
		resp.Agent = agentToResponse(s.Agent)
	}
	if s.ServiceType != nil {
		resp.ServiceType = serviceTypeToResponse(s.ServiceType)
	}
	if s.Group != nil {
		resp.Group = serviceGroupToResponse(s.Group)
	}
	return resp
}
