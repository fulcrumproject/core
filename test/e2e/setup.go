//go:build e2e

package e2e

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/database"
	"github.com/fulcrumproject/core/pkg/testhelpers"
)

const vaultKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

var coreRealm = testhelpers.Realm{
	URL:      "http://localhost:8080",
	Name:     "fulcrum",
	ClientID: "fulcrum-api",
	Secret:   "secret",
}

func startServer(t *testing.T, tdb *database.TestDB) (apiURL, healthURL string) {
	t.Helper()
	dsn := tdb.DSN()
	return testhelpers.Start(t, testhelpers.Config{
		BinaryPath: "./cmd/fulcrum",
		DB:         tdb,
		Realm:      coreRealm,
		ExtraEnv: []string{
			"FULCRUM_METRIC_DB_DSN=" + dsn,
			"FULCRUM_SCHEDULER_LOCKER_DB_DSN=" + dsn,
			"FULCRUM_VAULT_ENCRYPTION_KEY=" + vaultKey,
			"FULCRUM_KEYCLOAK_ADMIN=true",
		},
		Cover: true,
	})
}
