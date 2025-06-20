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
