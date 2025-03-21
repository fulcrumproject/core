package api

import (
	"context"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type BrokerHandler struct {
	querier   domain.BrokerQuerier
	commander domain.BrokerCommander
	authz     domain.Authorizer
}

func NewBrokerHandler(
	querier domain.BrokerQuerier,
	commander domain.BrokerCommander,
	authz domain.Authorizer,
) *BrokerHandler {
	return &BrokerHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all broker routes registered
func (h *BrokerHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(IDMiddleware)
			r.Get("/{id}", h.handleGet)
			r.Patch("/{id}", h.handleUpdate)
			r.Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *BrokerHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	if err := h.authz.AuthorizeCtx(r.Context(), domain.SubjectBroker, domain.ActionCreate, &domain.EmptyAuthScope); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	broker, err := h.commander.Create(r.Context(), req.Name)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, brokerToResponse(broker))
}

func (h *BrokerHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionRead)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	broker, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, brokerToResponse(broker))
}

func (h *BrokerHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
	if err := h.authz.Authorize(id, domain.SubjectBroker, domain.ActionRead, &domain.EmptyAuthScope); err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	result, err := h.querier.List(r.Context(), id.Scope(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, NewPageResponse(result, brokerToResponse))
}

func (h *BrokerHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionUpdate)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	var req struct {
		Name *string `json:"name"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	broker, err := h.commander.Update(r.Context(), id, req.Name)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, brokerToResponse(broker))
}

func (h *BrokerHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r)
	_, err := h.authorize(r.Context(), id, domain.ActionDelete)
	if err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BrokerResponse represents the response body for broker operations
type BrokerResponse struct {
	ID        domain.UUID `json:"id"`
	Name      string      `json:"name"`
	CreatedAt JSONUTCTime `json:"createdAt"`
	UpdatedAt JSONUTCTime `json:"updatedAt"`
}

// brokerToResponse converts a domain.Broker to a BrokerResponse
func brokerToResponse(b *domain.Broker) *BrokerResponse {
	return &BrokerResponse{
		ID:        b.ID,
		Name:      b.Name,
		CreatedAt: JSONUTCTime(b.CreatedAt),
		UpdatedAt: JSONUTCTime(b.UpdatedAt),
	}
}

func (h *BrokerHandler) authorize(ctx context.Context, id domain.UUID, action domain.AuthAction) (*domain.AuthScope, error) {
	scope, err := h.querier.AuthScope(ctx, id)
	if err != nil {
		return nil, err
	}
	err = h.authz.AuthorizeCtx(ctx, domain.SubjectBroker, action, scope)
	if err != nil {
		return nil, err
	}
	return scope, nil
}
