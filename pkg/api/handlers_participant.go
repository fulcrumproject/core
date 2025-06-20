package api

import (
	"net/http"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateParticipantRequest struct {
	Name   string                   `json:"name"`
	Status domain.ParticipantStatus `json:"status"`
}

type UpdateParticipantRequest struct {
	Name   *string                   `json:"name"`
	Status *domain.ParticipantStatus `json:"status"`
}

type ParticipantHandler struct {
	querier   domain.ParticipantQuerier
	commander domain.ParticipantCommander
	authz     auth.Authorizer
}

func NewParticipantHandler(
	querier domain.ParticipantQuerier,
	commander domain.ParticipantCommander,
	authz auth.Authorizer,
) *ParticipantHandler {
	return &ParticipantHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all participant routes registered
func (h *ParticipantHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeParticipant, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, participantToResponse))

		// Create endpoint - decode body, then simple authorization
		r.With(
			middlewares.DecodeBody[CreateParticipantRequest](),
			middlewares.AuthzSimple(authz.ObjectTypeParticipant, authz.ActionCreate, h.authz),
		).Post("/", h.handleCreate)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using participant's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeParticipant, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier, participantToResponse))

			// Update endpoint - decode body, authorize using participant's scope
			r.With(
				middlewares.DecodeBody[UpdateParticipantRequest](),
				middlewares.AuthzFromID(authz.ObjectTypeParticipant, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", h.handleUpdate)

			// Delete endpoint - authorize using participant's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeParticipant, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

func (h *ParticipantHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	req := middlewares.MustGetBody[CreateParticipantRequest](r.Context())

	participant, err := h.commander.Create(r.Context(), req.Name, req.Status)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, participantToResponse(participant))
}

func (h *ParticipantHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := middlewares.MustGetID(r.Context())
	req := middlewares.MustGetBody[UpdateParticipantRequest](r.Context())

	participant, err := h.commander.Update(r.Context(), id, req.Name, req.Status)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, participantToResponse(participant))
}

// ParticipantResponse represents the response body for participant operations
type ParticipantResponse struct {
	ID        properties.UUID          `json:"id"`
	Name      string                   `json:"name"`
	Status    domain.ParticipantStatus `json:"status"`
	CreatedAt JSONUTCTime              `json:"createdAt"`
	UpdatedAt JSONUTCTime              `json:"updatedAt"`
}

// participantToResponse converts a domain.Participant to a ParticipantResponse
func participantToResponse(p *domain.Participant) *ParticipantResponse {
	return &ParticipantResponse{
		ID:        p.ID,
		Name:      p.Name,
		Status:    p.Status,
		CreatedAt: JSONUTCTime(p.CreatedAt),
		UpdatedAt: JSONUTCTime(p.UpdatedAt),
	}
}
