package api

import (
	"net/http"
	"time"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/middlewares"
	"github.com/fulcrumproject/commons/properties"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// Request types

// CreateTokenRequest represents a request to create a new token
type CreateTokenRequest struct {
	Name     string           `json:"name"`
	Role     auth.Role        `json:"role"`
	ScopeID  *properties.UUID `json:"scopeId,omitempty"`
	AgentID  *properties.UUID `json:"agentId,omitempty"`
	ExpireAt *time.Time       `json:"expireAt,omitempty"` // Match the original field name in tests
}

// UpdateTokenRequest represents a request to update a token
type UpdateTokenRequest struct {
	Name     *string    `json:"name,omitempty"`
	ExpireAt *time.Time `json:"expireAt,omitempty"`
}

// CreateTokenScopeExtractor creates an extractor that sets the target scope based on token role
func CreateTokenScopeExtractor() middlewares.ObjectScopeExtractor {
	return func(r *http.Request) (auth.ObjectScope, error) {
		// Get decoded body from context
		body := middlewares.MustGetBody[CreateTokenRequest](r.Context())

		// Determine scope based on role
		scope := &auth.DefaultObjectScope{}

		switch body.Role {
		case auth.RoleParticipant:
			scope.ParticipantID = body.ScopeID
		case auth.RoleAgent:
			scope.AgentID = body.AgentID
		}

		return scope, nil
	}
}

type TokenHandler struct {
	querier      domain.TokenQuerier
	commander    domain.TokenCommander
	agentQuerier domain.AgentQuerier
	authz        auth.Authorizer
}

func NewTokenHandler(
	querier domain.TokenQuerier,
	commander domain.TokenCommander,
	agentQuerier domain.AgentQuerier,
	authz auth.Authorizer,
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
		// List - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeToken, authz.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create - decode body + specialized scope extractor for authorization
		r.With(
			middlewares.DecodeBody[CreateTokenRequest](),
			middlewares.AuthzFromExtractor(
				authz.ObjectTypeToken,
				authz.ActionCreate,
				h.authz,
				CreateTokenScopeExtractor(),
			),
		).Post("/", h.handleCreate)

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeToken, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", h.handleGet)

			// Update - decode body + authorize from resource ID
			r.With(
				middlewares.DecodeBody[UpdateTokenRequest](),
				middlewares.AuthzFromID(authz.ObjectTypeToken, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", h.handleUpdate)

			// Delete - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeToken, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", h.handleDelete)

			// Regenerate - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeToken, authz.ActionGenerateToken, h.authz, h.querier.AuthScope),
			).Post("/{id}/regenerate", h.handleRegenerateValue)
		})
	}
}

func (h *TokenHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	// Get decoded body from context
	req := middlewares.MustGetBody[CreateTokenRequest](r.Context())

	token, err := h.commander.Create(r.Context(), req.Name, req.Role, req.ExpireAt, req.ScopeID)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, tokenToResponse(token))
}

func (h *TokenHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	token, err := h.querier.Get(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, tokenToResponse(token))
}

func (h *TokenHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := auth.MustGetIdentity(r.Context())

	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	result, err := h.querier.List(r.Context(), &id.Scope, pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, NewPageResponse(result, tokenToResponse))
}

func (h *TokenHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())
	req := middlewares.MustGetBody[UpdateTokenRequest](r.Context())

	token, err := h.commander.Update(r.Context(), id, req.Name, req.ExpireAt)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, tokenToResponse(token))
}

func (h *TokenHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TokenHandler) handleRegenerateValue(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	token, err := h.commander.Regenerate(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, tokenToResponse(token))
}

// TokenResponse represents the response body for token operations
type TokenResponse struct {
	ID            properties.UUID  `json:"id"`
	Name          string           `json:"name"`
	Role          auth.Role        `json:"role"`
	ExpireAt      JSONUTCTime      `json:"expireAt"`
	ParticipantID *properties.UUID `json:"participantId,omitempty"`
	AgentID       *properties.UUID `json:"agentId,omitempty"`
	CreatedAt     JSONUTCTime      `json:"createdAt"`
	UpdatedAt     JSONUTCTime      `json:"updatedAt"`
	Value         string           `json:"value,omitempty"`
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
