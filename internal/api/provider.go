package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// CreateProviderRequest represents the request body for creating a provider
type CreateProviderRequest struct {
	Name        string               `json:"name"`
	State       domain.ProviderState `json:"state"`
	CountryCode string               `json:"countryCode,omitempty"`
	Attributes  map[string][]string  `json:"attributes,omitempty"`
}

// UpdateProviderRequest represents the request body for updating a provider
type UpdateProviderRequest struct {
	Name        string               `json:"name"`
	State       domain.ProviderState `json:"state"`
	CountryCode string               `json:"countryCode,omitempty"`
	Attributes  map[string][]string  `json:"attributes,omitempty"`
}

// ProviderResponse represents the response body for provider operations
type ProviderResponse struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	State       domain.ProviderState `json:"state"`
	CountryCode string               `json:"countryCode,omitempty"`
	Attributes  map[string][]string  `json:"attributes,omitempty"`
	CreatedAt   string               `json:"createdAt"`
	UpdatedAt   string               `json:"updatedAt"`
}

// toResponse converts a domain.Provider to a ProviderResponse
func toResponse(p *domain.Provider) *ProviderResponse {
	return &ProviderResponse{
		ID:          uuid.UUID(p.ID).String(),
		Name:        string(p.Name),
		State:       p.State,
		CountryCode: string(p.CountryCode),
		Attributes:  map[string][]string(p.Attributes),
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type ProviderHandler struct {
	repo domain.ProviderRepository
}

func NewProviderHandler(repo domain.ProviderRepository) *ProviderHandler {
	return &ProviderHandler{repo: repo}
}

// Create handles the creation of a new provider
func (h *ProviderHandler) Create(ctx context.Context, req *CreateProviderRequest) (*ProviderResponse, error) {
	provider := &domain.Provider{
		Name:        domain.Name(req.Name),
		State:       req.State,
		CountryCode: domain.CountryCode(req.CountryCode),
		Attributes:  domain.Attributes(req.Attributes),
	}

	if err := provider.Validate(); err != nil {
		return nil, err
	}

	if err := h.repo.Create(ctx, provider); err != nil {
		return nil, err
	}

	return toResponse(provider), nil
}

// Update handles updating an existing provider
func (h *ProviderHandler) Update(ctx context.Context, id string, req *UpdateProviderRequest) (*ProviderResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid provider ID format")
	}

	existing, err := h.repo.FindByID(ctx, domain.UUID(uid))
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("provider not found")
	}

	existing.Name = domain.Name(req.Name)
	existing.State = req.State
	existing.CountryCode = domain.CountryCode(req.CountryCode)
	existing.Attributes = domain.Attributes(req.Attributes)

	if err := existing.Validate(); err != nil {
		return nil, err
	}

	if err := h.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return toResponse(existing), nil
}

// Get retrieves a provider by ID
func (h *ProviderHandler) Get(ctx context.Context, id string) (*ProviderResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.New("invalid provider ID format")
	}

	provider, err := h.repo.FindByID(ctx, domain.UUID(uid))
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, errors.New("provider not found")
	}

	return toResponse(provider), nil
}

// List retrieves all providers matching the given filters
func (h *ProviderHandler) List(ctx context.Context, filters map[string]interface{}) ([]*ProviderResponse, error) {
	providers, err := h.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	response := make([]*ProviderResponse, len(providers))
	for i, provider := range providers {
		provider := provider // Create a new variable to avoid pointer issues
		response[i] = toResponse(&provider)
	}

	return response, nil
}

// Delete removes a provider by ID
func (h *ProviderHandler) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid provider ID format")
	}

	return h.repo.Delete(ctx, domain.UUID(uid))
}

// Routes returns the router with all provider routes registered
func (h *ProviderHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var req CreateProviderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		resp, err := h.Create(r.Context(), &req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		resp, err := h.Get(r.Context(), id)
		if err != nil {
			if err.Error() == "provider not found" {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Parse query parameters for filters
		filters := make(map[string]interface{})
		resp, err := h.List(r.Context(), filters)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Put("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req UpdateProviderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		resp, err := h.Update(r.Context(), id, &req)
		if err != nil {
			if err.Error() == "provider not found" {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := h.Delete(r.Context(), id); err != nil {
			if err.Error() == "provider not found" {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	return r
}
