package api

import (
	"net/http"
	"time"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type TokenHandler struct {
	querier   domain.TokenQuerier
	commander domain.TokenCommander
}

func NewTokenHandler(
	querier domain.TokenQuerier,
	commander domain.TokenCommander,
) *TokenHandler {
	return &TokenHandler{
		querier:   querier,
		commander: commander,
	}
}

// Routes returns the router with all token routes registered
func (h *TokenHandler) Routes(authzMW AuthzMiddlewareFunc) func(r chi.Router) {
	return func(r chi.Router) {
		r.With(authzMW(domain.SubjectToken, domain.ActionList)).Get("/", h.handleList)
		r.With(authzMW(domain.SubjectToken, domain.ActionCreate)).Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(UUIDMiddleware)
			r.With(authzMW(domain.SubjectToken, domain.ActionRead)).Get("/{id}", h.handleGet)
			r.With(authzMW(domain.SubjectToken, domain.ActionUpdate)).Patch("/{id}", h.handleUpdate)
			r.With(authzMW(domain.SubjectToken, domain.ActionDelete)).Delete("/{id}", h.handleDelete)
			r.With(authzMW(domain.SubjectToken, domain.ActionGenerateToken)).Post("/{id}/regenerate", h.handleRegenerateValue)
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
	token, err := h.commander.Create(r.Context(), req.Name, req.Role, req.ExpireAt, req.ScopeID)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, tokenToResponse(token))
}

func (h *TokenHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
	token, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, tokenToResponse(token))
}

func (h *TokenHandler) handleList(w http.ResponseWriter, r *http.Request) {
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
	render.JSON(w, r, NewPageResponse(result, tokenToResponse))
}

func (h *TokenHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
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
	id := MustGetUUIDParam(r)
	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TokenHandler) handleRegenerateValue(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
	token, err := h.commander.Regenerate(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, tokenToResponse(token))
}

// TokenResponse represents the response body for token operations
type TokenResponse struct {
	ID         domain.UUID     `json:"id"`
	Name       string          `json:"name"`
	Role       domain.AuthRole `json:"role"`
	ExpireAt   JSONUTCTime     `json:"expireAt"`
	ProviderID *domain.UUID    `json:"scopeId,omitempty"`
	AgentID    *domain.UUID    `json:"agentId,omitempty"`
	BrokerID   *domain.UUID    `json:"providerId,omitempty"`
	CreatedAt  JSONUTCTime     `json:"createdAt"`
	UpdatedAt  JSONUTCTime     `json:"updatedAt"`
	Value      string          `json:"value,omitempty"`
}

// tokenToResponse converts a domain.Token to a TokenResponse
func tokenToResponse(t *domain.Token) *TokenResponse {
	return &TokenResponse{
		ID:         t.ID,
		Name:       t.Name,
		Role:       t.Role,
		ExpireAt:   JSONUTCTime(t.ExpireAt),
		ProviderID: t.ProviderID,
		AgentID:    t.AgentID,
		BrokerID:   t.BrokerID,
		CreatedAt:  JSONUTCTime(t.CreatedAt),
		UpdatedAt:  JSONUTCTime(t.UpdatedAt),
		Value:      t.PlainValue,
	}
}
