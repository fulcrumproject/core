package api

import (
	"net/http"

	"fulcrumproject.org/core/pkg/authz"
	"fulcrumproject.org/core/pkg/domain"
	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/middlewares"
	"github.com/fulcrumproject/commons/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateAgentRequest struct {
	Name        string          `json:"name"`
	ProviderID  properties.UUID `json:"providerId"`
	AgentTypeID properties.UUID `json:"agentTypeId"`
	Tags        []string        `json:"tags"`
}

// auth.ObjectScope implements auth.ObjectScopeProvider interface
func (r CreateAgentRequest) ObjectScope() (auth.ObjectScope, error) {
	return &auth.DefaultObjectScope{ParticipantID: &r.ProviderID}, nil
}

type UpdateAgentRequest struct {
	Name   *string             `json:"name"`
	Status *domain.AgentStatus `json:"status"`
	Tags   *[]string           `json:"tags"`
}

type UpdateAgentStatusRequest struct {
	Status domain.AgentStatus `json:"status"`
}

type AgentHandler struct {
	querier   domain.AgentQuerier
	commander domain.AgentCommander
	authz     auth.Authorizer
}

func NewAgentHandler(
	querier domain.AgentQuerier,
	commander domain.AgentCommander,
	authz auth.Authorizer,
) *AgentHandler {
	return &AgentHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

func (h *AgentHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeAgent, authz.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create endpoint - decode body, then authorize with provider ID
		r.With(
			middlewares.DecodeBody[CreateAgentRequest](),
			middlewares.AuthzFromBody[CreateAgentRequest](authz.ObjectTypeAgent, authz.ActionCreate, h.authz),
		).Post("/", h.handleCreate)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using agent's provider
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgent, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", h.handleGet)

			// Update endpoint - decode body, authorize using agent's provider
			r.With(
				middlewares.DecodeBody[UpdateAgentRequest](),
				middlewares.AuthzFromID(authz.ObjectTypeAgent, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", h.handleUpdate)

			// Delete endpoint - authorize using agent's provider
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgent, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", h.handleDelete)
		})

		// Agent-specific routes (me endpoints)
		// Note: These endpoints have special auth requirements
		r.With(
			middlewares.MustHaveRoles(auth.RoleAgent),
			middlewares.DecodeBody[UpdateAgentStatusRequest](),
		).Put("/me/status", h.handleUpdateStatusMe)

		r.With(
			middlewares.MustHaveRoles(auth.RoleAgent),
		).Get("/me", h.handleGetMe)
	}
}

func (h *AgentHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	p := middlewares.MustGetBody[CreateAgentRequest](r.Context())

	agent, err := h.commander.Create(
		r.Context(),
		p.Name,
		p.ProviderID,
		p.AgentTypeID,
		p.Tags,
	)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	agent, err := h.querier.Get(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, agentToResponse(agent))
}

// handleGetMe handles GET /agents/me
// This endpoint allows agents to retrieve their own information
func (h *AgentHandler) handleGetMe(w http.ResponseWriter, r *http.Request) {
	agentID := auth.MustGetIdentity(r.Context()).Scope.AgentID

	agent, err := h.querier.Get(r.Context(), *agentID)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := auth.MustGetIdentity(r.Context())
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	result, err := h.querier.List(r.Context(), &id.Scope, pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, agentToResponse))
}

func (h *AgentHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())
	p := middlewares.MustGetBody[UpdateAgentRequest](r.Context())

	agent, err := h.commander.Update(
		r.Context(),
		id,
		p.Name,
		p.Status,
		p.Tags,
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
	p := middlewares.MustGetBody[UpdateAgentStatusRequest](r.Context())
	agentID := auth.MustGetIdentity(r.Context()).Scope.AgentID

	agent, err := h.commander.UpdateStatus(r.Context(), *agentID, p.Status)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AgentResponse represents the response body for agent operations
type AgentResponse struct {
	ID          properties.UUID      `json:"id"`
	Name        string               `json:"name"`
	Status      domain.AgentStatus   `json:"status"`
	ProviderID  properties.UUID      `json:"providerId"`
	AgentTypeID properties.UUID      `json:"agentTypeId"`
	Tags        []string             `json:"tags"`
	Participant *ParticipantResponse `json:"participant,omitempty"`
	AgentType   *AgentTypeResponse   `json:"agentType,omitempty"`
	CreatedAt   JSONUTCTime          `json:"createdAt"`
	UpdatedAt   JSONUTCTime          `json:"updatedAt"`
}

// agentToResponse converts a domain.Agent to an AgentResponse
func agentToResponse(a *domain.Agent) *AgentResponse {
	response := &AgentResponse{
		ID:          a.ID,
		Name:        a.Name,
		Status:      a.Status,
		ProviderID:  a.ProviderID,
		AgentTypeID: a.AgentTypeID,
		Tags:        []string(a.Tags),
		CreatedAt:   JSONUTCTime(a.CreatedAt),
		UpdatedAt:   JSONUTCTime(a.UpdatedAt),
	}
	if a.Provider != nil {
		response.Participant = participantToResponse(a.Provider)
	}
	if a.AgentType != nil {
		response.AgentType = agentTypeToResponse(a.AgentType)
	}
	return response
}
