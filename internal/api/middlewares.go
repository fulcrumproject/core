package api

import (
	"context"
	"net/http"
	"strings"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// Context keys for authentication
type contextKey string

const (
	agentContextKey = contextKey("agent")
	uuidContextKey  = contextKey("uuid")
)

// AgentAuthMiddleware authenticates requests using agent tokens
func AgentAuthMiddleware(repo domain.AgentQuerier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				render.Render(w, r, ErrUnauthorized())
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			tokenHash := domain.HashToken(token)

			// Find agent by token hash
			agent, err := repo.FindByTokenHash(r.Context(), tokenHash)
			if err != nil {
				render.Render(w, r, ErrUnauthorized())
				return
			}

			// Store authenticated agent in request context
			ctx := context.WithValue(r.Context(), agentContextKey, agent)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAuthenticatedAgent retrieves the authenticated agent from the request context
func GetAuthenticatedAgent(r *http.Request) *domain.Agent {
	agent, _ := r.Context().Value(agentContextKey).(*domain.Agent)
	return agent
}

// UUIDMiddleware extracts and validates the UUID from URL paths with /{id} format
func UUIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract id parameter from URL
		idParam := chi.URLParam(r, "id")
		if idParam != "" {
			// Parse and validate UUID
			id, err := domain.ParseUUID(idParam)
			if err != nil {
				render.Render(w, r, ErrInvalidRequest(err))
				return
			}

			// Store UUID in request context
			ctx := context.WithValue(r.Context(), uuidContextKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// If no id parameter, just continue
		next.ServeHTTP(w, r)
	})
}

// GetUUIDParam retrieves the UUID from the request context
// Panics if UUID is not found in context
func GetUUIDParam(r *http.Request) domain.UUID {
	id, ok := r.Context().Value(uuidContextKey).(domain.UUID)
	if !ok {
		panic("UUID not found in request context")
	}
	return id
}
