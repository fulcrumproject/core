package api

import (
	"context"
	"net/http"
	"strings"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type contextKey string

const (
	uuidContextKey    = contextKey("uuid")
	agentIDContextKey = contextKey("agentID")
)

type AuthzMiddlewareFunc func(subject domain.AuthSubject, action domain.AuthAction) func(http.Handler) http.Handler

// AuthzMiddleware authenticates and authorizes the request with the given subject and action and set the authorization scope
func AuthzMiddleware(auth domain.Authenticator, authz domain.Authorizer) AuthzMiddlewareFunc {
	return func(subject domain.AuthSubject, action domain.AuthAction) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token := extractTokenFromRequest(r)
				id := auth.Authenticate(r.Context(), token)
				if id == nil {
					render.Render(w, r, ErrUnauthorized())
					return
				}

				err := authz.Authorize(r.Context(), id, subject, action)
				if err != nil {
					render.Render(w, r, ErrDomain(err))
					return
				}

				ctx := domain.WithAuthIdentity(r.Context(), id)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}
	}
}

// extractTokenFromRequest gets the token from authorization header only
func extractTokenFromRequest(r *http.Request) string {
	// Get token from Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// No valid Authorization header found with Bearer token
	return ""

}

// UUIDMiddleware extracts and validates the UUID from URL paths with /{id} format
func UUIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idParam := chi.URLParam(r, "id")
		if idParam != "" {
			id, err := domain.ParseUUID(idParam)
			if err != nil {
				render.Render(w, r, ErrInvalidRequest(err))
				return
			}

			ctx := context.WithValue(r.Context(), uuidContextKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// MustGetUUIDParam retrieves the UUID from the request context
func MustGetUUIDParam(r *http.Request) domain.UUID {
	id, ok := r.Context().Value(uuidContextKey).(domain.UUID)
	if !ok {
		panic("UUID not found in request context")
	}
	return id
}

// AgentAuthMiddleware ensures that the request is from an authenticated agent with a valid scope ID
// and stores the agent ID in the request context for later retrieval
func AgentAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := domain.GetAuthIdentity(r.Context())
		if id == nil || !id.IsRole(domain.RoleAgent) || id.Scope().AgentID == nil {
			render.Render(w, r, ErrUnauthorized())
			return
		}

		// Store the agent ID in the context for easy retrieval
		ctx := context.WithValue(r.Context(), agentIDContextKey, *id.Scope().AgentID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// MustGetAgentID retrieves the agent ID from the request context
// This should be called only after the AgentAuthMiddleware has been applied
func MustGetAgentID(r *http.Request) domain.UUID {
	agentID, ok := r.Context().Value(agentIDContextKey).(domain.UUID)
	if !ok {
		panic("Agent ID not found in request context. Make sure AgentAuthMiddleware is applied.")
	}
	return agentID
}
