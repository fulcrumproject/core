package api

import (
	"log/slog"
	"net/http"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/chi/v5"
)

// AgentInstallPublicHandler serves the unauthenticated install URL for target hosts.
// Any error or exceptional state yields a uniform 404 with an empty body — server-side
// logs describe the real cause for ops debugging. No information leak.
type AgentInstallPublicHandler struct {
	querier domain.AgentInstallCommandQuerier
	vault   schema.Vault
}

func NewAgentInstallPublicHandler(
	querier domain.AgentInstallCommandQuerier,
	vault schema.Vault,
) *AgentInstallPublicHandler {
	return &AgentInstallPublicHandler{
		querier: querier,
		vault:   vault,
	}
}

func (h *AgentInstallPublicHandler) Fetch(w http.ResponseWriter, r *http.Request) {
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
	slog.Info("public install fetch → 404", "reason", reason)
	w.WriteHeader(http.StatusNotFound)
}
