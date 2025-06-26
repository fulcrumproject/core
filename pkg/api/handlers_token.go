package api

import (
	"context"
	"net/http"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
)

// Request types

// CreateTokenReq represents a request to create a new token
type CreateTokenReq struct {
	Name     string           `json:"name"`
	Role     auth.Role        `json:"role"`
	ScopeID  *properties.UUID `json:"scopeId,omitempty"`
	AgentID  *properties.UUID `json:"agentId,omitempty"`
	ExpireAt *time.Time       `json:"expireAt,omitempty"` // Match the original field name in tests
}

// UpdateTokenReq represents a request to update a token
type UpdateTokenReq struct {
	Name     *string    `json:"name,omitempty"`
	ExpireAt *time.Time `json:"expireAt,omitempty"`
}

// CreateTokenScopeExtractor creates an extractor that sets the target scope based on token role
func CreateTokenScopeExtractor() middlewares.ObjectScopeExtractor {
	return func(r *http.Request) (auth.ObjectScope, error) {
		// Get decoded body from context
		body := middlewares.MustGetBody[CreateTokenReq](r.Context())

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
		).Get("/", List(h.querier, TokenToRes))

		// Create - using standard Create handler
		r.With(
			middlewares.DecodeBody[CreateTokenReq](),
			middlewares.AuthzFromExtractor(
				authz.ObjectTypeToken,
				authz.ActionCreate,
				h.authz,
				CreateTokenScopeExtractor(),
			),
		).Post("/", Create(h.Create, TokenToRes))

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeToken, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, TokenToRes))

			// Update - using standard Update handler
			r.With(
				middlewares.DecodeBody[UpdateTokenReq](),
				middlewares.AuthzFromID(authz.ObjectTypeToken, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, TokenToRes))

			// Delete - authorize from resource ID
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeToken, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))

			// Regenerate - using standard ActionWithoutBody handler
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeToken, authz.ActionGenerateToken, h.authz, h.querier.AuthScope),
			).Post("/{id}/regenerate", ActionWithoutBody(h.commander.Regenerate, TokenToRes))
		})
	}
}

// Adapter functions that convert request structs to commander method calls

func (h *TokenHandler) Create(ctx context.Context, req *CreateTokenReq) (*domain.Token, error) {
	return h.commander.Create(ctx, req.Name, req.Role, req.ExpireAt, req.ScopeID)
}

func (h *TokenHandler) Update(ctx context.Context, id properties.UUID, req *UpdateTokenReq) (*domain.Token, error) {
	return h.commander.Update(ctx, id, req.Name, req.ExpireAt)
}

// TokenRes represents the response body for token operations
type TokenRes struct {
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

// TokenToRes converts a domain.Token to a TokenResponse
func TokenToRes(t *domain.Token) *TokenRes {
	return &TokenRes{
		ID:            t.ID,
		Name:          t.Name,
		Role:          t.Role,
		ExpireAt:      JSONUTCTime(t.ExpireAt),
		ParticipantID: t.ParticipantID,
		AgentID:       t.AgentID,
		CreatedAt:     JSONUTCTime(t.CreatedAt),
		UpdatedAt:     JSONUTCTime(t.UpdatedAt),
		Value:         t.PlainValue, // Only populated on create/regenerate
	}
}
