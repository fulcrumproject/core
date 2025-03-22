package api

import (
	"context"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
)

// Helper to add ID param to context
func addIDToChiContext(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// simulateIDMiddleware simulates the IDMiddleware by adding the UUID to the context
func simulateIDMiddleware(r *http.Request, idStr string) *http.Request {
	// Parse the ID string to a domain.UUID
	id, err := domain.ParseUUID(idStr)
	if err != nil {
		panic(err)
	}

	// Add the UUID to the context using the same key as in the actual middleware
	ctx := context.WithValue(r.Context(), contextKey("uuid"), id)
	return r.WithContext(ctx)
}
