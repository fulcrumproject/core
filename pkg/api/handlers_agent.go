package api

import (
	"context"
	"net/http"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreateAgentReq struct {
	Name        string          `json:"name"`
	ProviderID  properties.UUID `json:"providerId"`
	AgentTypeID properties.UUID `json:"agentTypeId"`
	Tags        []string        `json:"tags"`
}

// auth.ObjectScope implements auth.ObjectScopeProvider interface
func (r CreateAgentReq) ObjectScope() (auth.ObjectScope, error) {
	return &auth.DefaultObjectScope{ParticipantID: &r.ProviderID}, nil
}

type UpdateAgentReq struct {
	Name   *string             `json:"name"`
	Status *domain.AgentStatus `json:"status"`
	Tags   *[]string           `json:"tags"`
}

type UpdateAgentStatusReq struct {
	Status domain.AgentStatus `json:"status"`
}

type AgentHandler struct {
	querier   domain.AgentQuerier
	commander domain.AgentCommander
	authz     auth.Authorizer
}

func NewAgentHandler(
	querier domain.AgentQuerier,
	commander domain.AgentCommander,
	authz auth.Authorizer,
) *AgentHandler {
	return &AgentHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

func (h *AgentHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeAgent, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, AgentToRes))

		// Create endpoint - using standard Create handler
		r.With(
			middlewares.DecodeBody[CreateAgentReq](),
			middlewares.AuthzFromBody[CreateAgentReq](authz.ObjectTypeAgent, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, AgentToRes))

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using agent's provider
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgent, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, AgentToRes))

			// Update endpoint - using standard Update handler
			r.With(
				middlewares.DecodeBody[UpdateAgentReq](),
				middlewares.AuthzFromID(authz.ObjectTypeAgent, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, AgentToRes))

			// Delete endpoint - authorize using agent's provider
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgent, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})

		// Agent-specific routes (me endpoints)
		// Note: These endpoints have special auth requirements
		r.With(
			middlewares.MustHaveRoles(auth.RoleAgent),
			middlewares.DecodeBody[UpdateAgentStatusReq](),
		).Put("/me/status", UpdateWithoutID(h.UpdateStatusMe, AgentToRes))

		r.With(
			middlewares.MustHaveRoles(auth.RoleAgent),
		).Get("/me", h.GetMe)
	}
}

// Adapter functions that convert request structs to commander method calls
func (h *AgentHandler) Create(ctx context.Context, req *CreateAgentReq) (*domain.Agent, error) {
	return h.commander.Create(ctx, req.Name, req.ProviderID, req.AgentTypeID, req.Tags)
}

// Adapter functions that convert request structs to commander method calls
func (h *AgentHandler) Update(ctx context.Context, id properties.UUID, req *UpdateAgentReq) (*domain.Agent, error) {
	return h.commander.Update(ctx, id, req.Name, req.Status, req.Tags)
}

// Adapter functions that convert request structs to commander method calls
func (h *AgentHandler) UpdateStatusMe(ctx context.Context, req *UpdateAgentStatusReq) (*domain.Agent, error) {
	agentID := auth.MustGetIdentity(ctx).Scope.AgentID
	return h.commander.UpdateStatus(ctx, *agentID, req.Status)
}

// GetMe handles GET /agents/me
// This endpoint allows agents to retrieve their own information
func (h *AgentHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	agentID := auth.MustGetIdentity(r.Context()).Scope.AgentID

	agent, err := h.querier.Get(r.Context(), *agentID)
	if err != nil {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, AgentToRes(agent))
}

// AgentRes represents the response body for agent operations
type AgentRes struct {
	ID          properties.UUID    `json:"id"`
	Name        string             `json:"name"`
	Status      domain.AgentStatus `json:"status"`
	ProviderID  properties.UUID    `json:"providerId"`
	AgentTypeID properties.UUID    `json:"agentTypeId"`
	Tags        []string           `json:"tags"`
	Participant *ParticipantRes    `json:"participant,omitempty"`
	AgentType   *AgentTypeRes      `json:"agentType,omitempty"`
	CreatedAt   JSONUTCTime        `json:"createdAt"`
	UpdatedAt   JSONUTCTime        `json:"updatedAt"`
}

// AgentToRes converts a domain.Agent to an AgentResponse
func AgentToRes(a *domain.Agent) *AgentRes {
	response := &AgentRes{
		ID:          a.ID,
		Name:        a.Name,
		Status:      a.Status,
		ProviderID:  a.ProviderID,
		AgentTypeID: a.AgentTypeID,
		Tags:        []string(a.Tags),
		CreatedAt:   JSONUTCTime(a.CreatedAt),
		UpdatedAt:   JSONUTCTime(a.UpdatedAt),
	}
	if a.Provider != nil {
		response.Participant = ParticipantToRes(a.Provider)
	}
	if a.AgentType != nil {
		response.AgentType = AgentTypeToRes(a.AgentType)
	}
	return response
}
