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
}

func NewAuditEntryHandler(
	querier domain.AuditEntryRepository,
	commander domain.AuditEntryCommander,

) *AuditEntryHandler {
	return &AuditEntryHandler{
		querier:   querier,
		commander: commander,
	}
}

// Routes returns the router with all audit entry routes registered
func (h *AuditEntryHandler) Routes(authzMW AuthzMiddlewareFunc) func(r chi.Router) {
	return func(r chi.Router) {
		r.With(authzMW(domain.SubjectAuditEntry, domain.ActionList)).Get("/", h.handleList)
		r.With(authzMW(domain.SubjectAuditEntry, domain.ActionCreate)).Post("/", h.handleCreate)
	}
}

func (h *AuditEntryHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var p struct {
		AuthorityType string      `json:"authorityType"`
		AuthorityID   string      `json:"authorityId"`
		Type          string      `json:"type"`
		Properties    domain.JSON `json:"properties"`
	}
	if err := render.Decode(r, &p); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	auditEntry, err := h.commander.Create(r.Context(), p.AuthorityType, p.AuthorityID, p.Type, p.Properties)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, auditEntryToResponse(auditEntry))
}

func (h *AuditEntryHandler) handleList(w http.ResponseWriter, r *http.Request) {
	pag, err := parsePageRequest(r)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	result, err := h.querier.List(r.Context(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, NewPageResponse(result, auditEntryToResponse))
}

// AuditEntryResponse represents the response body for audit entry operations
type AuditEntryResponse struct {
	ID            domain.UUID `json:"id"`
	AuthorityType string      `json:"authorityType"`
	AuthorityID   string      `json:"authorityId"`
	Type          string      `json:"type"`
	Properties    domain.JSON `json:"properties"`
	CreatedAt     JSONUTCTime `json:"createdAt"`
	UpdatedAt     JSONUTCTime `json:"updatedAt"`
}

// auditEntryToResponse converts a domain.AuditEntry to an AuditEntryResponse
func auditEntryToResponse(ae *domain.AuditEntry) *AuditEntryResponse {
	return &AuditEntryResponse{
		ID:            ae.ID,
		AuthorityType: ae.AuthorityType,
		AuthorityID:   ae.AuthorityID,
		Type:          ae.Type,
		Properties:    ae.Properties,
		CreatedAt:     JSONUTCTime(ae.CreatedAt),
		UpdatedAt:     JSONUTCTime(ae.UpdatedAt),
	}
}
