package api

import (
	"errors"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// CreateUpdateProviderRequest represents the request body for creating a provider
type CreateUpdateProviderRequest struct {
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
	CreatedAt   JSONUTCTime          `json:"createdAt"`
	UpdatedAt   JSONUTCTime          `json:"updatedAt"`
}

// provderToResponse converts a domain.Provider to a ProviderResponse
func provderToResponse(p *domain.Provider) *ProviderResponse {
	return &ProviderResponse{
		ID:          p.ID.String(),
		Name:        string(p.Name),
		State:       p.State,
		CountryCode: string(p.CountryCode),
		Attributes:  map[string][]string(p.Attributes),
		CreatedAt:   JSONUTCTime(p.CreatedAt),
		UpdatedAt:   JSONUTCTime(p.UpdatedAt),
	}
}

type ProviderHandler struct {
	repo      domain.ProviderRepository
	agentRepo domain.AgentRepository
}

func NewProviderHandler(repo domain.ProviderRepository, agentRepo domain.AgentRepository) *ProviderHandler {
	return &ProviderHandler{repo: repo, agentRepo: agentRepo}
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
	var req CreateUpdateProviderRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	provider := &domain.Provider{
		Name:        req.Name,
		State:       req.State,
		CountryCode: domain.CountryCode(req.CountryCode),
		Attributes:  domain.Attributes(req.Attributes),
	}

	if err := provider.Validate(); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if err := h.repo.Create(r.Context(), provider); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, provderToResponse(provider))
}

func (h *ProviderHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
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

	render.JSON(w, r, NewPageResponse(result, provderToResponse))
}

func (h *ProviderHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	var req CreateUpdateProviderRequest
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
	provider.Name = req.Name
	provider.State = req.State
	provider.CountryCode = domain.CountryCode(req.CountryCode)
	provider.Attributes = domain.Attributes(req.Attributes)

	if err := provider.Validate(); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	if err := h.repo.Save(r.Context(), provider); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.JSON(w, r, provderToResponse(provider))
}

func (h *ProviderHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	provider, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	numOfAgents, err := h.agentRepo.CountByProvider(r.Context(), provider.ID)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	if numOfAgents > 0 {
		render.Render(w, r, ErrInvalidRequest(errors.New("cannot delete provider with associated agents")))
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
