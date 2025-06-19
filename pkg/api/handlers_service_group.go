package api

import (
	"net/http"

	"fulcrumproject.org/core/pkg/authz"
	"fulcrumproject.org/core/pkg/domain"
	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/middlewares"
	"github.com/fulcrumproject/commons/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateServiceGroupRequest struct {
	Name       string          `json:"name"`
	ConsumerID properties.UUID `json:"consumerId"`
}

func (r CreateServiceGroupRequest) ObjectScope() (auth.ObjectScope, error) {
	return &auth.DefaultObjectScope{ConsumerID: &r.ConsumerID}, nil
}

type UpdateServiceGroupRequest struct {
	Name *string `json:"name"`
}

type ServiceGroupHandler struct {
	querier   domain.ServiceGroupQuerier
	commander domain.ServiceGroupCommander
	authz     auth.Authorizer
}

func NewServiceGroupHandler(
	querier domain.ServiceGroupQuerier,
	commander domain.ServiceGroupCommander,
	authz auth.Authorizer,
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
			middlewares.AuthzSimple(authz.ObjectTypeServiceGroup, authz.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create endpoint - decode body, then authorize with consumer ID
		r.With(
			middlewares.DecodeBody[CreateServiceGroupRequest](),
			middlewares.AuthzFromBody[CreateServiceGroupRequest](authz.ObjectTypeServiceGroup, authz.ActionCreate, h.authz),
		).Post("/", h.handleCreate)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using service group's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceGroup, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", h.handleGet)

			// Update endpoint - decode body, authorize using service group's scope
			r.With(
				middlewares.DecodeBody[UpdateServiceGroupRequest](),
				middlewares.AuthzFromID(authz.ObjectTypeServiceGroup, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", h.handleUpdate)

			// Delete endpoint - authorize using service group's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeServiceGroup, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *ServiceGroupHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	req := middlewares.MustGetBody[CreateServiceGroupRequest](r.Context())

	sg, err := h.commander.Create(r.Context(), req.Name, req.ConsumerID)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serviceGroupToResponse(sg))
}

func (h *ServiceGroupHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	serviceGroup, err := h.querier.Get(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, serviceGroupToResponse(serviceGroup))
}

func (h *ServiceGroupHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := auth.MustGetIdentity(r.Context())
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	result, err := h.querier.List(r.Context(), &id.Scope, pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, serviceGroupToResponse))
}

func (h *ServiceGroupHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())
	req := middlewares.MustGetBody[UpdateServiceGroupRequest](r.Context())

	sg, err := h.commander.Update(r.Context(), id, req.Name)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, serviceGroupToResponse(sg))
}

func (h *ServiceGroupHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())

	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ServiceGroupResponse represents the response body for service group operations
type ServiceGroupResponse struct {
	ID        properties.UUID `json:"id"`
	Name      string          `json:"name"`
	CreatedAt JSONUTCTime     `json:"createdAt"`
	UpdatedAt JSONUTCTime     `json:"updatedAt"`
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
