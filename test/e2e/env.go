//go:build e2e

package e2e

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/database"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"resty.dev/v3"
)

type Env struct {
	ServerURL string
	HealthURL string
	DB        *gorm.DB
	Seed      *Fixtures

	AdminClient       *resty.Client
	ParticipantClient *resty.Client
	ConsumerClient    *resty.Client
	AgentClient       *resty.Client
}

func newEnv(t *testing.T, tdb *database.TestDB, serverURL, healthURL string, seed *Fixtures) *Env {
	t.Helper()
	return &Env{
		ServerURL:         serverURL,
		HealthURL:         healthURL,
		DB:                tdb.DB,
		Seed:              seed,
		AdminClient:       roleClient(t, serverURL, "admin1"),
		ParticipantClient: roleClient(t, serverURL, "participant1"),
		ConsumerClient:    roleClient(t, serverURL, "consumer1"),
		AgentClient:       roleClient(t, serverURL, "agent1"),
	}
}

func roleClient(t *testing.T, serverURL, username string) *resty.Client {
	t.Helper()
	tok, err := GetToken(username, "password")
	require.NoErrorf(t, err, "keycloak token for %s", username)
	return NewClient(serverURL, tok)
}
