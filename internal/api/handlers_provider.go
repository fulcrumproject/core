package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ProviderHandler struct {
	querier   domain.ProviderQuerier
	commander *domain.ProviderCommander
}

func NewProviderHandler(
	querier domain.ProviderQuerier,
	service *domain.ProviderCommander,
) *ProviderHandler {
	return &ProviderHandler{
		querier:   querier,
		commander: service,
	}
}

// Routes returns the router with all provider routes registered
func (h *ProviderHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(UUIDMiddleware)
			r.Get("/{id}", h.handleGet)
			r.Patch("/{id}", h.handleUpdate)
			r.Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *ProviderHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string               `json:"name"`
		State       domain.ProviderState `json:"state"`
		CountryCode domain.CountryCode   `json:"countryCode,omitempty"`
		Attributes  domain.Attributes    `json:"attributes,omitempty"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	provider, err := h.commander.Create(r.Context(), req.Name, req.State, req.CountryCode, req.Attributes)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, provderToResponse(provider))
}

func (h *ProviderHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
	provider, err := h.querier.FindByID(r.Context(), id)
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
	result, err := h.querier.List(r.Context(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, NewPageResponse(result, provderToResponse))
}

func (h *ProviderHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
	var req struct {
		Name        *string               `json:"name"`
		State       *domain.ProviderState `json:"state"`
		CountryCode *domain.CountryCode   `json:"countryCode,omitempty"`
		Attributes  *domain.Attributes    `json:"attributes,omitempty"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	provider, err := h.commander.Update(r.Context(), id, req.Name, req.State, req.CountryCode, req.Attributes)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, provderToResponse(provider))
}

func (h *ProviderHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ProviderResponse represents the response body for provider operations
type ProviderResponse struct {
	ID          domain.UUID          `json:"id"`
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
		ID:          p.ID,
		Name:        string(p.Name),
		State:       p.State,
		CountryCode: string(p.CountryCode),
		Attributes:  map[string][]string(p.Attributes),
		CreatedAt:   JSONUTCTime(p.CreatedAt),
		UpdatedAt:   JSONUTCTime(p.UpdatedAt),
	}
}
