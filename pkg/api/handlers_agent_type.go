package api

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/chi/v5"
)

type AgentTypeHandler struct {
	querier   domain.AgentTypeQuerier
	commander domain.AgentTypeCommander
	authz     auth.Authorizer
}

func NewAgentTypeHandler(
	querier domain.AgentTypeQuerier,
	commander domain.AgentTypeCommander,
	authz auth.Authorizer,
) *AgentTypeHandler {
	return &AgentTypeHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all agent type routes registered
func (h *AgentTypeHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// List endpoint - simple authorization
		r.With(
			middlewares.AuthzSimple(authz.ObjectTypeAgentType, authz.ActionRead, h.authz),
		).Get("/", List(h.querier, AgentTypeToRes))

		// Create endpoint - admin only
		r.With(
			middlewares.DecodeBody[CreateAgentTypeReq](),
			middlewares.AuthzSimple(authz.ObjectTypeAgentType, authz.ActionCreate, h.authz),
		).Post("/", Create(h.Create, AgentTypeToRes))

		// Resource-specific routes with ID
		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			// Get endpoint - authorize using agent type's scope
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgentType, authz.ActionRead, h.authz, h.querier.AuthScope),
			).Get("/{id}", Get(h.querier.Get, AgentTypeToRes))

			// Update endpoint - admin only
			r.With(
				middlewares.DecodeBody[UpdateAgentTypeReq](),
				middlewares.AuthzFromID(authz.ObjectTypeAgentType, authz.ActionUpdate, h.authz, h.querier.AuthScope),
			).Patch("/{id}", Update(h.Update, AgentTypeToRes))

			// Delete endpoint - admin only
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgentType, authz.ActionDelete, h.authz, h.querier.AuthScope),
			).Delete("/{id}", Delete(h.querier, h.commander.Delete))
		})
	}
}

// CreateAgentTypeReq represents the request body for creating agent types
type CreateAgentTypeReq struct {
	Name                string            `json:"name"`
	ServiceTypeIds      []properties.UUID `json:"serviceTypeIds,omitempty"`
	ConfigurationSchema schema.Schema     `json:"configurationSchema"`
}

// UpdateAgentTypeReq represents the request body for updating agent types
type UpdateAgentTypeReq struct {
	Name                *string            `json:"name"`
	ServiceTypeIds      *[]properties.UUID `json:"serviceTypeIds,omitempty"`
	ConfigurationSchema *schema.Schema     `json:"configurationSchema,omitempty"`
}

// AgentTypeRes represents the response body for agent type operations
type AgentTypeRes struct {
	ID                  properties.UUID   `json:"id"`
	Name                string            `json:"name"`
	CreatedAt           JSONUTCTime       `json:"createdAt"`
	UpdatedAt           JSONUTCTime       `json:"updatedAt"`
	ServiceTypeIds      []properties.UUID `json:"serviceTypeIds"`
	ConfigurationSchema schema.Schema     `json:"configurationSchema"`
}

// AgentTypeToRes converts a domain.AgentType to an AgentTypeResponse
func AgentTypeToRes(at *domain.AgentType) *AgentTypeRes {
	response := &AgentTypeRes{
		ID:                  at.ID,
		Name:                at.Name,
		CreatedAt:           JSONUTCTime(at.CreatedAt),
		UpdatedAt:           JSONUTCTime(at.UpdatedAt),
		ServiceTypeIds:      make([]properties.UUID, 0),
		ConfigurationSchema: at.ConfigurationSchema,
	}
	for _, st := range at.ServiceTypes {
		response.ServiceTypeIds = append(response.ServiceTypeIds, st.ID)
	}
	return response
}

// Adapter functions that convert request structs to commander method calls

func (h *AgentTypeHandler) Create(ctx context.Context, req *CreateAgentTypeReq) (*domain.AgentType, error) {
	params := domain.CreateAgentTypeParams{
		Name:                req.Name,
		ServiceTypeIds:      req.ServiceTypeIds,
		ConfigurationSchema: req.ConfigurationSchema,
	}
	return h.commander.Create(ctx, params)
}

func (h *AgentTypeHandler) Update(ctx context.Context, id properties.UUID, req *UpdateAgentTypeReq) (*domain.AgentType, error) {
	params := domain.UpdateAgentTypeParams{
		ID:                  id,
		Name:                req.Name,
		ServiceTypeIds:      req.ServiceTypeIds,
		ConfigurationSchema: req.ConfigurationSchema,
	}
	return h.commander.Update(ctx, params)
}
