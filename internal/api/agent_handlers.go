package api

import (
	"net/http"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// CreateUpdateAgentRequest represents the request body for creating/updating an agent
type CreateUpdateAgentRequest struct {
	Name        string                 `json:"name"`
	State       domain.AgentState      `json:"state"`
	TokenHash   string                 `json:"tokenHash"`
	CountryCode string                 `json:"countryCode,omitempty"`
	Attributes  map[string][]string    `json:"attributes,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	ProviderID  string                 `json:"providerId"`
	AgentTypeID string                 `json:"agentTypeId"`
}

// AgentResponse represents the response body for agent operations
type AgentResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	State       domain.AgentState      `json:"state"`
	CountryCode string                 `json:"countryCode,omitempty"`
	Attributes  map[string][]string    `json:"attributes,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	ProviderID  string                 `json:"providerId"`
	AgentTypeID string                 `json:"agentTypeId"`
	Provider    *ProviderResponse      `json:"provider,omitempty"`
	AgentType   *AgentTypeResponse     `json:"agentType,omitempty"`
	CreatedAt   JSONUTCTime            `json:"createdAt"`
	UpdatedAt   JSONUTCTime            `json:"updatedAt"`
}

// AgentCreateResponse extends AgentResponse with a token field
type AgentCreateResponse struct {
	*AgentResponse
	Token string `json:"token,omitempty"` // Only included in creation response
}

// agentToResponse converts a domain.Agent to an AgentResponse
func agentToResponse(a *domain.Agent) *AgentResponse {
	response := &AgentResponse{
		ID:          a.ID.String(),
		Name:        a.Name,
		State:       a.State,
		CountryCode: a.CountryCode,
		Attributes:  map[string][]string(a.Attributes),
		Properties:  a.Properties,
		ProviderID:  a.ProviderID.String(),
		AgentTypeID: a.AgentTypeID.String(),
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

type AgentHandler struct {
	repo domain.AgentRepository
}

func NewAgentHandler(repo domain.AgentRepository) *AgentHandler {
	return &AgentHandler{repo: repo}
}

// Routes returns the router with all agent routes registered
func (h *AgentHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware for all agent routes
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", h.handleList)
	r.Post("/", h.handleCreate)
	r.Get("/{id}", h.handleGet)
	r.Put("/{id}", h.handleUpdate)
	r.Delete("/{id}", h.handleDelete)
	r.Post("/{id}/rotate-token", h.handleRotateToken) // New endpoint

	// Agent-authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(AgentAuthMiddleware(h.repo))
		r.Put("/me/status", h.handleUpdateStatus) // Endpoint for agents to update their status
	})

	return r
}

func (h *AgentHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateUpdateAgentRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	providerID, err := domain.ParseUUID(req.ProviderID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agentTypeID, err := domain.ParseUUID(req.AgentTypeID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agent := &domain.Agent{
		Name:  req.Name,
		State: req.State,
		// Token is now generated, not provided in the request
		CountryCode: req.CountryCode,
		Attributes:  domain.Attributes(req.Attributes),
		Properties:  req.Properties,
		ProviderID:  providerID,
		AgentTypeID: agentTypeID,
	}

	if !agent.State.IsValid() {
		render.Render(w, r, ErrInvalidRequest(domain.ErrInvalidAgentState))
		return
	}

	// Generate a secure token for the agent
	token, err := agent.GenerateToken()
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	if err := h.repo.Create(r.Context(), agent); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	// Return the agent with the token (one time only)
	resp := &AgentCreateResponse{
		AgentResponse: agentToResponse(agent),
		Token:         token,
	}
	render.JSON(w, r, resp)
	render.Status(r, http.StatusCreated)
}

func (h *AgentHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agent, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
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

	result, err := h.repo.List(r.Context(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, agentToResponse))
}

func (h *AgentHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	var req CreateUpdateAgentRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agent, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	providerID, err := domain.ParseUUID(req.ProviderID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agentTypeID, err := domain.ParseUUID(req.AgentTypeID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Update fields
	agent.Name = req.Name
	agent.State = req.State
	// Keep existing token hash
	currentTokenHash := agent.TokenHash
	agent.CountryCode = req.CountryCode
	agent.TokenHash = currentTokenHash // Preserve the token hash
	agent.Properties = req.Properties
	agent.ProviderID = providerID
	agent.AgentTypeID = agentTypeID

	if !agent.State.IsValid() {
		render.Render(w, r, ErrInvalidRequest(domain.ErrInvalidAgentState))
		return
	}

	if err := h.repo.Save(r.Context(), agent); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	_, err = h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleRotateToken handles POST /agents/{id}/rotate-token
func (h *AgentHandler) handleRotateToken(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agent, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	// Generate a new token
	token, err := agent.GenerateToken()
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	// Save the agent with the new token hash
	if err := h.repo.Save(r.Context(), agent); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	// Return the new token
	resp := &AgentCreateResponse{
		AgentResponse: agentToResponse(agent),
		Token:         token,
	}
	render.JSON(w, r, resp)
}

// UpdateAgentStatusRequest represents the request body for updating agent status
type UpdateAgentStatusRequest struct {
	State domain.AgentState `json:"state"`
}

// handleUpdateStatus handles PUT /agents/me/status
// This endpoint allows agents to update their own status
func (h *AgentHandler) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	// Get authenticated agent from context (set by middleware)
	agent := GetAuthenticatedAgent(r)
	if agent == nil {
		render.Render(w, r, ErrUnauthorized())
		return
	}

	// Parse the request body
	var req UpdateAgentStatusRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Update the agent's state
	agent.State = req.State
	agent.LastStatusUpdate = time.Now()
	if err := h.repo.Save(r.Context(), agent); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.JSON(w, r, agentToResponse(agent))
}
