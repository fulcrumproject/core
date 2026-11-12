// CMP launch handlers authorize a participant's browser jump into one service.
// Identity and destination come from credentials and persisted agent settings.
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
	"github.com/go-jose/go-jose/v4/jwt"
)

// CmpLaunchHandler can be mounted alongside an application's service routes.
// A nil signer keeps issuance unavailable without disabling existing APIs.
type CmpLaunchHandler struct {
	services  domain.ServiceQuerier
	agents    domain.AgentQuerier
	commander *domain.CmpLaunchCommander
	authz     authz.Authorizer
	signer    *sso.Signer
}

func NewCmpLaunchHandler(services domain.ServiceQuerier, agents domain.AgentQuerier, commander *domain.CmpLaunchCommander, authorizer authz.Authorizer, signer *sso.Signer) *CmpLaunchHandler {
	return &CmpLaunchHandler{services: services, agents: agents, commander: commander, authz: authorizer, signer: signer}
}

func (h *CmpLaunchHandler) Routes() func(chi.Router) {
	return func(r chi.Router) {
		r.With(
			middlewares.ID,
			middlewares.DecodeBody[CmpLaunchTokenReq](),
			middlewares.AuthzFromID(authz.ObjectTypeService, authz.ActionCmpLaunch, h.authz, h.services.AuthScope),
		).Post("/{id}/cmp-launch-token", h.CmpLaunchToken)
	}
}

// CmpLaunchTokenReq carries optional, opaque marketplace-asserted attributes.
type CmpLaunchTokenReq struct {
	Ext map[string]any `json:"ext,omitempty"`
}

// CmpLaunchTokenRes is submitted by the browser to redirectUrl using POST.
type CmpLaunchTokenRes struct {
	RedirectURL string    `json:"redirectUrl"`
	Token       string    `json:"token"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func (h *CmpLaunchHandler) CmpLaunchToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)
	req := middlewares.MustGetBody[CmpLaunchTokenReq](ctx)
	if err := sso.ValidateExt(req.Ext); err != nil {
		render.Render(w, r, ErrDomain(domain.NewInvalidInputErrorf("%s", err.Error())))
		return
	}

	identity := auth.MustGetIdentity(ctx)
	pid := identity.Scope.ParticipantID
	// Defense in depth: agents carry a participant scope too, but cannot launch.
	if identity.Role != auth.RoleParticipant || pid == nil {
		render.Render(w, r, response.ErrUnauthorized(domain.NewUnauthorizedErrorf("only a participant of the service can launch its CMP")))
		return
	}
	svc, err := h.services.Get(ctx, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	var relation string
	switch *pid {
	case svc.ProviderID:
		relation = "provider"
	case svc.ConsumerID:
		relation = "consumer"
	default:
		render.Render(w, r, response.ErrUnauthorized(domain.NewUnauthorizedErrorf("caller is neither provider nor consumer of the service")))
		return
	}
	if h.signer == nil {
		render.Render(w, r, &response.ErrRes{Status: http.StatusServiceUnavailable, Message: "CMP SSO signing is not configured"})
		return
	}
	agent, err := h.agents.Get(ctx, svc.AgentID)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	landingURL, audience, ok := cmpTarget(agent.Configuration)
	if !ok {
		render.Render(w, r, &response.ErrRes{Status: http.StatusUnprocessableEntity, Message: "the service's agent has no CMP configured"})
		return
	}
	token, expiresAt, err := h.signer.Sign(sso.LaunchClaims{
		Claims:        jwt.Claims{Subject: pid.String(), Audience: jwt.Audience{audience}},
		ParticipantID: pid.String(), ServiceID: svc.ID.String(), Relation: relation, Ext: req.Ext,
	})
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	if err := h.commander.Record(ctx, svc, relation, audience); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	render.JSON(w, r, CmpLaunchTokenRes{RedirectURL: landingURL, Token: token, ExpiresAt: expiresAt})
}

// cmpTarget preserves CEM's target lookup. Agent-type configuration schemas
// must restrict cmpLandingUrl to an administrator-approved HTTPS destination.
func cmpTarget(cfg *properties.JSON) (landingURL, audience string, ok bool) {
	if cfg == nil {
		return "", "", false
	}
	landingURL, lok := (*cfg)["cmpLandingUrl"].(string)
	audience, aok := (*cfg)["cmpAudience"].(string)
	return landingURL, audience, lok && aok && landingURL != "" && audience != ""
}
