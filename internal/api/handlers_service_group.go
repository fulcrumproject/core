package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ServiceGroupHandler struct {
	querier   domain.ServiceGroupQuerier
	commander *domain.ServiceGroupCommander
}

func NewServiceGroupHandler(
	querier domain.ServiceGroupQuerier,
	commander *domain.ServiceGroupCommander,
) *ServiceGroupHandler {
	return &ServiceGroupHandler{
		commander: commander,
		querier:   querier,
	}
}

// Routes returns the router with all service group routes registered
func (h *ServiceGroupHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.handleList)
		r.Post("/", h.handleCreate)
		r.Group(func(r chi.Router) {
			r.Use(UUIDMiddleware)
			r.Get("/{id}", h.handleGet)
			r.Patch("/{id}", h.handleUpdate)
			r.Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *ServiceGroupHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	sg, err := h.commander.Create(r.Context(), req.Name)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serviceGroupToResponse(sg))
}

func (h *ServiceGroupHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
	serviceGroup, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}
	render.JSON(w, r, serviceGroupToResponse(serviceGroup))
}

func (h *ServiceGroupHandler) handleList(w http.ResponseWriter, r *http.Request) {
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
	render.JSON(w, r, NewPageResponse(result, serviceGroupToResponse))
}

func (h *ServiceGroupHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
	var req struct {
		Name *string `json:"name"`
	}
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	sg, err := h.commander.Update(r.Context(), id, req.Name)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, serviceGroupToResponse(sg))
}

func (h *ServiceGroupHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := GetUUIDParam(r)
	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ServiceGroupResponse represents the response body for service group operations
type ServiceGroupResponse struct {
	ID        domain.UUID `json:"id"`
	Name      string      `json:"name"`
	CreatedAt JSONUTCTime `json:"createdAt"`
	UpdatedAt JSONUTCTime `json:"updatedAt"`
}

// serviceGroupToResponse converts a domain.ServiceGroup to a ServiceGroupResponse
func serviceGroupToResponse(sg *domain.ServiceGroup) *ServiceGroupResponse {
	return &ServiceGroupResponse{
		ID:        sg.ID,
		Name:      sg.Name,
		CreatedAt: JSONUTCTime(sg.CreatedAt),
		UpdatedAt: JSONUTCTime(sg.UpdatedAt),
	}
}
