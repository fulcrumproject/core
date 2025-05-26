package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateServiceGroupRequest struct {
	Name       string      `json:"name"`
	ConsumerID domain.UUID `json:"consumerId"`
}

func (r CreateServiceGroupRequest) AuthTargetScope() (*domain.AuthTargetScope, error) {
	return &domain.AuthTargetScope{ConsumerID: &r.ConsumerID}, nil
}

type UpdateServiceGroupRequest struct {
	Name *string `json:"name"`
}

type ServiceGroupHandler struct {
	querier   domain.ServiceGroupQuerier
	commander domain.ServiceGroupCommander
	authz     domain.Authorizer
}

func NewServiceGroupHandler(
	querier domain.ServiceGroupQuerier,
	commander domain.ServiceGroupCommander,
	authz domain.Authorizer,
) *ServiceGroupHandler {
	return &ServiceGroupHandler{
		commander: commander,
		querier:   querier,
		authz:     authz,
	}
}

// Routes returns the router with all service group routes registered
func (h *ServiceGroupHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			AuthzSimple(domain.SubjectServiceGroup, domain.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create endpoint - decode body, then authorize with consumer ID
		r.With(
			DecodeBody[CreateServiceGroupRequest](),
			AuthzFromBody[CreateServiceGroupRequest](domain.SubjectServiceGroup, domain.ActionCreate, h.authz),
		).Post("/", h.handleCreate)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(ID)

			// Get endpoint - authorize using service group's scope
			r.With(
				AuthzFromID(domain.SubjectServiceGroup, domain.ActionRead, h.authz, h.querier),
			).Get("/{id}", h.handleGet)

			// Update endpoint - decode body, authorize using service group's scope
			r.With(
				DecodeBody[UpdateServiceGroupRequest](),
				AuthzFromID(domain.SubjectServiceGroup, domain.ActionUpdate, h.authz, h.querier),
			).Patch("/{id}", h.handleUpdate)

			// Delete endpoint - authorize using service group's scope
			r.With(
				AuthzFromID(domain.SubjectServiceGroup, domain.ActionDelete, h.authz, h.querier),
			).Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *ServiceGroupHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	req := MustGetBody[CreateServiceGroupRequest](r.Context())

	sg, err := h.commander.Create(r.Context(), req.Name, req.ConsumerID)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serviceGroupToResponse(sg))
}

func (h *ServiceGroupHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	serviceGroup, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, serviceGroupToResponse(serviceGroup))
}

func (h *ServiceGroupHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
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

	render.JSON(w, r, NewPageResponse(result, serviceGroupToResponse))
}

func (h *ServiceGroupHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())
	req := MustGetBody[UpdateServiceGroupRequest](r.Context())

	sg, err := h.commander.Update(r.Context(), id, req.Name)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, serviceGroupToResponse(sg))
}

func (h *ServiceGroupHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

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
