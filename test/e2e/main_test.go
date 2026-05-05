//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/database"
	"github.com/stretchr/testify/require"
)

func TestE2E(t *testing.T) {
	tdb := database.NewTestDB(t)
	t.Cleanup(func() { tdb.Cleanup(t) })

	serverURL := startServer(t, tdb)
	seed := mustSeed(t, tdb.DB)
	env := newEnv(t, tdb, serverURL, seed)

	t.Run("smoke/admin lists participants", func(t *testing.T) {
		var body api.PageRes[api.ParticipantRes]
		resp, err := env.AdminClient.R().SetResult(&body).Get("/participants")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.GreaterOrEqual(t, body.TotalItems, int64(2), "expected provider1 + consumer1 from seed")
	})

	t.Run("smoke/agent token authenticates", func(t *testing.T) {
		var body api.AgentRes
		resp, err := env.AgentClient.R().SetResult(&body).Get("/agents/me")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.String())
		require.Equal(t, env.Seed.Agent.ID, body.ID)
	})

	t.Run("participants", func(t *testing.T) { testParticipant(t, env) })
	t.Run("tokens", func(t *testing.T) { testToken(t, env) })
	t.Run("agents", func(t *testing.T) { testAgent(t, env) })
	t.Run("agent types", func(t *testing.T) { testAgentType(t, env) })
	t.Run("service types", func(t *testing.T) { testServiceType(t, env) })
	t.Run("service option types", func(t *testing.T) { testServiceOptionType(t, env) })
	t.Run("service options", func(t *testing.T) { testServiceOption(t, env) })
	t.Run("agent pools", func(t *testing.T) { testAgentPool(t, env) })
	t.Run("agent pool values", func(t *testing.T) { testAgentPoolValue(t, env) })
	t.Run("service pool sets", func(t *testing.T) { testServicePoolSet(t, env) })
	t.Run("service pools", func(t *testing.T) { testServicePool(t, env) })
	t.Run("service pool values", func(t *testing.T) { testServicePoolValue(t, env) })
	t.Run("service groups", func(t *testing.T) { testServiceGroup(t, env) })
	t.Run("services", func(t *testing.T) { testService(t, env) })
}
