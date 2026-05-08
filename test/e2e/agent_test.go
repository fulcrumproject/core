//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func testAgent(t *testing.T, env *Env) {
	t.Run("admin lists agents includes seed", func(t *testing.T) {
		page := testhelpers.MustList[api.AgentRes](t, env.AdminClient, "/agents")
		require.GreaterOrEqual(t, page.TotalItems, int64(1))
	})

	t.Run("create+get+delete agent", func(t *testing.T) {
		name := "agent-" + testhelpers.Uniq()
		created := testhelpers.MustPost[api.CreateAgentReq, api.AgentRes](t, env.AdminClient, "/agents", api.CreateAgentReq{
			Name:        name,
			ProviderID:  env.Seed.Provider.ID,
			AgentTypeID: env.Seed.AgentType.ID,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, env.Seed.Provider.ID, created.ProviderID)
		require.Equal(t, env.Seed.AgentType.ID, created.AgentTypeID)
		require.NotEmpty(t, created.Status, "Status populated on create")
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := testhelpers.MustGet[api.AgentRes](t, env.AdminClient, "/agents", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.ProviderID, got.ProviderID)
		require.Equal(t, created.AgentTypeID, got.AgentTypeID)
		require.Equal(t, created.Status, got.Status)

		page := testhelpers.MustList[api.AgentRes](t, env.AdminClient, "/agents")
		require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created agent")

		testhelpers.MustDelete(t, env.AdminClient, "/agents", created.ID)
		testhelpers.AssertGone(t, env.AdminClient, "/agents", created.ID)
	})

	t.Run("install-command GET returns metadata even when expired (regression #208)", func(t *testing.T) {
		// Spin up a fresh agent so we don't perturb the seed agent's install
		// token (the smoke /me subtest already used it).
		ag := testhelpers.MustPost[api.CreateAgentReq, api.AgentRes](t, env.AdminClient, "/agents", api.CreateAgentReq{
			Name:        "agent-it-" + testhelpers.Uniq(),
			ProviderID:  env.Seed.Provider.ID,
			AgentTypeID: env.Seed.AgentType.ID,
		})
		t.Cleanup(func() { testhelpers.MustDelete(t, env.AdminClient, "/agents", ag.ID) })

		issued := mustCreateInstallToken(t, env.AdminClient, ag.ID)
		require.NotEmpty(t, issued.InstallCommand)
		require.NotEmpty(t, issued.URL)
		require.True(t, time.Time(issued.ExpiresAt).After(time.Now()), "fresh install token must expire in the future")

		// Expire the token directly in the DB. The HTTP API doesn't expose a
		// way to backdate the expiry on demand.
		require.NoError(t, env.DB.Model(&domain.AgentInstallToken{}).
			Where("agent_id = ?", ag.ID).
			Update("expires_at", time.Now().Add(-1*time.Hour)).Error)

		var meta api.InstallTokenMetaRes
		resp, err := env.AdminClient.R().
			SetPathParam("id", ag.ID.String()).
			SetResult(&meta).
			Get("/agents/{id}/install-command")
		require.NoError(t, err)
		require.Equalf(t, http.StatusOK, resp.StatusCode(), "GET install-command must return metadata even after expiry: %s", resp.String())
		require.True(t, time.Time(meta.ExpiresAt).Before(time.Now()), "expiresAt should reflect the backdated value")
	})
}
