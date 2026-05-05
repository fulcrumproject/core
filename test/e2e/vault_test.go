//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func testVault(t *testing.T, env *Env) {
	t.Run("admin cannot read vault (agent-only route)", func(t *testing.T) {
		resp, err := env.AdminClient.R().
			SetPathParam("reference", "any-ref").
			Get("/vault/secrets/{reference}")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("agent gets 404 for unknown reference", func(t *testing.T) {
		resp, err := env.AgentClient.R().
			SetPathParam("reference", "no-such-secret-"+uniq()).
			Get("/vault/secrets/{reference}")
		require.NoError(t, err)
		require.Equalf(t, http.StatusNotFound, resp.StatusCode(), "body: %s", resp.String())
	})
}
