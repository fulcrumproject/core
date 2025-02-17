package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
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
	AgentType   *AgentTypeListResponse `json:"agentType,omitempty"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
}

// agentToResponse converts a domain.Agent to an AgentResponse
func agentToResponse(a *domain.Agent) *AgentResponse {
	response := &AgentResponse{
		ID:          uuid.UUID(a.ID).String(),
		Name:        a.Name,
		State:       a.State,
		CountryCode: a.CountryCode,
		Attributes:  map[string][]string(a.Attributes),
		Properties:  a.Properties,
		ProviderID:  uuid.UUID(a.ProviderID).String(),
		AgentTypeID: uuid.UUID(a.AgentTypeID).String(),
		CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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

	return r
}

func (h *AgentHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateUpdateAgentRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	providerID, err := domain.ParseID(req.ProviderID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agentTypeID, err := domain.ParseID(req.AgentTypeID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agent := &domain.Agent{
		Name:        req.Name,
		State:       req.State,
		TokenHash:   req.TokenHash,
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

	if err := h.repo.Create(r.Context(), agent); err != nil {
		render.Render(w, r, ErrInternalServer(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
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
	filters := ParseFilters(r, []FilterConfig{
		{
			Param: "name",
		},
		{
			Param:  "state",
			Valuer: func(v string) interface{} { return domain.AgentState(v) },
		},
		{
			Param:  "countryCode",
			Query:  "country_code",
			Valuer: func(v string) interface{} { return v },
		},
		{
			Param: "providerId",
			Query: "provider_id",
			Valuer: func(v string) interface{} {
				id, err := domain.ParseID(v)
				if err != nil {
					return nil
				}
				return id
			},
		},
		{
			Param: "agentTypeId",
			Query: "agent_type_id",
			Valuer: func(v string) interface{} {
				id, err := domain.ParseID(v)
				if err != nil {
					return nil
				}
				return id
			},
		},
	})
	sorting := ParseSorting(r)
	pagination := ParsePagination(r)

	result, err := h.repo.List(r.Context(), filters, sorting, pagination)
	if err != nil {
		render.Render(w, r, ErrInternalServer(err))
		return
	}

	response := make([]*AgentResponse, len(result.Items))
	for i, agent := range result.Items {
		response[i] = agentToResponse(&agent)
	}

	render.JSON(w, r, &PaginatedResponse[*AgentResponse]{
		Items:       response,
		TotalItems:  result.TotalItems,
		TotalPages:  result.TotalPages,
		CurrentPage: result.CurrentPage,
		HasNext:     result.HasNext,
		HasPrev:     result.HasPrev,
	})
}

func (h *AgentHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
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

	providerID, err := domain.ParseID(req.ProviderID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agentTypeID, err := domain.ParseID(req.AgentTypeID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Update fields
	agent.Name = req.Name
	agent.State = req.State
	agent.TokenHash = req.TokenHash
	agent.CountryCode = req.CountryCode
	agent.Attributes = domain.Attributes(req.Attributes)
	agent.Properties = req.Properties
	agent.ProviderID = providerID
	agent.AgentTypeID = agentTypeID

	if !agent.State.IsValid() {
		render.Render(w, r, ErrInvalidRequest(domain.ErrInvalidAgentState))
		return
	}

	if err := h.repo.Save(r.Context(), agent); err != nil {
		render.Render(w, r, ErrInternalServer(err))
		return
	}

	render.JSON(w, r, agentToResponse(agent))
}

func (h *AgentHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
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
		render.Render(w, r, ErrInternalServer(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
