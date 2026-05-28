package api

import (
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

const infrastructureInstallConfigFullPath = "/api/v1/infrastructures" + InstallConfigSubPath

// buildInfrastructureInstallURL is the infrastructure-mount twin of
// buildAgentInstallURL. The two paths share InstallConfigSubPath ("/install/
// {token}/config") so the trailing segment matches in both URL surfaces and
// only the entity prefix differs.
func buildInfrastructureInstallURL(publicBaseURL, token string) string {
	return strings.TrimRight(publicBaseURL, "/") + strings.Replace(infrastructureInstallConfigFullPath, "{token}", token, 1)
}

type InfrastructureInstallTokenHandler struct {
	querier                 domain.InstallTokenQuerier
	commander               domain.InstallTokenCommander
	infrastructureAuthScope middlewares.ObjectScopeLoader
	authz                   authz.Authorizer
	vault                   schema.Vault
	publicBaseURL           string
}

func NewInfrastructureInstallTokenHandler(
	querier domain.InstallTokenQuerier,
	commander domain.InstallTokenCommander,
	infrastructureAuthScope middlewares.ObjectScopeLoader,
	authorizer authz.Authorizer,
	vault schema.Vault,
	publicBaseURL string,
) *InfrastructureInstallTokenHandler {
	return &InfrastructureInstallTokenHandler{
		querier:                 querier,
		commander:               commander,
		infrastructureAuthScope: infrastructureAuthScope,
		authz:                   authorizer,
		vault:                   vault,
		publicBaseURL:           publicBaseURL,
	}
}

// Routes mounts under `/infrastructures` alongside InfrastructureHandler.
// Mirrors AgentInstallTokenHandler.Routes — same URL shape under a different
// entity prefix.
func (h *InfrastructureInstallTokenHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		r.With(
			middlewares.MustHaveRoles(auth.RoleAdmin, auth.RoleParticipant, auth.RoleAgent),
		).Get(InstallConfigSubPath, h.Fetch)

		r.Group(func(r chi.Router) {
			r.Use(middlewares.ID)

			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeInfrastructure, authz.ActionRead, h.authz, h.infrastructureAuthScope),
			).Get("/{id}/install-command", h.Get)
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeInfrastructure, authz.ActionUpdate, h.authz, h.infrastructureAuthScope),
			).Post("/{id}/install-command", h.Create)
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeInfrastructure, authz.ActionUpdate, h.authz, h.infrastructureAuthScope),
			).Post("/{id}/install-command/regenerate", h.Regenerate)
			r.With(
				middlewares.AuthzFromID(authz.ObjectTypeInfrastructure, authz.ActionUpdate, h.authz, h.infrastructureAuthScope),
			).Delete("/{id}/install-command", h.Revoke)
		})
	}
}

func (h *InfrastructureInstallTokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	tok, err := h.commander.Create(ctx, domain.InstallTokenEntityTypeInfrastructure, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	h.renderInstallToken(w, r, tok, http.StatusCreated)
}

func (h *InfrastructureInstallTokenHandler) Regenerate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	tok, err := h.commander.Regenerate(ctx, domain.InstallTokenEntityTypeInfrastructure, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	h.renderInstallToken(w, r, tok, http.StatusOK)
}

func (h *InfrastructureInstallTokenHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	tok, err := h.querier.GetByEntity(ctx, domain.InstallTokenEntityTypeInfrastructure, id)
	if err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	render.JSON(w, r, InstallTokenMetaRes{
		ID:        tok.ID,
		ExpiresAt: JSONUTCTime(tok.ExpiresAt),
		CreatedAt: JSONUTCTime(tok.CreatedAt),
	})
}

func (h *InfrastructureInstallTokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := middlewares.MustGetID(ctx)

	if err := h.commander.Revoke(ctx, domain.InstallTokenEntityTypeInfrastructure, id); err != nil {
		render.Render(w, r, ErrDomain(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Fetch serves the rendered infrastructure configuration to an installer.
// Mirrors AgentInstallTokenHandler.Fetch — same auth gate, same log-only-
// reason 404 policy, same secret-resolution + template-render pipeline. The
// entity-type mismatch branch is what keeps the two install URL surfaces
// strictly separated.
func (h *InfrastructureInstallTokenHandler) Fetch(w http.ResponseWriter, r *http.Request) {
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
	if tok.EntityType != domain.InstallTokenEntityTypeInfrastructure {
		slog.Info("install fetch → 404", "reason", "install token entity type mismatch", "got", string(tok.EntityType))
		render.Render(w, r, ErrNotFound())
		return
	}
	if tok.IsExpired() {
		slog.Info("install fetch → 404", "reason", "install token expired")
		render.Render(w, r, ErrNotFound())
		return
	}

	infra := tok.Infrastructure
	if infra == nil || infra.InfrastructureType == nil || !infra.InfrastructureType.HasInstallTemplates() {
		slog.Info("install fetch → 404", "reason", "infrastructure type has no install templates configured")
		render.Render(w, r, ErrNotFound())
		return
	}

	data := map[string]any{}
	if infra.Configuration != nil {
		data = map[string]any(*infra.Configuration)
	}

	resolved, err := schema.ResolveSecrets(ctx, h.vault, infra.InfrastructureType.ConfigurationSchema, data)
	if err != nil {
		slog.Warn("install fetch → 404", "reason", "vault resolution failed: "+err.Error())
		render.Render(w, r, ErrNotFound())
		return
	}

	body, err := infra.InfrastructureType.RenderConfigTemplate(resolved)
	if err != nil {
		slog.Warn("install fetch → 404", "reason", "config template render failed: "+err.Error())
		render.Render(w, r, ErrNotFound())
		return
	}

	w.Header().Set("Content-Type", infra.InfrastructureType.ConfigContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func (h *InfrastructureInstallTokenHandler) renderInstallToken(
	w http.ResponseWriter,
	r *http.Request,
	tok *domain.InstallToken,
	status int,
) {
	infra := tok.Infrastructure
	url := buildInfrastructureInstallURL(h.publicBaseURL, tok.PlainToken)

	data := map[string]any{}
	if infra.Configuration != nil {
		data = map[string]any(*infra.Configuration)
	}

	cmdText, err := infra.InfrastructureType.RenderCmdTemplate(data, url, tok.PlainBootstrapToken)
	if err != nil {
		render.Render(w, r, ErrInternal(err))
		return
	}

	writeInstallTokenJSON(w, status, InstallTokenRes{
		InstallCommand: cmdText,
		URL:            url,
		ExpiresAt:      JSONUTCTime(tok.ExpiresAt),
	})
}
