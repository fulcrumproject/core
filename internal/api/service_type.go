package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"fulcrumproject.org/core/internal/domain"
)

type ServiceTypeHandler struct {
	repo domain.ServiceTypeRepository
}

func NewServiceTypeHandler(repo domain.ServiceTypeRepository) *ServiceTypeHandler {
	return &ServiceTypeHandler{repo: repo}
}

func (h *ServiceTypeHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/agent-type/{agentTypeId}", h.ListByAgentType)
	r.Put("/{id}/resource-definitions", h.UpdateResourceDefinitions)

	return r
}

func (h *ServiceTypeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name                string      `json:"name"`
		ResourceDefinitions domain.JSON `json:"resourceDefinitions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serviceType, err := domain.NewServiceType(input.Name, input.ResourceDefinitions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(r.Context(), serviceType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(serviceType)
}

func (h *ServiceTypeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serviceType, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Service type not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(serviceType)
}

func (h *ServiceTypeHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	serviceType, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Service type not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	serviceType.Name = input.Name

	if err := h.repo.Update(r.Context(), serviceType); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(serviceType)
}

func (h *ServiceTypeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Service type not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ServiceTypeHandler) List(w http.ResponseWriter, r *http.Request) {
	serviceTypes, err := h.repo.List(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(serviceTypes)
}

func (h *ServiceTypeHandler) ListByAgentType(w http.ResponseWriter, r *http.Request) {
	agentTypeID, err := ParseUUID(chi.URLParam(r, "agentTypeId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serviceTypes, err := h.repo.FindByAgentType(r.Context(), agentTypeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(serviceTypes)
}

func (h *ServiceTypeHandler) UpdateResourceDefinitions(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var input struct {
		ResourceDefinitions domain.JSON `json:"resourceDefinitions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateResourceDefinitions(r.Context(), id, input.ResourceDefinitions); err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Service type not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
