package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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

// provderToResponse converts a domain.Provider to a ProviderResponse
func provderToResponse(p *domain.Provider) *ProviderResponse {
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

// Routes returns the router with all provider routes registered
func (h *ProviderHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware for all provider routes
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", h.handleList)
	r.Post("/", h.handleCreate)
	r.Get("/{id}", h.handleGet)
	r.Put("/{id}", h.handleUpdate)
	r.Delete("/{id}", h.handleDelete)

	return r
}

func (h *ProviderHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateProviderRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	provider := &domain.Provider{
		Name:        domain.Name(req.Name),
		State:       req.State,
		CountryCode: domain.CountryCode(req.CountryCode),
		Attributes:  domain.Attributes(req.Attributes),
	}

	if err := provider.Validate(); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if err := h.repo.Create(r.Context(), provider); err != nil {
		render.Render(w, r, ErrInternalServer(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, provderToResponse(provider))
}

func (h *ProviderHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	provider, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, provderToResponse(provider))
}

func (h *ProviderHandler) handleList(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters for filtering
	filters := make(map[string]interface{})
	if state := r.URL.Query().Get("state"); state != "" {
		filters["state"] = domain.ProviderState(state)
	}
	if countryCode := r.URL.Query().Get("countryCode"); countryCode != "" {
		filters["country_code"] = domain.CountryCode(countryCode)
	}

	providers, err := h.repo.List(r.Context(), filters)
	if err != nil {
		render.Render(w, r, ErrInternalServer(err))
		return
	}

	response := make([]*ProviderResponse, len(providers))
	for i, provider := range providers {
		response[i] = provderToResponse(&provider)
	}

	render.JSON(w, r, response)
}

func (h *ProviderHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	var req UpdateProviderRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	provider, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	// Update fields
	provider.Name = domain.Name(req.Name)
	provider.State = req.State
	provider.CountryCode = domain.CountryCode(req.CountryCode)
	provider.Attributes = domain.Attributes(req.Attributes)

	if err := provider.Validate(); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if err := h.repo.Save(r.Context(), provider); err != nil {
		render.Render(w, r, ErrInternalServer(err))
		return
	}

	render.JSON(w, r, provderToResponse(provider))
}

func (h *ProviderHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
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
