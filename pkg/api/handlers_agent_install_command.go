package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/schema"
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

type InstallCommandRes struct {
	InstallCommand string      `json:"installCommand"`
	URL            string      `json:"url"`
	ExpiresAt      JSONUTCTime `json:"expiresAt"`
}

func (h *AgentInstallCommandHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	cmd, err := h.commander.Create(ctx, id)
	if err != nil {
		if errors.As(err, &domain.ConflictError{}) {
			render.Render(w, r, ErrConflictWithCode("install_command_exists", err.Error()))
			return
		}
		if errors.As(err, &domain.InvalidInputError{}) {
			render.Render(w, r, ErrUnprocessableWithCode("install_not_configured", err.Error()))
			return
		}
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
		if errors.As(err, &domain.NotFoundError{}) {
			render.Render(w, r, ErrNotFoundWithCode("install_command_not_found"))
			return
		}
		if errors.As(err, &domain.InvalidInputError{}) {
			render.Render(w, r, ErrUnprocessableWithCode("install_not_configured", err.Error()))
			return
		}
		render.Render(w, r, ErrDomain(err))
		return
	}

	h.renderInstallCommand(w, r, cmd, cmd.PlainToken, http.StatusOK)
}

func (h *AgentInstallCommandHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	cmd, err := h.querier.GetByAgentID(ctx, id)
	if err != nil {
		if errors.As(err, &domain.NotFoundError{}) {
			render.Render(w, r, ErrNotFoundWithCode("install_command_not_found"))
			return
		}
		render.Render(w, r, ErrDomain(err))
		return
	}
	if cmd.IsExpired() {
		render.Render(w, r, ErrNotFoundWithCode("install_command_expired"))
		return
	}

	raw, err := h.vault.Get(ctx, cmd.VaultKey)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}
	plain, ok := raw.(string)
	if !ok {
		render.Render(w, r, ErrInternal(fmt.Errorf("vault returned non-string for install token")))
		return
	}

	h.renderInstallCommand(w, r, cmd, plain, http.StatusOK)
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
	if agent.AgentType == nil || agent.AgentType.CmdTemplate == "" {
		render.Render(w, r, ErrUnprocessableWithCode(
			"install_not_configured",
			"agent type has no install templates configured",
		))
		return
	}

	url := domain.BuildInstallURL(h.publicBaseURL, plain)

	data := map[string]any{}
	if agent.Configuration != nil {
		data = map[string]any(*agent.Configuration)
	}

	cmdText, err := domain.RenderCmdTemplate(agent.AgentType, data, url)
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
