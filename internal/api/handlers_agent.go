package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type AgentHandler struct {
	querier   domain.AgentQuerier
	commander *domain.AgentCommander
}

func NewAgentHandler(
	querier domain.AgentQuerier,
	commander *domain.AgentCommander,
) *AgentHandler {
	return &AgentHandler{
		querier:   querier,
		commander: commander,
	}
}

// Routes returns the router with all agent routes registered
func (h *AgentHandler) Routes(agentAuthMw func(http.Handler) http.Handler) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(UUIDMiddleware)
			r.Get("/{id}", h.handleGet)
			r.Patch("/{id}", h.handleUpdate)
			r.Delete("/{id}", h.handleDelete)
			r.Post("/{id}/rotate-token", h.handleRotateToken)
		})
		// Agent-authenticated routes TODO remove with global auth
		r.Group(func(r chi.Router) {
			r.Use(agentAuthMw)
			r.Put("/me/status", h.handleUpdateStatus) // Endpoint for agents to update their status
			r.Get("/me", h.handleGetMe)               // Endpoint for agents to get their own information
		})
	}
}

func (h *AgentHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Name        string             `json:"name"`
		CountryCode domain.CountryCode `json:"countryCode,omitempty"`
		Attributes  domain.Attributes  `json:"attributes,omitempty"`
		ProviderID  domain.UUID        `json:"providerId"`
		AgentTypeID domain.UUID        `json:"agentTypeId"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
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
	// Return the agent with the token (one time only)
	resp := struct {
		*AgentResponse
		Token string `json:"token,omitempty"`
	}{
		AgentResponse: agentToResponse(agent),
		Token:         agent.Token,
	}
	render.JSON(w, r, resp)
	render.Status(r, http.StatusCreated)
}

func (h *AgentHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
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
	// Get authenticated agent from context (set by middleware)
	agent := GetAuthenticatedAgent(r)
	if agent == nil {
		render.Render(w, r, ErrUnauthorized())
		return
	}
	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleList(w http.ResponseWriter, r *http.Request) {
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
	render.JSON(w, r, NewPageResponse(result, agentToResponse))
}

func (h *AgentHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
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

func (h *AgentHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
	_, err := h.querier.FindByID(r.Context(), id)
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

// handleRotateToken handles POST /agents/{id}/rotate-token
func (h *AgentHandler) handleRotateToken(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
	agent, err := h.commander.RotateToken(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, agentToResponse(agent))
}

// handleUpdateStatus handles PUT /agents/me/status
// This endpoint allows agents to update their own status
func (h *AgentHandler) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	agent := GetAuthenticatedAgent(r)
	if agent == nil {
		render.Render(w, r, ErrUnauthorized())
		return
	}
	var p struct {
		State domain.AgentState `json:"state"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	agent, err := h.commander.UpdateState(r.Context(), agent.ID, p.State)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, agentToResponse(agent))
}

// AgentResponse represents the response body for agent operations
type AgentResponse struct {
	ID          domain.UUID        `json:"id"`
	Name        string             `json:"name"`
	State       domain.AgentState  `json:"state"`
	Token       string             `json:"token,omitempty"`
	CountryCode domain.CountryCode `json:"countryCode,omitempty"`
	Attributes  domain.Attributes  `json:"attributes,omitempty"`
	ProviderID  domain.UUID        `json:"providerId"`
	AgentTypeID domain.UUID        `json:"agentTypeId"`
	Provider    *ProviderResponse  `json:"provider,omitempty"`
	AgentType   *AgentTypeResponse `json:"agentType,omitempty"`
	CreatedAt   JSONUTCTime        `json:"createdAt"`
	UpdatedAt   JSONUTCTime        `json:"updatedAt"`
}

// agentToResponse converts a domain.Agent to an AgentResponse
func agentToResponse(a *domain.Agent) *AgentResponse {
	response := &AgentResponse{
		ID:          a.ID,
		Name:        a.Name,
		State:       a.State,
		Token:       a.Token,
		CountryCode: a.CountryCode,
		Attributes:  map[string][]string(a.Attributes),
		ProviderID:  a.ProviderID,
		AgentTypeID: a.AgentTypeID,
		CreatedAt:   JSONUTCTime(a.CreatedAt),
		UpdatedAt:   JSONUTCTime(a.UpdatedAt),
	}
	if a.Provider != nil {
		response.Provider = provderToResponse(a.Provider)
	}
	if a.AgentType != nil {
		response.AgentType = agentTypeToResponse(a.AgentType)
	}
	return response
}
