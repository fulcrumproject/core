//go:build e2e

package e2e

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/database"
)

func TestE2E(t *testing.T) {
	tdb := database.NewTestDB(t)
	t.Cleanup(func() { tdb.Cleanup(t) })

	serverURL, healthURL := startServer(t, tdb)
	seed := mustSeed(t, tdb.DB)
	env := newEnv(t, tdb, serverURL, healthURL, seed)

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
	t.Run("jobs", func(t *testing.T) { testJob(t, env) })
	t.Run("events", func(t *testing.T) { testEvent(t, env) })
	t.Run("metric types", func(t *testing.T) { testMetricType(t, env) })
	t.Run("metric entries", func(t *testing.T) { testMetricEntry(t, env) })
	t.Run("vault", func(t *testing.T) { testVault(t, env) })
	t.Run("keycloak users", func(t *testing.T) { testKeycloakUser(t, env) })
	t.Run("health", func(t *testing.T) { testHealth(t, env) })
}
