package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"fulcrumproject.org/core/internal/domain"
)

type ProviderHandler struct {
	repo domain.ProviderRepository
}

func NewProviderHandler(repo domain.ProviderRepository) *ProviderHandler {
	return &ProviderHandler{repo: repo}
}

func (h *ProviderHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/country/{code}", h.ListByCountry)
	r.Put("/{id}/state", h.UpdateState)

	return r
}

func (h *ProviderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string            `json:"name"`
		CountryCode string            `json:"countryCode"`
		Attributes  domain.Attributes `json:"attributes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	provider, err := domain.NewProvider(input.Name, input.CountryCode, input.Attributes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(r.Context(), provider); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(provider)
}

func (h *ProviderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	provider, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(provider)
}

func (h *ProviderHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var input struct {
		Name        string            `json:"name"`
		CountryCode string            `json:"countryCode"`
		Attributes  domain.Attributes `json:"attributes,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	provider, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	provider.Name = input.Name
	provider.CountryCode = input.CountryCode
	if input.Attributes != nil {
		attrs, err := input.Attributes.ToGormAttributes()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		provider.Attributes = attrs
	}

	if err := h.repo.Update(r.Context(), provider); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(provider)
}

func (h *ProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProviderHandler) List(w http.ResponseWriter, r *http.Request) {
	providers, err := h.repo.List(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(providers)
}

func (h *ProviderHandler) ListByCountry(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	providers, err := h.repo.FindByCountryCode(r.Context(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(providers)
}

func (h *ProviderHandler) UpdateState(w http.ResponseWriter, r *http.Request) {
	id, err := ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var input struct {
		State domain.ProviderState `json:"state"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateState(r.Context(), id, input.State); err != nil {
		if err == domain.ErrNotFound {
			http.Error(w, "Provider not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
