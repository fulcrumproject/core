package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// InstallConfigSubPath is the install-config route relative to the /agents
// mount; used both by the chi route registration (it accepts {token} as a
// chi placeholder) and by buildInstallURL (which substitutes the placeholder
// with the actual plain token). Keeping a single source means renaming the
// route updates the rendered installCommand automatically.
const InstallConfigSubPath = "/install/{token}/config"

const installConfigFullPath = "/api/v1/agents" + InstallConfigSubPath

// buildInstallURL joins the public base URL with the install-config path,
// substituting the chi {token} placeholder. The endpoint is authenticated:
// callers must also supply a Bearer token (issued alongside the install
// token — see AgentInstallToken.BootstrapTokenID).
func buildInstallURL(publicBaseURL, token string) string {
	return strings.TrimRight(publicBaseURL, "/") + strings.Replace(installConfigFullPath, "{token}", token, 1)
}

type AgentInstallTokenHandler struct {
	querier        domain.AgentInstallTokenQuerier
	commander      domain.AgentInstallTokenCommander
	agentAuthScope middlewares.ObjectScopeLoader
	authz          authz.Authorizer
	vault          schema.Vault
	publicBaseURL  string
}

func NewAgentInstallTokenHandler(
	querier domain.AgentInstallTokenQuerier,
	commander domain.AgentInstallTokenCommander,
	agentAuthScope middlewares.ObjectScopeLoader,
	authorizer authz.Authorizer,
	vault schema.Vault,
	publicBaseURL string,
) *AgentInstallTokenHandler {
	return &AgentInstallTokenHandler{
		querier:        querier,
		commander:      commander,
		agentAuthScope: agentAuthScope,
		authz:          authorizer,
		vault:          vault,
		publicBaseURL:  publicBaseURL,
	}
}

// Routes registers all install-token endpoints. Mount under `/agents`
// alongside AgentHandler.Routes(); the install-token routes are split into
// the token-keyed Fetch (no agent ID in URL) and the agent-scoped CRUD
// (`/{id}/install-command…`).
func (h *AgentInstallTokenHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// Install-config fetch — token-keyed, no agent ID in URL. Requires a
		// bearer token with admin / participant / agent role in addition to
		// the install token (issued by POST /{id}/install-command). The
		// trailing /config segment exists to keep the path unambiguous
		// against /{id}/install-command.
		r.With(
			middlewares.MustHaveRoles(auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent),
		).Get(InstallConfigSubPath, h.Fetch)

		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgent, authz.ActionRead, h.authz, h.agentAuthScope),
			).Get("/{id}/install-command", h.Get)
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgent, authz.ActionUpdate, h.authz, h.agentAuthScope),
			).Post("/{id}/install-command", h.Create)
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgent, authz.ActionUpdate, h.authz, h.agentAuthScope),
			).Post("/{id}/install-command/regenerate", h.Regenerate)
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeAgent, authz.ActionUpdate, h.authz, h.agentAuthScope),
			).Delete("/{id}/install-command", h.Revoke)
		})
	}
}

// InstallTokenRes is the Create/Regenerate response — the plain token is
// rendered into InstallCommand and URL exactly once and is not recoverable
// afterwards.
type InstallTokenRes struct {
	InstallCommand string      `json:"installCommand"`
	URL            string      `json:"url"`
	ExpiresAt      JSONUTCTime `json:"expiresAt"`
}

// InstallTokenMetaRes is the GET response — metadata only, no token, no URL.
// If the admin lost the token, they must Regenerate.
type InstallTokenMetaRes struct {
	ExpiresAt JSONUTCTime `json:"expiresAt"`
	CreatedAt JSONUTCTime `json:"createdAt"`
}

