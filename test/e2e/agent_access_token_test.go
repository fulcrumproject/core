// Tests application JWT issuance through a real Core server and shared public JWKS.
// All policy, signing material, credentials and database rows are synthetic fixtures.

//go:build e2e

package e2e

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/database"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/fulcrumproject/core/pkg/sso"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"resty.dev/v3"
)

func applicationSigningConfig(t *testing.T) sso.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	return sso.Config{SigningKey: string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})), KeyID: "application-test", Issuer: "https://core.example"}
}

func applicationConfigFile(t *testing.T, signing sso.Config, policy sso.ApplicationConfig) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{"cmpSso": signing, "applicationTokens": policy})
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "core.json")
	require.NoError(t, os.WriteFile(path, data, 0600))
	return path
}

func TestApplicationInvalidBoot(t *testing.T) {
	signing := applicationSigningConfig(t)
	id := testhelpers.AgentID.String()
	for name, policy := range map[string]sso.ApplicationConfig{
		"invalid agent ID":       {Grants: map[string]sso.ApplicationGrant{"invalid": {Audience: "fae", Roles: []string{"pve"}}}},
		"missing audience":       {Grants: map[string]sso.ApplicationGrant{id: {Roles: []string{"pve"}}}},
		"missing roles":          {Grants: map[string]sso.ApplicationGrant{id: {Audience: "fae"}}},
		"negative TTL":           {TokenTTL: -time.Second},
		"excessive TTL":          {TokenTTL: time.Hour},
		"policy without signing": {Grants: map[string]sso.ApplicationGrant{id: {Audience: "fae", Roles: []string{"pve"}}}},
	} {
		t.Run(name, func(t *testing.T) {
			keyCfg := signing
			if name == "policy without signing" {
				keyCfg = sso.Config{}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "go", "run", "./cmd/fulcrum", "-config", applicationConfigFile(t, keyCfg, policy))
			cmd.Dir = testhelpers.RepoRoot(t)
			cmd.Env = append(os.Environ(), "FULCRUM_DB_DSN=host="+t.TempDir()+" user=synthetic dbname=unavailable", "FULCRUM_PUBLIC_BASE_URL=http://127.0.0.1:3000", "FULCRUM_CMP_SSO_SIGNING_KEY=", "FULCRUM_CMP_SSO_KEY_ID=", "FULCRUM_CMP_SSO_ISSUER=", "FULCRUM_APPLICATION_TOKEN_TTL=")
			out, err := cmd.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(out), "Invalid application token configuration")
			require.NotContains(t, string(out), "Failed to initialize database")
			require.NotContains(t, string(out), "PRIVATE KEY")
		})
	}
}

