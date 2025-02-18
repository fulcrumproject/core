package api

import (
	"errors"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// CreateUpdateServiceGroupRequest represents the request body for creating a service group
type CreateUpdateServiceGroupRequest struct {
	Name string `json:"name"`
}

// ServiceGroupResponse represents the response body for service group operations
type ServiceGroupResponse struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	CreatedAt JSONUTCTime `json:"createdAt"`
	UpdatedAt JSONUTCTime `json:"updatedAt"`
}

// serviceGroupToResponse converts a domain.ServiceGroup to a ServiceGroupResponse
func serviceGroupToResponse(sg *domain.ServiceGroup) *ServiceGroupResponse {
	return &ServiceGroupResponse{
		ID:        sg.ID.String(),
		Name:      sg.Name,
		CreatedAt: JSONUTCTime(sg.CreatedAt),
		UpdatedAt: JSONUTCTime(sg.UpdatedAt),
	}
}

type ServiceGroupHandler struct {
	repo        domain.ServiceGroupRepository
	serviceRepo domain.ServiceRepository
}

func NewServiceGroupHandler(repo domain.ServiceGroupRepository, serviceRepo domain.ServiceRepository) *ServiceGroupHandler {
	return &ServiceGroupHandler{repo: repo, serviceRepo: serviceRepo}
}

// Routes returns the router with all service group routes registered
func (h *ServiceGroupHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware for all service group routes
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", h.handleList)
	r.Post("/", h.handleCreate)
	r.Get("/{id}", h.handleGet)
	r.Put("/{id}", h.handleUpdate)
	r.Delete("/{id}", h.handleDelete)

	return r
}

func (h *ServiceGroupHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateUpdateServiceGroupRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	serviceGroup := &domain.ServiceGroup{
		Name: req.Name,
	}

	if err := h.repo.Create(r.Context(), serviceGroup); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serviceGroupToResponse(serviceGroup))
}

func (h *ServiceGroupHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	serviceGroup, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, serviceGroupToResponse(serviceGroup))
}

func (h *ServiceGroupHandler) handleList(w http.ResponseWriter, r *http.Request) {
	filter := parseSimpleFilter(r)
	sorting := parseSorting(r)
	pagination := parsePagination(r)

	result, err := h.repo.List(r.Context(), filter, sorting, pagination)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPaginatedResponse(result, serviceGroupToResponse))
}

func (h *ServiceGroupHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	var req CreateUpdateServiceGroupRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	serviceGroup, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	// Update fields
	serviceGroup.Name = req.Name

	if err := h.repo.Save(r.Context(), serviceGroup); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.JSON(w, r, serviceGroupToResponse(serviceGroup))
}

func (h *ServiceGroupHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	serviceGroup, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	numOfServices, err := h.serviceRepo.CountByGroup(r.Context(), serviceGroup.ID)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	if numOfServices > 0 {
		render.Render(w, r, ErrInvalidRequest(errors.New("cannot delete service group with associated services")))
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
