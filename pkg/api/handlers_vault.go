// Vault handlers for secret resolution
package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// VaultHandler handles vault secret resolution endpoints
type VaultHandler struct {
	vault schema.Vault
}

// NewVaultHandler creates a new vault handler
func NewVaultHandler(vault schema.Vault) *VaultHandler {
	return &VaultHandler{
		vault: vault,
	}
}

// Routes returns the router configuration function with all vault routes registered
func (h *VaultHandler) Routes() func(r chi.Router) {
	return func(r chi.Router) {
		// Only agents can resolve secrets
		r.Use(middlewares.MustHaveRoles(auth.RoleAgent))

		r.Get("/{reference}", h.GetSecret)
	}
}

// GetSecretRes represents the response for secret retrieval
type GetSecretRes struct {
	Value any `json:"value"`
}

// GetSecret retrieves a secret by its reference
// Only accessible by authenticated agents
func (h *VaultHandler) GetSecret(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reference := chi.URLParam(r, "reference")

	if reference == "" {
		render.Render(w, r, ErrInvalidRequest(fmt.Errorf("reference parameter is required")))
		return
	}

	// Retrieve secret from vault
	value, err := h.vault.Get(ctx, reference)
	if err != nil {
		slog.Error("Failed to retrieve secret", "reference", reference, "error", err)
		render.Render(w, r, ErrNotFound())
		return
	}

	// Return the secret value
	render.Status(r, http.StatusOK)
	render.JSON(w, r, GetSecretRes{Value: value})
}

