package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type ServiceActivationHandler struct {
	querier   domain.ServiceActivationQuerier
	commander domain.ServiceActivationCommander
	authz     domain.Authorizer
}

func NewServiceActivationHandler(
	querier domain.ServiceActivationQuerier,
	commander domain.ServiceActivationCommander,
	authz domain.Authorizer,
) *ServiceActivationHandler {
	return &ServiceActivationHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Request types

// CreateServiceActivationRequest represents the request to create a service activation
type CreateServiceActivationRequest struct {
	ProviderID    domain.UUID   `json:"providerId"`
	ServiceTypeID domain.UUID   `json:"serviceTypeId"`
	Tags          []string      `json:"tags"`
	AgentIDs      []domain.UUID `json:"agentIds,omitempty"`
}

// UpdateServiceActivationRequest represents the request to update a service activation
type UpdateServiceActivationRequest struct {
	Tags     *[]string      `json:"tags,omitempty"`
	AgentIDs *[]domain.UUID `json:"agentIds,omitempty"`
}

// CreateServiceActivationScopeExtractor creates an extractor that gets scope from the request body
func CreateServiceActivationScopeExtractor() AuthTargetScopeExtractor {
	return func(r *http.Request) (*domain.AuthTargetScope, error) {
		// Get decoded body from context
		body := MustGetBody[CreateServiceActivationRequest](r.Context())

		return &domain.AuthTargetScope{
			ParticipantID: &body.ProviderID,
		}, nil
	}
}

// Routes returns the router with all service activation routes registered
func (h *ServiceActivationHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List - simple authorization
		r.With(
			AuthzSimple(domain.SubjectServiceActivation, domain.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create - decode body + authorization from body
		r.With(
			DecodeBody[CreateServiceActivationRequest](),
			AuthzFromExtractor(
				domain.SubjectServiceActivation,
				domain.ActionCreate,
				h.authz,
				CreateServiceActivationScopeExtractor(),
			),
		).Post("/", h.handleCreate)

		// Resource-specific routes
		r.Group(func(r chi.Router) {
			r.Use(ID)

			// Get - authorize from resource ID
			r.With(
				AuthzFromID(domain.SubjectServiceActivation, domain.ActionRead, h.authz, h.querier),
			).Get("/{id}", h.handleGet)

			// Update - decode body + authorize from resource ID
			r.With(
				DecodeBody[UpdateServiceActivationRequest](),
				AuthzFromID(domain.SubjectServiceActivation, domain.ActionUpdate, h.authz, h.querier),
			).Patch("/{id}", h.handleUpdate)

			// Delete - authorize from resource ID
			r.With(
				AuthzFromID(domain.SubjectServiceActivation, domain.ActionDelete, h.authz, h.querier),
			).Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *ServiceActivationHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	// Get decoded body from context
	body := MustGetBody[CreateServiceActivationRequest](r.Context())

	serviceActivation, err := h.commander.Create(
		r.Context(),
		body.ProviderID,
		body.ServiceTypeID,
		body.Tags,
		body.AgentIDs,
	)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, serviceActivationToResponse(serviceActivation))
}

func (h *ServiceActivationHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	serviceActivation, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, serviceActivationToResponse(serviceActivation))
}

func (h *ServiceActivationHandler) handleList(w http.ResponseWriter, r *http.Request) {
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

	render.JSON(w, r, NewPageResponse(result, serviceActivationToResponse))
}

func (h *ServiceActivationHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())
	body := MustGetBody[UpdateServiceActivationRequest](r.Context())

	serviceActivation, err := h.commander.Update(
		r.Context(),
		id,
		body.Tags,
		body.AgentIDs,
	)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, serviceActivationToResponse(serviceActivation))
}

func (h *ServiceActivationHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ServiceActivationResponse represents the response body for service activation operations
type ServiceActivationResponse struct {
	ID            domain.UUID   `json:"id"`
	ProviderID    domain.UUID   `json:"providerId"`
	ServiceTypeID domain.UUID   `json:"serviceTypeId"`
	Tags          []string      `json:"tags"`
	AgentIDs      []domain.UUID `json:"agentIds"`
	CreatedAt     JSONUTCTime   `json:"createdAt"`
	UpdatedAt     JSONUTCTime   `json:"updatedAt"`
}

// serviceActivationToResponse converts a domain.ServiceActivation to a ServiceActivationResponse
func serviceActivationToResponse(sa *domain.ServiceActivation) *ServiceActivationResponse {
	// Extract agent IDs from the Agents slice
	agentIDs := make([]domain.UUID, len(sa.Agents))
	for i, agent := range sa.Agents {
		agentIDs[i] = agent.ID
	}

	resp := &ServiceActivationResponse{
		ID:            sa.ID,
		ProviderID:    sa.ProviderID,
		ServiceTypeID: sa.ServiceTypeID,
		Tags:          []string(sa.Tags),
		AgentIDs:      agentIDs,
		CreatedAt:     JSONUTCTime(sa.CreatedAt),
		UpdatedAt:     JSONUTCTime(sa.UpdatedAt),
	}
	return resp
}
