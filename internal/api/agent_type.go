package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"fulcrumproject.org/core/internal/domain"
)

type AgentTypeHandler struct {
	repo domain.AgentTypeRepository
}

func NewAgentTypeHandler(repo domain.AgentTypeRepository) *AgentTypeHandler {
	return &AgentTypeHandler{repo: repo}
}

func (h *AgentTypeHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/service-type/{serviceTypeId}", h.ListByServiceType)
	r.Post("/{id}/service-types/{serviceTypeId}", h.AddServiceType)
	r.Delete("/{id}/service-types/{serviceTypeId}", h.RemoveServiceType)

	return r
}

func (h *AgentTypeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agentType, err := domain.NewAgentType(input.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(r.Context(), agentType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agentType)
}

func (h *AgentTypeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agentType, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Agent type not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agentType)
}

func (h *AgentTypeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var input struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agentType, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Agent type not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	agentType.Name = input.Name

	if err := h.repo.Update(r.Context(), agentType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agentType)
}

func (h *AgentTypeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Agent type not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AgentTypeHandler) List(w http.ResponseWriter, r *http.Request) {
	agentTypes, err := h.repo.List(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agentTypes)
}

func (h *AgentTypeHandler) ListByServiceType(w http.ResponseWriter, r *http.Request) {
	serviceTypeID, err := ParseUUID(chi.URLParam(r, "serviceTypeId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agentTypes, err := h.repo.FindByServiceType(r.Context(), serviceTypeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(agentTypes)
}

func (h *AgentTypeHandler) AddServiceType(w http.ResponseWriter, r *http.Request) {
	agentTypeID, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serviceTypeID, err := ParseUUID(chi.URLParam(r, "serviceTypeId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.AddServiceType(r.Context(), agentTypeID, serviceTypeID); err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Agent type or service type not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *AgentTypeHandler) RemoveServiceType(w http.ResponseWriter, r *http.Request) {
	agentTypeID, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serviceTypeID, err := ParseUUID(chi.URLParam(r, "serviceTypeId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.RemoveServiceType(r.Context(), agentTypeID, serviceTypeID); err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Agent type or service type not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
