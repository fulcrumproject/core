package api

import (
	"context"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type AgentHandler struct {
	querier   domain.AgentQuerier
	commander domain.AgentCommander
	authz     domain.Authorizer
}

func NewAgentHandler(
	querier domain.AgentQuerier,
	commander domain.AgentCommander,
	authz domain.Authorizer,
) *AgentHandler {
	return &AgentHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all agent routes registered
func (h *AgentHandler) Routes() func(r chi.Router) {

	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(IDMiddleware)
			r.Get("/{id}", h.handleGet)
			r.Patch("/{id}", h.handleUpdate)
			r.Delete("/{id}", h.handleDelete)
		})
		r.Put("/me/status", h.handleUpdateStatusMe)
		r.Get("/me", h.handleGetMe)
	}
}

func (h *AgentHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Name          string             `json:"name"`
		CountryCode   domain.CountryCode `json:"countryCode,omitempty"`
		Attributes    domain.Attributes  `json:"attributes,omitempty"`
		ParticipantID domain.UUID        `json:"providerId"`
		AgentTypeID   domain.UUID        `json:"agentTypeId"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	scope := domain.AuthScope{ParticipantID: &p.ParticipantID}
	if err := h.authz.AuthorizeCtx(r.Context(), domain.SubjectAgent, domain.ActionCreate, &scope); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	agent, err := h.commander.Create(
		r.Context(),
		p.Name,
		p.CountryCode,
		p.Attributes,
		p.ParticipantID,
		p.AgentTypeID,
	)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionRead)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	agent, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, agentToResponse(agent))
}

// handleGetMe handles GET /agents/me
// This endpoint allows agents to retrieve their own information
func (h *AgentHandler) handleGetMe(w http.ResponseWriter, r *http.Request) {
	agentID, err := MustGetAgentID(r.Context())
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	agent, err := h.querier.FindByID(r.Context(), agentID)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
	if err := h.authz.Authorize(id, domain.SubjectAgent, domain.ActionRead, &domain.EmptyAuthScope); err != nil {
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
	render.JSON(w, r, NewPageResponse(result, agentToResponse))
}

func (h *AgentHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionUpdate)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	var p struct {
		Name        *string             `json:"name"`
		State       *domain.AgentState  `json:"state"`
		CountryCode *domain.CountryCode `json:"countryCode,omitempty"`
		Attributes  *domain.Attributes  `json:"attributes,omitempty"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	agent, err := h.commander.Update(
		r.Context(),
		id,
		p.Name,
		p.CountryCode,
		p.Attributes,
		p.State,
	)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, agentToResponse(agent))
}

// handleUpdateStatusMe handles PUT /agents/me/status
// This endpoint allows agents to update their own status
func (h *AgentHandler) handleUpdateStatusMe(w http.ResponseWriter, r *http.Request) {
	var p struct {
		State domain.AgentState `json:"state"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	agentID, err := MustGetAgentID(r.Context())
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	agent, err := h.commander.UpdateState(r.Context(), agentID, p.State)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionDelete)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	_, err = h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AgentResponse represents the response body for agent operations
type AgentResponse struct {
	ID            domain.UUID          `json:"id"`
	Name          string               `json:"name"`
	State         domain.AgentState    `json:"state"`
	CountryCode   domain.CountryCode   `json:"countryCode,omitempty"`
	Attributes    domain.Attributes    `json:"attributes,omitempty"`
	ParticipantID domain.UUID          `json:"participantId"`
	AgentTypeID   domain.UUID          `json:"agentTypeId"`
	Participant   *ParticipantResponse `json:"participant,omitempty"`
	AgentType     *AgentTypeResponse   `json:"agentType,omitempty"`
	CreatedAt     JSONUTCTime          `json:"createdAt"`
	UpdatedAt     JSONUTCTime          `json:"updatedAt"`
}

// agentToResponse converts a domain.Agent to an AgentResponse
func agentToResponse(a *domain.Agent) *AgentResponse {
	response := &AgentResponse{
		ID:            a.ID,
		Name:          a.Name,
		State:         a.State,
		CountryCode:   a.CountryCode,
		Attributes:    map[string][]string(a.Attributes),
		ParticipantID: a.ParticipantID,
		AgentTypeID:   a.AgentTypeID,
		CreatedAt:     JSONUTCTime(a.CreatedAt),
		UpdatedAt:     JSONUTCTime(a.UpdatedAt),
	}
	if a.Participant != nil {
		response.Participant = participantToResponse(a.Participant)
	}
	if a.AgentType != nil {
		response.AgentType = agentTypeToResponse(a.AgentType)
	}
	return response
}

// TODO review with agents/me
func MustGetAgentID(ctx context.Context) (domain.UUID, error) {
	id := domain.MustGetAuthIdentity(ctx)
	if !id.IsRole(domain.RoleAgent) {
		return uuid.Nil, domain.NewUnauthorizedErrorf("must be authenticated as agent")
	}
	if id.Scope().AgentID == nil {
		return uuid.Nil, domain.NewUnauthorizedErrorf("agent with nil scope")
	}
	return *id.Scope().AgentID, nil
}

func (h *AgentHandler) authorize(ctx context.Context, id domain.UUID, action domain.AuthAction) (*domain.AuthScope, error) {
	scope, err := h.querier.AuthScope(ctx, id)
	if err != nil {
		return nil, err
	}
	err = h.authz.AuthorizeCtx(ctx, domain.SubjectAgent, action, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}
