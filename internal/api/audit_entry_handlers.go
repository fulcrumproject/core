package api

import (
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// CreateAuditEntryRequest represents the request body for creating an audit entry
type CreateAuditEntryRequest struct {
	AuthorityType string      `json:"authorityType"`
	AuthorityID   string      `json:"authorityId"`
	Type          string      `json:"type"`
	Properties    domain.JSON `json:"properties"`
}

// AuditEntryResponse represents the response body for audit entry operations
type AuditEntryResponse struct {
	ID            string      `json:"id"`
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
		ID:            ae.ID.String(),
		AuthorityType: ae.AuthorityType,
		AuthorityID:   ae.AuthorityID,
		Type:          ae.Type,
		Properties:    ae.Properties,
		CreatedAt:     JSONUTCTime(ae.CreatedAt),
		UpdatedAt:     JSONUTCTime(ae.UpdatedAt),
	}
}

type AuditEntryHandler struct {
	repo domain.AuditEntryRepository
}

func NewAuditEntryHandler(repo domain.AuditEntryRepository) *AuditEntryHandler {
	return &AuditEntryHandler{repo: repo}
}

// Routes returns the router with all audit entry routes registered
func (h *AuditEntryHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Middleware for all audit entry routes
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", h.handleList)
	r.Post("/", h.handleCreate)

	return r
}

func (h *AuditEntryHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateAuditEntryRequest
	if err := render.Decode(r, &req); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	auditEntry := &domain.AuditEntry{
		AuthorityType: req.AuthorityType,
		AuthorityID:   req.AuthorityID,
		Type:          req.Type,
		Properties:    req.Properties,
	}

	if err := h.repo.Create(r.Context(), auditEntry); err != nil {
		render.Render(w, r, ErrInternal(err))
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

	result, err := h.repo.List(r.Context(), pag)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	render.JSON(w, r, NewPageResponse(result, auditEntryToResponse))
}
