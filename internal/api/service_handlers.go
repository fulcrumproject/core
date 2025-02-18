package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// CreateUpdateServiceRequest represents the request body for creating/updating a service
type CreateUpdateServiceRequest struct {
	Name          string                 `json:"name"`
	State         domain.ServiceState    `json:"state"`
	Attributes    domain.Attributes      `json:"attributes"`
	Resources     map[string]interface{} `json:"resources"`
	AgentID       string                 `json:"agentId"`
	ServiceTypeID string                 `json:"serviceTypeId"`
	GroupID       string                 `json:"groupId,omitempty"`
}

// ServiceResponse represents the response body for service operations
type ServiceResponse struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	State         domain.ServiceState    `json:"state"`
	Attributes    domain.Attributes      `json:"attributes"`
	Resources     map[string]interface{} `json:"resources"`
	AgentID       string                 `json:"agentId"`
	ServiceTypeID string                 `json:"serviceTypeId"`
	GroupID       string                 `json:"groupId,omitempty"`
	CreatedAt     JSONUTCTime            `json:"createdAt"`
	UpdatedAt     JSONUTCTime            `json:"updatedAt"`

	// Relationships
	Agent       *AgentResponse        `json:"agent,omitempty"`
	ServiceType *ServiceTypeResponse  `json:"serviceType,omitempty"`
	Group       *ServiceGroupResponse `json:"group,omitempty"`
}

// serviceToResponse converts a domain.Service to a ServiceResponse
func serviceToResponse(s *domain.Service) *ServiceResponse {
	resp := &ServiceResponse{
		ID:            s.ID.String(),
		Name:          s.Name,
		State:         s.State,
		Attributes:    s.Attributes,
		Resources:     s.Resources,
		AgentID:       s.AgentID.String(),
		ServiceTypeID: s.ServiceTypeID.String(),
		CreatedAt:     JSONUTCTime(s.CreatedAt),
		UpdatedAt:     JSONUTCTime(s.UpdatedAt),
	}

	if s.GroupID != (domain.UUID{}) {
		resp.GroupID = s.GroupID.String()
	}

	if s.Agent != nil {
		resp.Agent = agentToResponse(s.Agent)
	}
	if s.ServiceType != nil {
		resp.ServiceType = serviceTypeToResponse(s.ServiceType)
	}
	if s.Group != nil {
		resp.Group = serviceGroupToResponse(s.Group)
	}

	return resp
}

type ServiceHandler struct {
	repo domain.ServiceRepository
}

func NewServiceHandler(repo domain.ServiceRepository) *ServiceHandler {
	return &ServiceHandler{repo: repo}
}

// Routes returns the router with all service routes registered
func (h *ServiceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware for all service routes
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", h.handleList)
	r.Post("/", h.handleCreate)
	r.Get("/{id}", h.handleGet)
	r.Put("/{id}", h.handleUpdate)
	r.Delete("/{id}", h.handleDelete)

	return r
}

func (h *ServiceHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateUpdateServiceRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	agentID, err := domain.ParseID(req.AgentID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	serviceTypeID, err := domain.ParseID(req.ServiceTypeID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	service := &domain.Service{
		Name:          req.Name,
		State:         req.State,
		Attributes:    req.Attributes,
		Resources:     req.Resources,
		AgentID:       agentID,
		ServiceTypeID: serviceTypeID,
	}

	if req.GroupID != "" {
		groupID, err := domain.ParseID(req.GroupID)
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		service.GroupID = groupID
	}

	if err := h.repo.Create(r.Context(), service); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	service, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleList(w http.ResponseWriter, r *http.Request) {
	filter := parseSimpleFilter(r)
	sorting := parseSorting(r)
	pagination := parsePagination(r)

	result, err := h.repo.List(r.Context(), filter, sorting, pagination)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPaginatedResponse(result, serviceToResponse))
}

func (h *ServiceHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	var req CreateUpdateServiceRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	service, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	agentID, err := domain.ParseID(req.AgentID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	serviceTypeID, err := domain.ParseID(req.ServiceTypeID)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// Update fields
	service.Name = req.Name
	service.State = req.State
	service.Attributes = req.Attributes
	service.Resources = req.Resources
	service.AgentID = agentID
	service.ServiceTypeID = serviceTypeID

	if req.GroupID != "" {
		groupID, err := domain.ParseID(req.GroupID)
		if err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		service.GroupID = groupID
	} else {
		service.GroupID = domain.UUID{}
	}

	if err := h.repo.Save(r.Context(), service); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	render.JSON(w, r, serviceToResponse(service))
}

func (h *ServiceHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	_, err = h.repo.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
