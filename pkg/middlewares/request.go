package middlewares

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type contextKey string

const (
	uuidContextKey        = contextKey("uuid")
	decodedBodyContextKey = contextKey("decodedBody")
)

// ID extracts and validates the UUID from URL paths with /{id} format
func ID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idParam := chi.URLParam(r, "id")
		if idParam != "" {
			id, err := properties.ParseUUID(idParam)
			if err != nil {
				render.Render(w, r, response.ErrInvalidRequest(err))
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
func MustGetID(ctx context.Context) properties.UUID {
	id, ok := ctx.Value(uuidContextKey).(properties.UUID)
	if !ok {
		panic("UUID not found in request context")
	}
	return id
}

// DecodeBody is middleware that decodes the request body into a struct
// and stores it in the request context for later middlewares and handlers
func DecodeBody[T any]() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a new instance of the target type
			v := new(T)

			// Decode the request body into the target
			if err := render.Decode(r, v); err != nil {
				render.Render(w, r, response.ErrInvalidRequest(err))
				return
			}

			// Store the decoded body in the context
			ctx := context.WithValue(r.Context(), decodedBodyContextKey, v)

			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// MustGetBody retrieves and casts the decoded body to a specific type
func MustGetBody[T any](ctx context.Context) T {
	var zero T
	body := ctx.Value(decodedBodyContextKey)
	if body == nil {
		panic("no decoded body found in context")
	}

	// First try direct type assertion
	if typed, ok := body.(T); ok {
		return typed
	}

	// If that fails, try pointer dereferencing (DecodeBody stores *T)
	if ptr, ok := body.(*T); ok {
		return *ptr
	}

	panic(fmt.Sprintf("expected body of type %T or *%T, got %T", zero, zero, body))
}
