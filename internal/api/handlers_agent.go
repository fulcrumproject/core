package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateAgentRequest struct {
	Name        string             `json:"name"`
	CountryCode domain.CountryCode `json:"countryCode,omitempty"`
	Attributes  domain.Attributes  `json:"attributes,omitempty"`
	ProviderID  domain.UUID        `json:"providerId"`
	AgentTypeID domain.UUID        `json:"agentTypeId"`
}

// AuthTargetScope implements AuthTargetScopeProvider interface
func (r CreateAgentRequest) AuthTargetScope() (*domain.AuthTargetScope, error) {
	return &domain.AuthTargetScope{ParticipantID: &r.ProviderID}, nil
}

type UpdateAgentRequest struct {
	Name        *string             `json:"name"`
	State       *domain.AgentState  `json:"state"`
	CountryCode *domain.CountryCode `json:"countryCode,omitempty"`
	Attributes  *domain.Attributes  `json:"attributes,omitempty"`
}

type UpdateAgentStatusRequest struct {
	State domain.AgentState `json:"state"`
}

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

func (h *AgentHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			AuthzSimple(domain.SubjectAgent, domain.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create endpoint - decode body, then authorize with provider ID
		r.With(
			DecodeBody[CreateAgentRequest](),
			AuthzFromBody[CreateAgentRequest](domain.SubjectAgent, domain.ActionCreate, h.authz),
		).Post("/", h.handleCreate)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(ID)

			// Get endpoint - authorize using agent's provider
			r.With(
				AuthzFromID(domain.SubjectAgent, domain.ActionRead, h.authz, h.querier),
			).Get("/{id}", h.handleGet)

			// Update endpoint - decode body, authorize using agent's provider
			r.With(
				DecodeBody[UpdateAgentRequest](),
				AuthzFromID(domain.SubjectAgent, domain.ActionUpdate, h.authz, h.querier),
			).Patch("/{id}", h.handleUpdate)

			// Delete endpoint - authorize using agent's provider
			r.With(
				AuthzFromID(domain.SubjectAgent, domain.ActionDelete, h.authz, h.querier),
			).Delete("/{id}", h.handleDelete)
		})

		// Agent-specific routes (me endpoints)
		// Note: These endpoints have special auth requirements
		r.With(
			RequireAgentIdentity(),
			DecodeBody[UpdateAgentStatusRequest](),
		).Put("/me/status", h.handleUpdateStatusMe)

		r.With(
			RequireAgentIdentity(),
		).Get("/me", h.handleGetMe)
	}
}

func (h *AgentHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	p := MustGetBody[CreateAgentRequest](r.Context())

	agent, err := h.commander.Create(
		r.Context(),
		p.Name,
		p.CountryCode,
		p.Attributes,
		p.ProviderID,
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
	id := MustGetID(r.Context())

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
	agentID := MustGetAgentID(r.Context())

	agent, err := h.querier.FindByID(r.Context(), agentID)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleList(w http.ResponseWriter, r *http.Request) {
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

	render.JSON(w, r, NewPageResponse(result, agentToResponse))
}

func (h *AgentHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())
	p := MustGetBody[UpdateAgentRequest](r.Context())

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
	p := MustGetBody[UpdateAgentStatusRequest](r.Context())
	agentID := MustGetAgentID(r.Context())

	agent, err := h.commander.UpdateState(r.Context(), agentID, p.State)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AgentResponse represents the response body for agent operations
type AgentResponse struct {
	ID          domain.UUID          `json:"id"`
	Name        string               `json:"name"`
	State       domain.AgentState    `json:"state"`
	CountryCode domain.CountryCode   `json:"countryCode,omitempty"`
	Attributes  domain.Attributes    `json:"attributes,omitempty"`
	ProviderID  domain.UUID          `json:"providerId"`
	AgentTypeID domain.UUID          `json:"agentTypeId"`
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
		State:       a.State,
		CountryCode: a.CountryCode,
		Attributes:  map[string][]string(a.Attributes),
		ProviderID:  a.ProviderID,
		AgentTypeID: a.AgentTypeID,
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
