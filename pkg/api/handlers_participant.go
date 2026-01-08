package api

import (
	"context"

	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
)

type CreateParticipantReq struct {
	Name   string                   `json:"name"`
	Status domain.ParticipantStatus `json:"status"`
}

type UpdateParticipantReq struct {
	Name   *string                   `json:"name"`
	Status *domain.ParticipantStatus `json:"status"`
}

type ParticipantHandler struct {
	querier   domain.ParticipantQuerier
	commander domain.ParticipantCommander
	authz     authz.Authorizer
}

func NewParticipantHandler(
	querier domain.ParticipantQuerier,
	commander domain.ParticipantCommander,
	authz authz.Authorizer,
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
		).Get("/", List(h.querier, ParticipantToRes))

		// Create endpoint - using standard Create handler
		r.With(
			middlewares.DecodeBody[CreateParticipantReq](),
			middlewares.AuthzSimple(authz.ObjectTypeParticipant, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, ParticipantToRes))

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using participant's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeParticipant, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, ParticipantToRes))

			// Update endpoint - using standard Update handler
			r.With(
				middlewares.DecodeBody[UpdateParticipantReq](),
				middlewares.AuthzFromID(authz.ObjectTypeParticipant, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, ParticipantToRes))

			// Delete endpoint - authorize using participant's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeParticipant, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

// Adapter functions that convert request structs to commander method calls

func (h *ParticipantHandler) Create(ctx context.Context, req *CreateParticipantReq) (*domain.Participant, error) {
	params := domain.CreateParticipantParams{
		Name:   req.Name,
		Status: req.Status,
	}
	return h.commander.Create(ctx, params)
}

func (h *ParticipantHandler) Update(ctx context.Context, id properties.UUID, req *UpdateParticipantReq) (*domain.Participant, error) {
	params := domain.UpdateParticipantParams{
		ID:     id,
		Name:   req.Name,
		Status: req.Status,
	}
	return h.commander.Update(ctx, params)
}

// ParticipantRes represents the response body for participant operations
type ParticipantRes struct {
	ID        properties.UUID          `json:"id"`
	Name      string                   `json:"name"`
	Status    domain.ParticipantStatus `json:"status"`
	CreatedAt JSONUTCTime              `json:"createdAt"`
	UpdatedAt JSONUTCTime              `json:"updatedAt"`
}

// ParticipantToRes converts a domain.Participant to a ParticipantResponse
func ParticipantToRes(p *domain.Participant) *ParticipantRes {
	return &ParticipantRes{
		ID:        p.ID,
		Name:      p.Name,
		Status:    p.Status,
		CreatedAt: JSONUTCTime(p.CreatedAt),
		UpdatedAt: JSONUTCTime(p.UpdatedAt),
	}
}
