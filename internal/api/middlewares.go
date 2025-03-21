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
	uuidContextKey = contextKey("uuid")
)

// AuthMiddleware adds the identity to the context retrieving it from the authenticator
func AuthMiddleware(auth domain.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractTokenFromRequest(r)
			if token == "" {
				render.Render(w, r, ErrUnauthenticated())
				return
			}
			id := auth.Authenticate(r.Context(), token)
			if id == nil {
				render.Render(w, r, ErrUnauthenticated())
				return
			}
			ctx := domain.WithAuthIdentity(r.Context(), id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
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

// IDMiddleware extracts and validates the UUID from URL paths with /{id} format
func IDMiddleware(next http.Handler) http.Handler {
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

// MustGetID retrieves the UUID from the request context
func MustGetID(r *http.Request) domain.UUID {
	id, ok := r.Context().Value(uuidContextKey).(domain.UUID)
	if !ok {
		panic("UUID not found in request context")
	}
	return id
}
