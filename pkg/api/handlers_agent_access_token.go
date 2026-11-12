// Agent access tokens authorize applications using operator-owned module grants.
// The authenticated agent scope is the only request input used for issuance.
package api

import (
	"net/http"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/response"
	"github.com/fulcrumproject/core/pkg/sso"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// AgentAccessTokenHandler can be mounted alongside an application's agent routes.
// Its policy must be built from Core-operator configuration, not agent metadata.
type AgentAccessTokenHandler struct {
	agents    domain.AgentQuerier
	commander *domain.AgentAccessTokenCommander
	authz     authz.Authorizer
	tokens    *sso.ApplicationTokens
}

func NewAgentAccessTokenHandler(agents domain.AgentQuerier, commander *domain.AgentAccessTokenCommander, authorizer authz.Authorizer, tokens *sso.ApplicationTokens) *AgentAccessTokenHandler {
	return &AgentAccessTokenHandler{agents: agents, commander: commander, authz: authorizer, tokens: tokens}
}

func (h *AgentAccessTokenHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.With(middlewares.AuthzSimple(authz.ObjectTypeAgent, authz.ActionAccessToken, h.authz)).Post("/me/access-token", h.AccessToken)
	}
}

type AgentAccessTokenRes struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// AccessToken ignores bodies and query parameters, including malformed JSON.
// No decoder or caller-selected claims/settings participate in this profile.
func (h *AgentAccessTokenHandler) AccessToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	identity := auth.MustGetIdentity(ctx)
	if identity.Role != auth.RoleAgent || identity.Scope.AgentID == nil || *identity.Scope.AgentID == (properties.UUID{}) || identity.Scope.ParticipantID == nil {
		render.Render(w, r, response.ErrUnauthorized(domain.NewUnauthorizedErrorf("an authenticated agent scope is required")))
		return
	}
	agent, err := h.agents.Get(ctx, *identity.Scope.AgentID)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	if agent == nil || agent.ID != *identity.Scope.AgentID || agent.ProviderID != *identity.Scope.ParticipantID {
		render.Render(w, r, response.ErrUnauthorized(domain.NewUnauthorizedErrorf("agent identity does not match persisted scope")))
		return
	}
	if !h.tokens.Enabled() {
		render.Render(w, r, &response.ErrRes{Status: http.StatusServiceUnavailable, Message: "application signing is not configured"})
		return
	}
	grant, ok := h.tokens.Grant(agent.ID)
	if !ok {
		render.Render(w, r, response.ErrUnauthorized(domain.NewUnauthorizedErrorf("agent has no application grant")))
		return
	}
	token, expiresAt, err := h.tokens.Issue(agent.ID)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	if err := h.commander.Record(ctx, agent, grant.Audience, grant.Roles); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	render.JSON(w, r, AgentAccessTokenRes{Token: token, ExpiresAt: expiresAt})
}