func TestAgentApplicationTokens(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		t.Run(name, func(t *testing.T) {
			tdb := database.NewTestDB(t)
			t.Cleanup(func() { tdb.Cleanup(t) })
			var seed *testhelpers.CoreFixtures
			require.NoError(t, tdb.DB.Transaction(func(tx *gorm.DB) error { var err error; seed, err = testhelpers.SeedCore(tx); return err }))
			// The shared fixture omits this timestamp; agent PATCH validation requires it.
			require.NoError(t, tdb.DB.Session(&gorm.Session{NewDB: true}).Model(seed.Agent).Update("last_status_update", time.Now()).Error)
			// Opaque agent authentication expects both scopes; the shared seed omits provider.
			require.NoError(t, tdb.DB.Session(&gorm.Session{NewDB: true}).Model(seed.AgentToken).Update("participant_id", seed.Provider.ID).Error)
			// Allow provider-owned configuration fields to prove they are not issuance policy.
			seed.AgentType.ConfigurationSchema.Properties["roles"] = schema.PropertyDefinition{Type: "json"}
			seed.AgentType.ConfigurationSchema.Properties["audience"] = schema.PropertyDefinition{Type: "string"}
			require.NoError(t, tdb.DB.Session(&gorm.Session{NewDB: true}).Save(seed.AgentType).Error)
			signing, policy := sso.Config{}, sso.ApplicationConfig{}
			if enabled {
				signing = applicationSigningConfig(t)
				policy.Grants = map[string]sso.ApplicationGrant{seed.Agent.ID.String(): {Audience: "fae-demo", Roles: []string{"pve"}}}
			}
			url, _ := testhelpers.Start(t, testhelpers.Config{BinaryPath: "./cmd/fulcrum", ConfigPath: applicationConfigFile(t, signing, policy), DB: tdb, Realm: coreRealm, Cover: true, ExtraEnv: []string{
				"FULCRUM_METRIC_DB_DSN=" + tdb.DSN(), "FULCRUM_SCHEDULER_LOCKER_DB_DSN=" + tdb.DSN(), "FULCRUM_CMP_SSO_SIGNING_KEY=", "FULCRUM_CMP_SSO_KEY_ID=", "FULCRUM_CMP_SSO_ISSUER=", "FULCRUM_APPLICATION_TOKEN_TTL=",
			}})
			public := resty.New()
			defer public.Close()
			resp, err := public.R().Get(url + "/.well-known/jwks.json")
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			var keys jose.JSONWebKeySet
			require.NoError(t, json.Unmarshal(resp.Bytes(), &keys))
			if enabled {
				require.Len(t, keys.Keys, 1)
				require.True(t, keys.Keys[0].IsPublic())
			} else {
				require.Empty(t, keys.Keys)
			}
			path := "/agents/me/access-token"
			resp, err = public.R().Post(url + "/api/v1" + path)
			require.NoError(t, err)
			require.Equal(t, 401, resp.StatusCode())
			for _, role := range []string{"participant1", "consumer1", "admin1"} {
				client := roleClient(t, url, role)
				defer client.Close()
				resp, err = client.R().Post(path)
				require.NoError(t, err)
				require.Equal(t, 403, resp.StatusCode(), resp.String())
			}
			provider := roleClient(t, url, "participant1")
			defer provider.Close()
			resp, err = provider.R().SetBody(map[string]any{"name": "provider changed metadata", "configuration": map[string]any{"roles": []string{"admin"}, "audience": "evil"}}).Patch("/agents/" + seed.Agent.ID.String())
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode(), resp.String())
			clients := []*resty.Client{testhelpers.NewClient(url, "e2e-agent-token"), roleClient(t, url, "agent1")}
			for _, client := range clients {
				defer client.Close()
				for _, body := range []string{"", `{"app":"another-agent","roles":["admin"],"aud":"evil","exp":9999999999}`, `malformed JSON`} {
					var result api.AgentAccessTokenRes
					resp, err = client.R().SetBody(body).SetResult(&result).Post(path)
					require.NoError(t, err)
					if !enabled {
						require.Equal(t, 503, resp.StatusCode(), resp.String())
						continue
					}
					require.Equal(t, http.StatusOK, resp.StatusCode(), resp.String())
					require.Equal(t, "no-store", resp.Header().Get("Cache-Control"))
					parsed, err := jwt.ParseSigned(result.Token, []jose.SignatureAlgorithm{jose.ES256})
					require.NoError(t, err)
					require.Equal(t, keys.Keys[0].KeyID, parsed.Headers[0].KeyID)
					var claims sso.ApplicationClaims
					require.NoError(t, parsed.Claims(keys, &claims))
					require.NoError(t, claims.Validate(jwt.Expected{Issuer: signing.Issuer, AnyAudience: jwt.Audience{"fae-demo"}, Time: time.Now()}))
					require.Equal(t, seed.Agent.ID.String(), claims.App)
					require.Equal(t, []string{"pve"}, claims.Roles)
					require.Equal(t, int64(300), int64(*claims.Expiry-*claims.IssuedAt))
					require.Equal(t, result.ExpiresAt.Unix(), claims.Expiry.Time().Unix())
					resp, err = public.R().SetHeader("Authorization", "Bearer "+result.Token).Post(url + "/api/v1" + path)
					require.NoError(t, err)
					require.Equal(t, 403, resp.StatusCode())
				}
			}
			if !enabled {
				return
			}
			// Providers can register another agent and mint its Core credential, not an application grant.
			var other api.AgentRes
			resp, err = provider.R().SetBody(api.CreateAgentReq{Name: "ungranted", ProviderID: seed.Provider.ID, AgentTypeID: seed.AgentType.ID, Configuration: &properties.JSON{"roles": []string{"pve"}, "audience": "fae-demo"}}).SetResult(&other).Post("/agents")
			require.NoError(t, err)
			require.Equal(t, 201, resp.StatusCode(), resp.String())
			var credential api.TokenRes
			resp, err = provider.R().SetBody(api.CreateTokenReq{Name: "ungranted credential", Role: auth.RoleAgent, ScopeID: &other.ID}).SetResult(&credential).Post("/tokens")
			require.NoError(t, err)
			require.Equal(t, 201, resp.StatusCode(), resp.String())
			ungranted := testhelpers.NewClient(url, credential.Value)
			defer ungranted.Close()
			resp, err = ungranted.R().SetBody(map[string]any{"app": seed.Agent.ID.String()}).Post(path)
			require.NoError(t, err)
			require.Equal(t, 403, resp.StatusCode(), resp.String())
			var events []domain.Event
			require.NoError(t, tdb.DB.Session(&gorm.Session{NewDB: true}).Where("type = ?", domain.EventTypeAgentAccessTokenIssued).Find(&events).Error)
			require.Len(t, events, 6)
			for _, event := range events {
				require.Equal(t, seed.Agent.ID, *event.AgentID)
				require.Equal(t, seed.Provider.ID, *event.ProviderID)
				require.Len(t, event.Payload, 2)
				require.Equal(t, "fae-demo", event.Payload["audience"])
				require.Equal(t, []any{"pve"}, event.Payload["roles"])
			}
			// Mandatory audit failure is observed over HTTP, not merely mocked.
			require.NoError(t, tdb.DB.Session(&gorm.Session{NewDB: true}).Exec(`CREATE FUNCTION reject_application_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.type = 'agent.access_token_issued' THEN RAISE EXCEPTION 'synthetic audit failure'; END IF; RETURN NEW; END $$`).Error)
			require.NoError(t, tdb.DB.Session(&gorm.Session{NewDB: true}).Exec(`CREATE TRIGGER reject_application_audit BEFORE INSERT ON events FOR EACH ROW EXECUTE FUNCTION reject_application_audit()`).Error)
			resp, err = clients[0].R().Post(path)
			require.NoError(t, err)
			require.Equal(t, 500, resp.StatusCode(), resp.String())
			require.NotContains(t, resp.String(), `"token":`)
			// A still-authenticated identity cannot issue after its persisted agent disappears.
			require.NoError(t, tdb.DB.Session(&gorm.Session{NewDB: true}).Delete(&domain.Agent{}, seed.Agent.ID).Error)
			resp, err = clients[1].R().Post(path)
			require.NoError(t, err)
			require.Equal(t, 404, resp.StatusCode(), resp.String())
			require.NotContains(t, resp.String(), `"token":`)
		})
	}
}
