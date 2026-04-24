package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type AgentInstallCommandHandler struct {
	querier       domain.AgentInstallCommandQuerier
	commander     domain.AgentInstallCommandCommander
	agentQuerier  domain.AgentQuerier
	vault         schema.Vault
	publicBaseURL string
}

func NewAgentInstallCommandHandler(
	querier domain.AgentInstallCommandQuerier,
	commander domain.AgentInstallCommandCommander,
	agentQuerier domain.AgentQuerier,
	vault schema.Vault,
	publicBaseURL string,
) *AgentInstallCommandHandler {
	return &AgentInstallCommandHandler{
		querier:       querier,
		commander:     commander,
		agentQuerier:  agentQuerier,
		vault:         vault,
		publicBaseURL: publicBaseURL,
	}
}

// InstallCommandRes is the Create/Regenerate response — the plain token is
// rendered into InstallCommand and URL exactly once and is not recoverable
// afterwards.
type InstallCommandRes struct {
	InstallCommand string      `json:"installCommand"`
	URL            string      `json:"url"`
	ExpiresAt      JSONUTCTime `json:"expiresAt"`
}

// InstallCommandMetaRes is the GET response — metadata only, no token, no URL.
// If the admin lost the token, they must Regenerate.
type InstallCommandMetaRes struct {
	ID        properties.UUID `json:"id"`
	ExpiresAt JSONUTCTime     `json:"expiresAt"`
	CreatedAt JSONUTCTime     `json:"createdAt"`
}

func (h *AgentInstallCommandHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	cmd, err := h.commander.Create(ctx, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	h.renderInstallCommand(w, r, cmd, cmd.PlainToken, http.StatusCreated)
}

func (h *AgentInstallCommandHandler) Regenerate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	cmd, err := h.commander.Regenerate(ctx, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	h.renderInstallCommand(w, r, cmd, cmd.PlainToken, http.StatusOK)
}

// Get returns metadata about the current install command. It never returns the
// plain token nor the rendered install URL — once the token leaves the
// Create/Regenerate response, it cannot be recovered.
func (h *AgentInstallCommandHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	cmd, err := h.querier.GetByAgentID(ctx, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	if cmd.IsExpired() {
		render.Render(w, r, ErrNotFound())
		return
	}

	render.JSON(w, r, InstallCommandMetaRes{
		ID:        cmd.ID,
		ExpiresAt: JSONUTCTime(cmd.ExpiresAt),
		CreatedAt: JSONUTCTime(cmd.CreatedAt),
	})
}

func (h *AgentInstallCommandHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	if err := h.commander.Revoke(ctx, id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Fetch serves the rendered agent configuration to an installer. It is mounted
// behind the standard auth middleware and restricted to participant/agent
// roles; the install token in the URL is the per-resource secret that selects
// which install command to render. Any error or exceptional state yields a
// uniform 404 with an empty body — server-side logs describe the real cause
// for ops debugging. No information leak.
func (h *AgentInstallCommandHandler) Fetch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := chi.URLParam(r, "token")
	if token == "" {
		uniform404(w, "empty token")
		return
	}

	cmd, err := h.querier.FindByHashedToken(ctx, domain.HashTokenValue(token))
	if err != nil {
		uniform404(w, "install command lookup failed: "+err.Error())
		return
	}
	if cmd.IsExpired() {
		uniform404(w, "install command expired")
		return
	}

	agent := cmd.Agent
	if agent == nil || agent.AgentType == nil || agent.AgentType.ConfigTemplate == "" {
		uniform404(w, "agent type has no install templates configured")
		return
	}

	data := map[string]any{}
	if agent.Configuration != nil {
		data = map[string]any(*agent.Configuration)
	}

	resolved, err := domain.ResolveVaultRefs(ctx, h.vault, agent.AgentType.ConfigurationSchema, data)
	if err != nil {
		uniform404(w, "vault resolution failed: "+err.Error())
		return
	}

	body, err := domain.RenderConfigTemplate(agent.AgentType, resolved)
	if err != nil {
		uniform404(w, "config template render failed: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", agent.AgentType.ConfigContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

// uniform404 writes a 404 with an empty body. The reason is logged server-side
// so ops can distinguish unknown / expired / template / vault / render failures
// without leaking anything to the caller.
func uniform404(w http.ResponseWriter, reason string) {
	slog.Info("install fetch → 404", "reason", reason)
	w.WriteHeader(http.StatusNotFound)
}

func (h *AgentInstallCommandHandler) renderInstallCommand(
	w http.ResponseWriter,
	r *http.Request,
	cmd *domain.AgentInstallCommand,
	plain string,
	status int,
) {
	ctx := r.Context()
	agent, err := h.agentQuerier.Get(ctx, cmd.AgentID)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	url := domain.BuildInstallURL(h.publicBaseURL, plain)

	data := map[string]any{}
	if agent.Configuration != nil {
		data = map[string]any(*agent.Configuration)
	}

	cmdText, err := domain.RenderCmdTemplate(agent.AgentType, data, url, cmd.PlainBootstrapToken)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	// Emit JSON without HTML-escaping so installCommand stays copy-pasteable:
	// Go's default json.Marshal would write `&&` as `&&`.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(InstallCommandRes{
		InstallCommand: cmdText,
		URL:            url,
		ExpiresAt:      JSONUTCTime(cmd.ExpiresAt),
	})
}
