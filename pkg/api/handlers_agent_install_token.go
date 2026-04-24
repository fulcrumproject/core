package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type AgentInstallTokenHandler struct {
	querier       domain.AgentInstallTokenQuerier
	commander     domain.AgentInstallTokenCommander
	vault         schema.Vault
	publicBaseURL string
}

func NewAgentInstallTokenHandler(
	querier domain.AgentInstallTokenQuerier,
	commander domain.AgentInstallTokenCommander,
	vault schema.Vault,
	publicBaseURL string,
) *AgentInstallTokenHandler {
	return &AgentInstallTokenHandler{
		querier:       querier,
		commander:     commander,
		vault:         vault,
		publicBaseURL: publicBaseURL,
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
	ID        properties.UUID `json:"id"`
	ExpiresAt JSONUTCTime     `json:"expiresAt"`
	CreatedAt JSONUTCTime     `json:"createdAt"`
}

func (h *AgentInstallTokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	tok, err := h.commander.Create(ctx, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	h.renderInstallToken(w, r, tok, tok.PlainToken, http.StatusCreated)
}

func (h *AgentInstallTokenHandler) Regenerate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	tok, err := h.commander.Regenerate(ctx, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}

	h.renderInstallToken(w, r, tok, tok.PlainToken, http.StatusOK)
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
		ID:        tok.ID,
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

// Fetch serves the rendered agent configuration to an installer. It is mounted
// behind the standard auth middleware and restricted to participant/agent
// roles; the install token in the URL is the per-resource secret that selects
// which install record to render. Any error or exceptional state yields a
// uniform 404 with an empty body — server-side logs describe the real cause
// for ops debugging. No information leak.
func (h *AgentInstallTokenHandler) Fetch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := chi.URLParam(r, "token")
	if token == "" {
		uniform404(w, slog.LevelInfo, "empty token")
		return
	}

	tok, err := h.querier.FindByHashedToken(ctx, domain.HashTokenValue(token))
	if err != nil {
		if errors.As(err, &domain.NotFoundError{}) {
			uniform404(w, slog.LevelInfo, "install token not found")
		} else {
			uniform404(w, slog.LevelWarn, "install token lookup failed: "+err.Error())
		}
		return
	}
	if tok.IsExpired() {
		uniform404(w, slog.LevelInfo, "install token expired")
		return
	}

	agent := tok.Agent
	if agent == nil || agent.AgentType == nil || agent.AgentType.ConfigTemplate == "" {
		uniform404(w, slog.LevelInfo, "agent type has no install templates configured")
		return
	}

	data := map[string]any{}
	if agent.Configuration != nil {
		data = map[string]any(*agent.Configuration)
	}

	resolved, err := domain.ResolveVaultRefs(ctx, h.vault, agent.AgentType.ConfigurationSchema, data)
	if err != nil {
		uniform404(w, slog.LevelWarn, "vault resolution failed: "+err.Error())
		return
	}

	body, err := domain.RenderConfigTemplate(agent.AgentType, resolved)
	if err != nil {
		uniform404(w, slog.LevelWarn, "config template render failed: "+err.Error())
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
// without leaking anything to the caller. Use Info for client-caused misses
// (unknown token, expired) and Warn for server-side failures (db, vault,
// template) so monitoring can alert on the latter.
func uniform404(w http.ResponseWriter, level slog.Level, reason string) {
	const msg = "install fetch → 404"
	switch level {
	case slog.LevelWarn:
		slog.Warn(msg, "reason", reason)
	default:
		slog.Info(msg, "reason", reason)
	}
	w.WriteHeader(http.StatusNotFound)
}

func (h *AgentInstallTokenHandler) renderInstallToken(
	w http.ResponseWriter,
	r *http.Request,
	tok *domain.AgentInstallToken,
	plain string,
	status int,
) {
	agent := tok.Agent
	url := domain.BuildInstallURL(h.publicBaseURL, plain)

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
	// Go's default json.Marshal would write `&&` as `&&`.
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
