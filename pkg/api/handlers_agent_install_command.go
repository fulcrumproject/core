package api

import (
	"encoding/json"
	"net/http"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/render"
)

type AgentInstallCommandHandler struct {
	querier       domain.AgentInstallCommandQuerier
	commander     domain.AgentInstallCommandCommander
	agentQuerier  domain.AgentQuerier
	publicBaseURL string
}

func NewAgentInstallCommandHandler(
	querier domain.AgentInstallCommandQuerier,
	commander domain.AgentInstallCommandCommander,
	agentQuerier domain.AgentQuerier,
	publicBaseURL string,
) *AgentInstallCommandHandler {
	return &AgentInstallCommandHandler{
		querier:       querier,
		commander:     commander,
		agentQuerier:  agentQuerier,
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
