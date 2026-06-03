//go:build e2e

package e2e

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/database"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"resty.dev/v3"
)

type Env struct {
	ServerURL string
	HealthURL string
	DB        *gorm.DB
	Seed      *testhelpers.CoreFixtures

	AdminClient    *resty.Client
	ProviderClient *resty.Client
	ConsumerClient *resty.Client
	AgentClient    *resty.Client
}

func newEnv(t *testing.T, tdb *database.TestDB, serverURL, healthURL string, seed *testhelpers.CoreFixtures) *Env {
	t.Helper()
	return &Env{
		ServerURL:      serverURL,
		HealthURL:      healthURL,
		DB:             tdb.DB,
		Seed:           seed,
		AdminClient:    roleClient(t, serverURL, "admin1"),
		ProviderClient: roleClient(t, serverURL, "participant1"),
		ConsumerClient: roleClient(t, serverURL, "consumer1"),
		AgentClient:    roleClient(t, serverURL, "agent1"),
	}
}

func roleClient(t *testing.T, serverURL, username string) *resty.Client {
	t.Helper()
	tok, err := coreRealm.GetToken(username, "password")
	require.NoErrorf(t, err, "keycloak token for %s", username)
	return testhelpers.NewClient(serverURL, tok)
}

func TestE2E(t *testing.T) {
	tdb := database.NewTestDB(t)
	t.Cleanup(func() { tdb.Cleanup(t) })

	serverURL, healthURL := startServer(t, tdb)

	var seed *testhelpers.CoreFixtures
	require.NoError(t, tdb.DB.Transaction(func(tx *gorm.DB) error {
		s, err := testhelpers.SeedCore(tx)
		if err != nil {
			return err
		}
		seed = s
		return nil
	}))

	env := newEnv(t, tdb, serverURL, healthURL, seed)

	t.Run("participants", func(t *testing.T) { testParticipant(t, env) })
	t.Run("tokens", func(t *testing.T) { testToken(t, env) })
	t.Run("agents", func(t *testing.T) { testAgent(t, env) })
	t.Run("agent types", func(t *testing.T) { testAgentType(t, env) })
	t.Run("infrastructure types", func(t *testing.T) { testInfrastructureType(t, env) })
	t.Run("infrastructures", func(t *testing.T) { testInfrastructure(t, env) })
	t.Run("service types", func(t *testing.T) { testServiceType(t, env) })
	t.Run("service option types", func(t *testing.T) { testServiceOptionType(t, env) })
	t.Run("service options", func(t *testing.T) { testServiceOption(t, env) })
	t.Run("config pools", func(t *testing.T) { testConfigPool(t, env) })
	t.Run("config pool values", func(t *testing.T) { testConfigPoolValue(t, env) })
	t.Run("service pool sets", func(t *testing.T) { testServicePoolSet(t, env) })
	t.Run("service pools", func(t *testing.T) { testServicePool(t, env) })
	t.Run("service pool values", func(t *testing.T) { testServicePoolValue(t, env) })
	t.Run("service groups", func(t *testing.T) { testServiceGroup(t, env) })
	t.Run("services", func(t *testing.T) { testService(t, env) })
	t.Run("jobs", func(t *testing.T) { testJob(t, env) })
	t.Run("scenario: agent install lifecycle", func(t *testing.T) { testAgentInstallScenario(t, env) })
	t.Run("scenario: infrastructure install lifecycle", func(t *testing.T) { testInfrastructureInstallScenario(t, env) })
	t.Run("scenario: fae proxmox pool allocation", func(t *testing.T) { testFaeProxmoxScenario(t, env) })
	t.Run("scenario: service lifecycle", func(t *testing.T) { testServiceLifecycleScenario(t, env) })
	t.Run("scenario: service create failure", func(t *testing.T) { testServiceCreateFailureScenario(t, env) })
	t.Run("events", func(t *testing.T) { testEvent(t, env) })
	t.Run("metric types", func(t *testing.T) { testMetricType(t, env) })
	t.Run("metric entries", func(t *testing.T) { testMetricEntry(t, env) })
	t.Run("vault", func(t *testing.T) { testVault(t, env) })
	t.Run("keycloak users", func(t *testing.T) { testKeycloakUser(t, env) })
	t.Run("health", func(t *testing.T) { testHealth(t, env) })
}
