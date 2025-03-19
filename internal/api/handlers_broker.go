package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type BrokerHandler struct {
	querier   domain.BrokerQuerier
	commander domain.BrokerCommander
}

func NewBrokerHandler(
	querier domain.BrokerQuerier,
	commander domain.BrokerCommander,
) *BrokerHandler {
	return &BrokerHandler{
		querier:   querier,
		commander: commander,
	}
}

// Routes returns the router with all broker routes registered
func (h *BrokerHandler) Routes(authzMW AuthzMiddlewareFunc) func(r chi.Router) {
	return func(r chi.Router) {
		r.With(authzMW(domain.SubjectBroker, domain.ActionList)).Get("/", h.handleList)
		r.With(authzMW(domain.SubjectBroker, domain.ActionCreate)).Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(UUIDMiddleware)
			r.With(authzMW(domain.SubjectBroker, domain.ActionRead)).Get("/{id}", h.handleGet)
			r.With(authzMW(domain.SubjectBroker, domain.ActionUpdate)).Patch("/{id}", h.handleUpdate)
			r.With(authzMW(domain.SubjectBroker, domain.ActionDelete)).Delete("/{id}", h.handleDelete)
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
	broker, err := h.commander.Create(r.Context(), req.Name)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, brokerToResponse(broker))
}

func (h *BrokerHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
	broker, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, brokerToResponse(broker))
}

func (h *BrokerHandler) handleList(w http.ResponseWriter, r *http.Request) {
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	result, err := h.querier.List(r.Context(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, NewPageResponse(result, brokerToResponse))
}

func (h *BrokerHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetUUIDParam(r)
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
	id := MustGetUUIDParam(r)
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
