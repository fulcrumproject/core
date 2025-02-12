package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"fulcrumproject.org/core/internal/domain"
)

type AgentHandler struct {
	repo domain.AgentRepository
}

func NewAgentHandler(repo domain.AgentRepository) *AgentHandler {
	return &AgentHandler{repo: repo}
}

func (h *AgentHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/provider/{providerId}", h.ListByProvider)
	r.Get("/type/{agentTypeId}", h.ListByAgentType)
	r.Put("/{id}/state", h.UpdateState)

	return r
}

func (h *AgentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string            `json:"name"`
		CountryCode string            `json:"countryCode"`
		Attributes  domain.Attributes `json:"attributes,omitempty"`
		Properties  domain.JSON       `json:"properties,omitempty"`
		ProviderID  domain.UUID       `json:"providerId"`
		AgentTypeID domain.UUID       `json:"agentTypeId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agent, err := domain.NewAgent(
		input.Name,
		input.CountryCode,
		input.Attributes,
		input.Properties,
		input.ProviderID,
		input.AgentTypeID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(r.Context(), agent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agent)
}

func (h *AgentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agent, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agent)
}

func (h *AgentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var input struct {
		Name        string            `json:"name"`
		CountryCode string            `json:"countryCode"`
		Attributes  domain.Attributes `json:"attributes,omitempty"`
		Properties  domain.JSON       `json:"properties,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agent, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	agent.Name = input.Name
	agent.CountryCode = input.CountryCode
	if input.Attributes != nil {
		attrs, err := input.Attributes.ToGormAttributes()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		agent.Attributes = attrs
	}
	if input.Properties != nil {
		props, err := input.Properties.ToGormJSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		agent.Properties = props
	}

	if err := h.repo.Update(r.Context(), agent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agent)
}

func (h *AgentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AgentHandler) List(w http.ResponseWriter, r *http.Request) {
	agents, err := h.repo.List(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agents)
}

func (h *AgentHandler) ListByProvider(w http.ResponseWriter, r *http.Request) {
	providerID, err := ParseUUID(chi.URLParam(r, "providerId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agents, err := h.repo.FindByProvider(r.Context(), providerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agents)
}

func (h *AgentHandler) ListByAgentType(w http.ResponseWriter, r *http.Request) {
	agentTypeID, err := ParseUUID(chi.URLParam(r, "agentTypeId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agents, err := h.repo.FindByAgentType(r.Context(), agentTypeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agents)
}

func (h *AgentHandler) UpdateState(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var input struct {
		State domain.AgentState `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateState(r.Context(), id, input.State); err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Agent not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
