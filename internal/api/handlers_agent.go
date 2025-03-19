package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type AgentHandler struct {
	querier   domain.AgentQuerier
	commander domain.AgentCommander
}

func NewAgentHandler(
	querier domain.AgentQuerier,
	commander domain.AgentCommander,
) *AgentHandler {
	return &AgentHandler{
		querier:   querier,
		commander: commander,
	}
}

// Routes returns the router with all agent routes registered
func (h *AgentHandler) Routes(authzMW AuthzMiddlewareFunc) func(r chi.Router) {
	return func(r chi.Router) {
		r.With(authzMW(domain.SubjectAgent, domain.ActionList)).Get("/", h.handleList)
		r.With(authzMW(domain.SubjectAgent, domain.ActionCreate)).Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(UUIDMiddleware)
			r.With(authzMW(domain.SubjectAgent, domain.ActionRead)).Get("/{id}", h.handleGet)
			r.With(authzMW(domain.SubjectAgent, domain.ActionUpdate)).Patch("/{id}", h.handleUpdate)
			r.With(authzMW(domain.SubjectAgent, domain.ActionDelete)).Delete("/{id}", h.handleDelete)
		})
		r.With(authzMW(domain.SubjectAgent, domain.ActionUpdateState), AgentAuthMiddleware).
			Put("/me/status", h.handleUpdateStatusMe)
		r.With(authzMW(domain.SubjectAgent, domain.ActionRead), AgentAuthMiddleware).
			Get("/me", h.handleGetMe)
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
	render.JSON(w, r, agentToResponse(agent))
	render.Status(r, http.StatusCreated)
}

func (h *AgentHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
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
	agentID := MustGetAgentID(r)
	agent, err := h.querier.FindByID(r.Context(), agentID)
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
	result, err := h.querier.List(r.Context(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, NewPageResponse(result, agentToResponse))
}

func (h *AgentHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
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
	agentID := MustGetAgentID(r)
	agent, err := h.commander.UpdateState(r.Context(), agentID, p.State)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
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

// AgentResponse represents the response body for agent operations
type AgentResponse struct {
	ID          domain.UUID        `json:"id"`
	Name        string             `json:"name"`
	State       domain.AgentState  `json:"state"`
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
