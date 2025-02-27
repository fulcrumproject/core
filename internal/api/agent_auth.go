package api

import (
	"context"
	"net/http"
	"strings"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/render"
)

// Context keys for authentication
type contextKey string

const agentContextKey = contextKey("agent")

// AgentAuthMiddleware authenticates requests using agent tokens
func AgentAuthMiddleware(repo domain.AgentRepository) func(http.Handler) http.Handler {
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
