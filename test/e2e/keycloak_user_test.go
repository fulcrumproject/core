//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/stretchr/testify/require"
)

func testKeycloakUser(t *testing.T, env *Env) {
	t.Run("admin lists keycloak users includes seeded usernames", func(t *testing.T) {
		page := mustList[api.KeycloakUserListItemRes](t, env.AdminClient, "/keycloak-users")
		require.GreaterOrEqual(t, page.TotalItems, int64(4), "admin1 + participant1 + consumer1 + agent1")
	})

	t.Run("admin creates+gets+updates+deletes keycloak user", func(t *testing.T) {
		username := "e2e-" + uniq()
		created := mustPost[api.CreateKeycloakUserReq, api.KeycloakUserRes](t, env.AdminClient, "/keycloak-users", api.CreateKeycloakUserReq{
			Username:      username,
			Email:         username + "@example.com",
			EmailVerified: true,
			FirstName:     "E2E",
			LastName:      "User",
			Password:      "password",
			Enabled:       true,
			Role:          auth.RoleParticipant,
			ParticipantID: env.Seed.Provider.ID.String(),
		})
		require.Equal(t, username, created.Username)
		t.Cleanup(func() {
			// Best-effort delete; any leftover doesn't break later runs since
			// usernames carry a unique suffix.
			_, _ = env.AdminClient.R().
				SetPathParam("id", created.ID).
				Delete("/keycloak-users/{id}")
		})

		var got api.KeycloakUserRes
		resp, err := env.AdminClient.R().
			SetPathParam("id", created.ID).
			SetResult(&got).
			Get("/keycloak-users/{id}")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.Equal(t, username, got.Username)

		newFirst := "Updated"
		var updated api.KeycloakUserRes
		resp, err = env.AdminClient.R().
			SetPathParam("id", created.ID).
			SetBody(api.UpdateKeycloakUserReq{FirstName: &newFirst}).
			SetResult(&updated).
			Patch("/keycloak-users/{id}")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.Equal(t, newFirst, updated.FirstName)
	})

	t.Run("participant cannot list keycloak users", func(t *testing.T) {
		resp, err := env.ParticipantClient.R().Get("/keycloak-users")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
