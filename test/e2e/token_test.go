//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/require"
)

func testToken(t *testing.T, env *Env) {
	t.Run("admin creates+gets+regenerates+deletes", func(t *testing.T) {
		expire := time.Now().Add(24 * time.Hour)
		name := "tok-" + uniq()
		created := mustPost[api.CreateTokenReq, api.TokenRes](t, env.AdminClient, "/tokens", api.CreateTokenReq{
			Name:     name,
			Role:     auth.RoleAdmin,
			ExpireAt: &expire,
		})
		require.NotEmpty(t, created.Value, "create response must include plaintext token")
		require.Equal(t, name, created.Name)
		require.Equal(t, auth.RoleAdmin, created.Role)
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := mustGet[api.TokenRes](t, env.AdminClient, "/tokens", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.Role, got.Role)
		require.Empty(t, got.Value, "GET response must NOT echo the plaintext token")

		page := mustList[api.TokenRes](t, env.AdminClient, "/tokens")
		require.True(t, containsID(page.Items, created.ID), "list must include just-created token")

		var regen api.TokenRes
		resp, err := env.AdminClient.R().
			SetPathParam("id", created.ID.String()).
			SetResult(&regen).
			Post("/tokens/{id}/regenerate")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.NotEmpty(t, regen.Value, "regenerate must return new plaintext token")
		require.NotEqual(t, created.Value, regen.Value)
		require.Equal(t, created.ID, regen.ID, "regenerate must not rotate the row ID")
		require.Equal(t, created.Role, regen.Role, "regenerate must not change role")

		// After regeneration, GET must still suppress the plaintext value.
		gotAfter := mustGet[api.TokenRes](t, env.AdminClient, "/tokens", created.ID)
		require.Empty(t, gotAfter.Value, "GET after regenerate must NOT echo plaintext")

		mustDelete(t, env.AdminClient, "/tokens", created.ID)
		assertGone(t, env.AdminClient, "/tokens", created.ID)
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
