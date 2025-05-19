package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type AuditEntryHandler struct {
	querier   domain.AuditEntryQuerier
	commander domain.AuditEntryCommander
	authz     domain.Authorizer
}

func NewAuditEntryHandler(
	querier domain.AuditEntryQuerier,
	commander domain.AuditEntryCommander,
	authz domain.Authorizer,
) *AuditEntryHandler {
	return &AuditEntryHandler{
		querier:   querier,
		commander: commander,
		authz:     authz,
	}
}

// Routes returns the router with all audit entry routes registered
func (h *AuditEntryHandler) Routes() func(r chi.Router) {

	return func(r chi.Router) {
		r.Get("/", h.handleList)
	}
}

func (h *AuditEntryHandler) handleList(w http.ResponseWriter, r *http.Request) {
	id := domain.MustGetAuthIdentity(r.Context())
	if err := h.authz.Authorize(id, domain.SubjectAuditEntry, domain.ActionRead, &domain.EmptyAuthScope); err != nil {
		render.Render(w, r, ErrUnauthorized(err))
		return
	}
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
	render.JSON(w, r, NewPageResponse(result, auditEntryToResponse))
}

// AuditEntryResponse represents the response body for audit entry operations
type AuditEntryResponse struct {
	ID            domain.UUID          `json:"id"`
	AuthorityType domain.AuthorityType `json:"authorityType"`
	AuthorityID   string               `json:"authorityId"`
	Type          domain.EventType     `json:"type"`
	Properties    domain.JSON          `json:"properties"`
	ProviderID    *domain.UUID         `json:"providerId,omitempty"`
	AgentID       *domain.UUID         `json:"agentId,omitempty"`
	BrokerID      *domain.UUID         `json:"brokerId,omitempty"`
	CreatedAt     JSONUTCTime          `json:"createdAt"`
	UpdatedAt     JSONUTCTime          `json:"updatedAt"`
}

// auditEntryToResponse converts a domain.AuditEntry to an AuditEntryResponse
func auditEntryToResponse(ae *domain.AuditEntry) *AuditEntryResponse {
	return &AuditEntryResponse{
		ID:            ae.ID,
		AuthorityType: ae.AuthorityType,
		AuthorityID:   ae.AuthorityID,
		Type:          ae.EventType,
		Properties:    ae.Properties,
		ProviderID:    ae.ProviderID,
		AgentID:       ae.AgentID,
		BrokerID:      ae.ConsumerID,
		CreatedAt:     JSONUTCTime(ae.CreatedAt),
		UpdatedAt:     JSONUTCTime(ae.UpdatedAt),
	}
}
