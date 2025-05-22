package api

import (
	"context"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ParticipantHandler struct {
	querier   domain.ParticipantQuerier
	commander domain.ParticipantCommander
	authz     domain.Authorizer
}

func NewParticipantHandler(
	querier domain.ParticipantQuerier,
	commander domain.ParticipantCommander,
	authz domain.Authorizer,
) *ParticipantHandler {
	return &ParticipantHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all participant routes registered
func (h *ParticipantHandler) Routes() func(r chi.Router) {

	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(IDMiddleware)
			r.Get("/{id}", h.handleGet)
			r.Patch("/{id}", h.handleUpdate)
			r.Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *ParticipantHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	if err := h.authz.AuthorizeCtx(r.Context(), domain.SubjectParticipant, domain.ActionCreate, &domain.EmptyAuthScope); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	var req struct {
		Name        string                  `json:"name"`
		State       domain.ParticipantState `json:"state"`
		CountryCode domain.CountryCode      `json:"countryCode,omitempty"`
		Attributes  domain.Attributes       `json:"attributes,omitempty"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	participant, err := h.commander.Create(r.Context(), req.Name, req.State, req.CountryCode, req.Attributes)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, participantToResponse(participant))
}

func (h *ParticipantHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionRead)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	participant, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, participantToResponse(participant))
}

func (h *ParticipantHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
	if err := h.authz.Authorize(id, domain.SubjectParticipant, domain.ActionRead, &domain.EmptyAuthScope); err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	result, err := h.querier.List(r.Context(), id.Scope(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, NewPageResponse(result, participantToResponse))
}

func (h *ParticipantHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionUpdate)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	var req struct {
		Name        *string                  `json:"name"`
		State       *domain.ParticipantState `json:"state"`
		CountryCode *domain.CountryCode      `json:"countryCode,omitempty"`
		Attributes  *domain.Attributes       `json:"attributes,omitempty"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	participant, err := h.commander.Update(r.Context(), id, req.Name, req.State, req.CountryCode, req.Attributes)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, participantToResponse(participant))
}

func (h *ParticipantHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionDelete)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ParticipantResponse represents the response body for participant operations
type ParticipantResponse struct {
	ID          domain.UUID             `json:"id"`
	Name        string                  `json:"name"`
	State       domain.ParticipantState `json:"state"`
	CountryCode string                  `json:"countryCode,omitempty"`
	Attributes  map[string][]string     `json:"attributes,omitempty"`
	CreatedAt   JSONUTCTime             `json:"createdAt"`
	UpdatedAt   JSONUTCTime             `json:"updatedAt"`
}

// participantToResponse converts a domain.Participant to a ParticipantResponse
func participantToResponse(p *domain.Participant) *ParticipantResponse {
	return &ParticipantResponse{
		ID:          p.ID,
		Name:        p.Name,
		State:       p.State,
		CountryCode: string(p.CountryCode),
		Attributes:  map[string][]string(p.Attributes),
		CreatedAt:   JSONUTCTime(p.CreatedAt),
		UpdatedAt:   JSONUTCTime(p.UpdatedAt),
	}
}

func (h *ParticipantHandler) authorize(ctx context.Context, id domain.UUID, action domain.AuthAction) (*domain.AuthScope, error) {
	scope, err := h.querier.AuthScope(ctx, id)
	if err != nil {
		return nil, err
	}
	err = h.authz.AuthorizeCtx(ctx, domain.SubjectParticipant, action, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}
