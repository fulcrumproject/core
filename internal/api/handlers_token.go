package api

import (
	"context"
	"net/http"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type TokenHandler struct {
	querier      domain.TokenQuerier
	commander    domain.TokenCommander
	agentQuerier domain.AgentQuerier
	authz        domain.Authorizer
}

func NewTokenHandler(
	querier domain.TokenQuerier,
	commander domain.TokenCommander,
	agentQuerier domain.AgentQuerier,
	authz domain.Authorizer,
) *TokenHandler {
	return &TokenHandler{
		querier:      querier,
		commander:    commander,
		agentQuerier: agentQuerier,
		authz:        authz,
	}
}

// Routes returns the router with all token routes registered
func (h *TokenHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(IDMiddleware)
			r.Get("/{id}", h.handleGet)
			r.Patch("/{id}", h.handleUpdate)
			r.Delete("/{id}", h.handleDelete)
			r.Post("/{id}/regenerate", h.handleRegenerateValue)
		})
	}
}

func (h *TokenHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string          `json:"name"`
		Role     domain.AuthRole `json:"role"`
		ExpireAt time.Time       `json:"expireAt"`
		ScopeID  *domain.UUID    `json:"scopeId,omitempty"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	var scope domain.AuthScope
	if req.ScopeID != nil {
		switch req.Role {
		case domain.RoleParticipant:
			scope.ParticipantID = req.ScopeID
		case domain.RoleAgent:
			scope.AgentID = req.ScopeID
		}
	}
	if err := h.authz.AuthorizeCtx(r.Context(), domain.SubjectToken, domain.ActionCreate, &scope); err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	token, err := h.commander.Create(r.Context(), req.Name, req.Role, req.ExpireAt, req.ScopeID)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, tokenToResponse(token))
}

func (h *TokenHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionRead)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	token, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, tokenToResponse(token))
}

func (h *TokenHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
	if err := h.authz.Authorize(id, domain.SubjectToken, domain.ActionRead, &domain.EmptyAuthScope); err != nil {
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
	render.JSON(w, r, NewPageResponse(result, tokenToResponse))
}

func (h *TokenHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionUpdate)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	var req struct {
		Name     *string    `json:"name"`
		ExpireAt *time.Time `json:"expireAt"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	token, err := h.commander.Update(r.Context(), id, req.Name, req.ExpireAt)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, tokenToResponse(token))
}

func (h *TokenHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
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

func (h *TokenHandler) handleRegenerateValue(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionGenerateToken)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	token, err := h.commander.Regenerate(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, tokenToResponse(token))
}

// TokenResponse represents the response body for token operations
type TokenResponse struct {
	ID            domain.UUID     `json:"id"`
	Name          string          `json:"name"`
	Role          domain.AuthRole `json:"role"`
	ExpireAt      JSONUTCTime     `json:"expireAt"`
	ParticipantID *domain.UUID    `json:"scopeId,omitempty"`
	AgentID       *domain.UUID    `json:"agentId,omitempty"`
	CreatedAt     JSONUTCTime     `json:"createdAt"`
	UpdatedAt     JSONUTCTime     `json:"updatedAt"`
	Value         string          `json:"value,omitempty"`
}

// tokenToResponse converts a domain.Token to a TokenResponse
func tokenToResponse(t *domain.Token) *TokenResponse {
	return &TokenResponse{
		ID:            t.ID,
		Name:          t.Name,
		Role:          t.Role,
		ExpireAt:      JSONUTCTime(t.ExpireAt),
		ParticipantID: t.ParticipantID,
		AgentID:       t.AgentID,
		CreatedAt:     JSONUTCTime(t.CreatedAt),
		UpdatedAt:     JSONUTCTime(t.UpdatedAt),
		Value:         t.PlainValue,
	}
}

func (h *TokenHandler) authorize(ctx context.Context, id domain.UUID, action domain.AuthAction) (*domain.AuthScope, error) {
	scope, err := h.querier.AuthScope(ctx, id)
	if err != nil {
		return nil, err
	}
	err = h.authz.AuthorizeCtx(ctx, domain.SubjectToken, action, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}