func (h *AgentInstallTokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	tok, err := h.commander.Create(ctx, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	h.renderInstallToken(w, r, tok, http.StatusCreated)
}

func (h *AgentInstallTokenHandler) Regenerate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	tok, err := h.commander.Regenerate(ctx, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	h.renderInstallToken(w, r, tok, http.StatusOK)
}

// Get returns metadata about the current install token. It never returns the
// plain token nor the rendered install URL — once the token leaves the
// Create/Regenerate response, it cannot be recovered.
func (h *AgentInstallTokenHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	tok, err := h.querier.GetByAgentID(ctx, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	if tok.IsExpired() {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, InstallTokenMetaRes{
		ExpiresAt: JSONUTCTime(tok.ExpiresAt),
		CreatedAt: JSONUTCTime(tok.CreatedAt),
	})
}

func (h *AgentInstallTokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	if err := h.commander.Revoke(ctx, id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Fetch serves the rendered agent configuration to an installer. The route is
// guarded by the standard auth middleware (admin/participant/agent roles), so
// callers without a valid bearer token receive 401/403 from the middleware.
// Past that gate, every failure mode (unknown / expired install token, missing
// templates, vault resolution failure, render error) collapses into the
// codebase's standard 404 response — the precise reason is recorded only in
// the server-side log so ops can alert without leaking the cause to the
// caller. Info is used for client-caused misses, Warn for server-side
// failures.
func (h *AgentInstallTokenHandler) Fetch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := chi.URLParam(r, "token")
	if token == "" {
		slog.Info("install fetch → 404", "reason", "empty token")
		render.Render(w, r, ErrNotFound())
		return
	}

	tok, err := h.querier.FindByHashedToken(ctx, domain.HashTokenValue(token))
	if err != nil {
		if errors.As(err, &domain.NotFoundError{}) {
			slog.Info("install fetch → 404", "reason", "install token not found")
		} else {
			slog.Warn("install fetch → 404", "reason", "install token lookup failed: "+err.Error())
		}
		render.Render(w, r, ErrNotFound())
		return
	}
	if tok.IsExpired() {
		slog.Info("install fetch → 404", "reason", "install token expired")
		render.Render(w, r, ErrNotFound())
		return
	}

	agent := tok.Agent
	if agent == nil || agent.AgentType == nil || !agent.AgentType.HasInstallTemplates() {
		slog.Info("install fetch → 404", "reason", "agent type has no install templates configured")
		render.Render(w, r, ErrNotFound())
		return
	}

	data := map[string]any{}
	if agent.Configuration != nil {
		data = map[string]any(*agent.Configuration)
	}

	resolved, err := schema.ResolveSecrets(ctx, h.vault, agent.AgentType.ConfigurationSchema, data)
	if err != nil {
		slog.Warn("install fetch → 404", "reason", "vault resolution failed: "+err.Error())
		render.Render(w, r, ErrNotFound())
		return
	}

	body, err := domain.RenderConfigTemplate(agent.AgentType, resolved)
	if err != nil {
		slog.Warn("install fetch → 404", "reason", "config template render failed: "+err.Error())
		render.Render(w, r, ErrNotFound())
		return
	}

	w.Header().Set("Content-Type", agent.AgentType.ConfigContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func (h *AgentInstallTokenHandler) renderInstallToken(
	w http.ResponseWriter,
	r *http.Request,
	tok *domain.AgentInstallToken,
	status int,
) {
	agent := tok.Agent
	url := buildInstallURL(h.publicBaseURL, tok.PlainToken)

	data := map[string]any{}
	if agent.Configuration != nil {
		data = map[string]any(*agent.Configuration)
	}

	cmdText, err := domain.RenderCmdTemplate(agent.AgentType, data, url, tok.PlainBootstrapToken)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	// Emit JSON without HTML-escaping so installCommand stays copy-pasteable:
	// Go's default json.Marshal would write `&&` (and other HTML-significant
	// chars) as `&&`. Decoders parse the escape correctly, but the
	// installCommand field is meant to be eyeballed and pasted into a shell
	// straight from the response, where the literal `&` survives and
	// breaks curl.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(InstallTokenRes{
		InstallCommand: cmdText,
		URL:            url,
		ExpiresAt:      JSONUTCTime(tok.ExpiresAt),
	})
}
