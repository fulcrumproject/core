package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateParticipantRequest struct {
	Name        string                   `json:"name"`
	Status      domain.ParticipantStatus `json:"status"`
	CountryCode domain.CountryCode       `json:"countryCode,omitempty"`
	Attributes  domain.Attributes        `json:"attributes,omitempty"`
}

type UpdateParticipantRequest struct {
	Name        *string                   `json:"name"`
	Status      *domain.ParticipantStatus `json:"status"`
	CountryCode *domain.CountryCode       `json:"countryCode,omitempty"`
	Attributes  *domain.Attributes        `json:"attributes,omitempty"`
}

type ParticipantHandler struct {
	querier   domain.ParticipantQuerier
	commander domain.ParticipantCommander
	authz     domain.Authorizer
}

func NewParticipantHandler(
	querier domain.ParticipantQuerier,
	commander domain.ParticipantCommander,
	authz domain.Authorizer,
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
			AuthzSimple(domain.SubjectParticipant, domain.ActionRead, h.authz),
		).Get("/", h.handleList)

		// Create endpoint - decode body, then simple authorization
		r.With(
			DecodeBody[CreateParticipantRequest](),
			AuthzSimple(domain.SubjectParticipant, domain.ActionCreate, h.authz),
		).Post("/", h.handleCreate)

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(ID)

			// Get endpoint - authorize using participant's scope
			r.With(
				AuthzFromID(domain.SubjectParticipant, domain.ActionRead, h.authz, h.querier),
			).Get("/{id}", h.handleGet)

			// Update endpoint - decode body, authorize using participant's scope
			r.With(
				DecodeBody[UpdateParticipantRequest](),
				AuthzFromID(domain.SubjectParticipant, domain.ActionUpdate, h.authz, h.querier),
			).Patch("/{id}", h.handleUpdate)

			// Delete endpoint - authorize using participant's scope
			r.With(
				AuthzFromID(domain.SubjectParticipant, domain.ActionDelete, h.authz, h.querier),
			).Delete("/{id}", h.handleDelete)
		})
	}
}

func (h *ParticipantHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	req := MustGetBody[CreateParticipantRequest](r.Context())

	participant, err := h.commander.Create(r.Context(), req.Name, req.Status, req.CountryCode, req.Attributes)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, participantToResponse(participant))
}

func (h *ParticipantHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	participant, err := h.querier.FindByID(r.Context(), id)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, participantToResponse(participant))
}

func (h *ParticipantHandler) handleList(w http.ResponseWriter, r *http.Request) {
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

	render.JSON(w, r, NewPageResponse(result, participantToResponse))
}

func (h *ParticipantHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())
	req := MustGetBody[UpdateParticipantRequest](r.Context())

	participant, err := h.commander.Update(r.Context(), id, req.Name, req.Status, req.CountryCode, req.Attributes)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, participantToResponse(participant))
}

func (h *ParticipantHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := MustGetID(r.Context())

	if err := h.commander.Delete(r.Context(), id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ParticipantResponse represents the response body for participant operations
type ParticipantResponse struct {
	ID          domain.UUID              `json:"id"`
	Name        string                   `json:"name"`
	Status      domain.ParticipantStatus `json:"status"`
	CountryCode string                   `json:"countryCode,omitempty"`
	Attributes  map[string][]string      `json:"attributes,omitempty"`
	CreatedAt   JSONUTCTime              `json:"createdAt"`
	UpdatedAt   JSONUTCTime              `json:"updatedAt"`
}

// participantToResponse converts a domain.Participant to a ParticipantResponse
func participantToResponse(p *domain.Participant) *ParticipantResponse {
	return &ParticipantResponse{
		ID:          p.ID,
		Name:        p.Name,
		Status:      p.Status,
		CountryCode: string(p.CountryCode),
		Attributes:  map[string][]string(p.Attributes),
		CreatedAt:   JSONUTCTime(p.CreatedAt),
		UpdatedAt:   JSONUTCTime(p.UpdatedAt),
	}
}
