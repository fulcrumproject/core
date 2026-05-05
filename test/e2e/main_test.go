//go:build e2e

package e2e

import (
	"fmt"
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

func NewClient(serverURL, authToken string) *resty.Client {
	return resty.New().
		SetBaseURL(serverURL + "/api/v1").
		SetAuthToken(authToken)
}

func GetToken(username, password string) (string, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", keycloakURL, keycloakRealm)
	var out struct {
		AccessToken string `json:"access_token"`
	}
	resp, err := resty.New().R().SetFormData(map[string]string{
		"grant_type":    "password",
		"client_id":     oauthClientID,
		"client_secret": oauthSecret,
		"username":      username,
		"password":      password,
	}).SetResult(&out).Post(tokenURL)

	if err != nil {
		return "", fmt.Errorf("keycloak token: %w", err)
	}
	if resp.IsError() {
		return "", fmt.Errorf("keycloak token: %s", resp.String())
	}

	return out.AccessToken, nil
}

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
