// Generic issuance accepts only audience, optional service ID and opaque ext.
// Identity and authorized resource context are resolved by native Core commands.
package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/jwtissuer"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/response"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-jose/go-jose/v4/jwt"
)

type TokenIssueHandler struct {
	commander *domain.TokenIssueCommander
	signer    *jwtissuer.Signer
}

func NewTokenIssueHandler(commander *domain.TokenIssueCommander, signer *jwtissuer.Signer) *TokenIssueHandler {
	return &TokenIssueHandler{commander: commander, signer: signer}
}

func (h *TokenIssueHandler) Routes() func(chi.Router) {
	return func(r chi.Router) { r.Post("/issue", h.Issue) }
}

type IssueTokenReq struct {
	Audience  string           `json:"audience"`
	ServiceID *properties.UUID `json:"serviceId,omitempty"`
	Ext       map[string]any   `json:"ext,omitempty"`
}

type IssueTokenRes struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (h *TokenIssueHandler) Issue(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	// This narrow signing surface rejects unknown fields and trailing JSON rather
	// than accepting caller-supplied identity, lifetime or arbitrary claims.
	var req *IssueTokenReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		render.Render(w, r, response.ErrInvalidRequest(err))
		return
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		render.Render(w, r, ErrDomain(domain.NewInvalidInputErrorf("expected one JSON object")))
		return
	}
	if req == nil || strings.TrimSpace(req.Audience) == "" || (req.ServiceID != nil && *req.ServiceID == (properties.UUID{})) {
		render.Render(w, r, ErrDomain(domain.NewInvalidInputErrorf("audience and any supplied serviceId must be nonempty")))
		return
	}
	if err := jwtissuer.ValidateExt(req.Ext); err != nil {
		render.Render(w, r, ErrDomain(domain.NewInvalidInputErrorf("%s", err.Error())))
		return
	}
	principal, err := h.commander.Resolve(r.Context(), req.ServiceID)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	if h.signer == nil {
		render.Render(w, r, &response.ErrRes{Status: http.StatusServiceUnavailable, Message: "JWT signing is not configured"})
		return
	}
	claims := jwtissuer.Claims{Claims: jwt.Claims{Audience: jwt.Audience{req.Audience}}, Role: string(principal.Role), ParticipantID: principal.ParticipantID.String(), Ext: req.Ext}
	if principal.AgentID != nil {
		claims.AgentID = principal.AgentID.String()
	}
	if principal.Service != nil {
		claims.ServiceID = principal.Service.ID.String()
		claims.Relation = principal.Relation
	}
	token, expiresAt, err := h.signer.Sign(claims)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	if err := h.commander.Record(r.Context(), principal, req.Audience); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, IssueTokenRes{Token: token, ExpiresAt: expiresAt})
}
