package api

import (
	"context"
	"net/http"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/render"
)

// List handles standard list operations that take a querier and a toResp function
func List[T domain.Entity, R any](querier domain.BaseEntityQuerier[T], toResp func(*T) *R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := auth.MustGetIdentity(r.Context())
		pag, err := ParsePageRequest(r)
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		result, err := querier.List(r.Context(), &id.Scope, pag)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		render.JSON(w, r, NewPageResponse(result, toResp))
	}
}

// Get handles standard get operations that take an ID from URL and return an entity
func Get[T domain.Entity, R any](querier domain.BaseEntityQuerier[T], toResp func(*T) *R) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := middlewares.MustGetID(r.Context())

		entity, err := querier.Get(r.Context(), id)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		render.JSON(w, r, toResp(entity))
	}
}

// Delete handles standard delete operations that take an ID from URL and return a deleted entity
func Delete[T domain.Entity](querier domain.BaseEntityQuerier[T], deleteFunc func(context.Context, properties.UUID) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := middlewares.MustGetID(r.Context())

		found, err := querier.Exists(r.Context(), id)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}
		if !found {
			render.Render(w, r, ErrNotFound())
			return
		}

		if err := deleteFunc(r.Context(), id); err != nil {
			render.Render(w, r, ErrInternal(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// Create handles standard create operations that take a request body and return a created entity
func Create[Req any, T domain.Entity, R any](
	createFunc func(context.Context, *Req) (*T, error),
	toResp func(*T) *R,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := middlewares.MustGetBody[Req](r.Context())

		entity, err := createFunc(r.Context(), &req)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, toResp(entity))
	}
}

// Update handles standard update operations that take an ID from URL and request body
func Update[Req any, T domain.Entity, R any](
	updateFunc func(context.Context, properties.UUID, *Req) (*T, error),
	toResp func(*T) *R,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := middlewares.MustGetID(r.Context())
		req := middlewares.MustGetBody[Req](r.Context())

		entity, err := updateFunc(r.Context(), id, &req)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		render.JSON(w, r, toResp(entity))
	}
}

// Action handles operations that take an ID and optionally a request body, returning an entity
func Action[Req any, T domain.Entity, R any](
	actionFunc func(context.Context, properties.UUID, *Req) (*T, error),
	toResp func(*T) *R,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := middlewares.MustGetID(r.Context())
		req := middlewares.MustGetBody[Req](r.Context())

		entity, err := actionFunc(r.Context(), id, &req)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		render.JSON(w, r, toResp(entity))
	}
}

// ActionWithoutBody handles operations that only need an ID and return an entity
func ActionWithoutBody[T domain.Entity, R any](
	actionFunc func(context.Context, properties.UUID) (*T, error),
	toResp func(*T) *R,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := middlewares.MustGetID(r.Context())

		entity, err := actionFunc(r.Context(), id)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		render.JSON(w, r, toResp(entity))
	}
}

// Command handles operations that don't return entities (like status changes)
func Command[Req any](
	commandFunc func(context.Context, properties.UUID, *Req) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := middlewares.MustGetID(r.Context())
		req := middlewares.MustGetBody[Req](r.Context())

		if err := commandFunc(r.Context(), id, &req); err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// CommandWithoutBody handles operations that only need an ID and don't return anything
func CommandWithoutBody(
	commandFunc func(context.Context, properties.UUID) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := middlewares.MustGetID(r.Context())

		if err := commandFunc(r.Context(), id); err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// UpdateWithoutID handles operations that take only a request body (no ID from URL) and return an entity
// This is useful for "me" endpoints and similar patterns where the ID comes from auth context
func UpdateWithoutID[Req any, T domain.Entity, R any](
	updateFunc func(context.Context, *Req) (*T, error),
	toResp func(*T) *R,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := middlewares.MustGetBody[Req](r.Context())

		entity, err := updateFunc(r.Context(), &req)
		if err != nil {
			render.Render(w, r, ErrDomain(err))
			return
		}

		render.JSON(w, r, toResp(entity))
	}
}
