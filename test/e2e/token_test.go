//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/stretchr/testify/require"
)

func testToken(t *testing.T, env *Env) {
	t.Run("admin creates+gets+regenerates+deletes", func(t *testing.T) {
		expire := time.Now().Add(24 * time.Hour)
		created := mustPost[api.CreateTokenReq, api.TokenRes](t, env.AdminClient, "/tokens", api.CreateTokenReq{
			Name:     "tok-" + uniq(),
			Role:     auth.RoleAdmin,
			ExpireAt: &expire,
		})
		require.NotEmpty(t, created.Value, "create response must include plaintext token")
		require.Equal(t, auth.RoleAdmin, created.Role)

		got := mustGet[api.TokenRes](t, env.AdminClient, "/tokens", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Empty(t, got.Value, "GET response must NOT echo the plaintext token")

		var regen api.TokenRes
		resp, err := env.AdminClient.R().
			SetPathParam("id", created.ID.String()).
			SetResult(&regen).
			Post("/tokens/{id}/regenerate")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.NotEmpty(t, regen.Value, "regenerate must return new plaintext token")
		require.NotEqual(t, created.Value, regen.Value)

		mustDelete(t, env.AdminClient, "/tokens", created.ID)
	})

	t.Run("participant cannot create admin token", func(t *testing.T) {
		resp, err := env.ParticipantClient.R().
			SetBody(api.CreateTokenReq{
				Name: "esc-" + uniq(),
				Role: auth.RoleAdmin,
			}).
			Post("/tokens")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("participant cannot create token scoped to another participant", func(t *testing.T) {
		// participant1 owns Provider; trying to mint a participant token for
		// the Consumer must 403.
		resp, err := env.ParticipantClient.R().
			SetBody(api.CreateTokenReq{
				Name:    "x-scope-" + uniq(),
				Role:    auth.RoleParticipant,
				ScopeID: &env.Seed.Consumer.ID,
			}).
			Post("/tokens")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
