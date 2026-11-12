// Tests public discovery and service-scoped launch through a running Core binary.
// Synthetic Keycloak credentials remain the source of incoming authentication.

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
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/database"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/sso"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"resty.dev/v3"
)

func TestCmpSsoInvalidBoot(t *testing.T) {
	for _, tc := range []struct{ name, key, kid, issuer, ttl, message string }{
		{name: "partial", kid: "test", ttl: "90s", message: "all required"},
		{name: "invalid key", key: "never-log-this-key", kid: "test", issuer: "https://core.example", ttl: "90s", message: "parse CMP SSO signing key"},
		{name: "negative TTL", key: "never-log-this-key", kid: "test", issuer: "https://core.example", ttl: "-1s", message: "TOKEN_TTL must be > 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "go", "run", "./cmd/fulcrum")
			cmd.Dir = testhelpers.RepoRoot(t)
			cmd.Env = append(os.Environ(), "FULCRUM_DB_DSN=host="+t.TempDir()+" user=synthetic dbname=unavailable", "FULCRUM_PUBLIC_BASE_URL=http://127.0.0.1:3000", "FULCRUM_CMP_SSO_SIGNING_KEY="+tc.key, "FULCRUM_CMP_SSO_KEY_ID="+tc.kid, "FULCRUM_CMP_SSO_ISSUER="+tc.issuer, "FULCRUM_CMP_SSO_TOKEN_TTL="+tc.ttl)
			out, err := cmd.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(out), "Invalid CMP SSO configuration")
			require.Contains(t, string(out), tc.message)
			require.NotContains(t, string(out), "Failed to initialize database")
			require.NotContains(t, string(out), "never-log-this-key")
		})
	}
}

func TestCmpSso(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		t.Run(name, func(t *testing.T) {
			tdb := database.NewTestDB(t)
			t.Cleanup(func() { tdb.Cleanup(t) })
			var seed *testhelpers.CoreFixtures
			require.NoError(t, tdb.DB.Transaction(func(tx *gorm.DB) error {
				var err error
				seed, err = testhelpers.SeedCore(tx)
				return err
			}))
			cfg := properties.JSON{"cmpLandingUrl": "https://cmp.example/sso/landing", "cmpAudience": "cmp-test"}
			require.NoError(t, tdb.DB.Session(&gorm.Session{NewDB: true}).Model(seed.Agent).Update("configuration", cfg).Error)
			env := []string{"FULCRUM_METRIC_DB_DSN=" + tdb.DSN(), "FULCRUM_SCHEDULER_LOCKER_DB_DSN=" + tdb.DSN(), "FULCRUM_CMP_SSO_SIGNING_KEY=", "FULCRUM_CMP_SSO_KEY_ID=", "FULCRUM_CMP_SSO_ISSUER=", "FULCRUM_CMP_SSO_TOKEN_TTL=90s"}
			if enabled {
				key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				require.NoError(t, err)
				der, err := x509.MarshalECPrivateKey(key)
				require.NoError(t, err)
				env = append(env, "FULCRUM_CMP_SSO_SIGNING_KEY="+string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})), "FULCRUM_CMP_SSO_KEY_ID=e2e-key", "FULCRUM_CMP_SSO_ISSUER=https://core.example")
			}
			url, _ := testhelpers.Start(t, testhelpers.Config{BinaryPath: "./cmd/fulcrum", DB: tdb, Realm: coreRealm, ExtraEnv: env, Cover: true})
			publicClient := resty.New()
			defer publicClient.Close()
			resp, err := publicClient.R().Get(url + "/.well-known/jwks.json")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())
			var keys jose.JSONWebKeySet
			require.NoError(t, json.Unmarshal(resp.Bytes(), &keys))
			if enabled {
				require.Len(t, keys.Keys, 1)
				require.True(t, keys.Keys[0].IsPublic())
				require.Equal(t, "e2e-key", keys.Keys[0].KeyID)
			} else {
				require.Empty(t, keys.Keys)
			}
			path := "/services/" + seed.Service.ID.String() + "/cmp-launch-token"
			resp, err = publicClient.R().SetBody(map[string]any{}).Post(url + "/api/v1" + path)
			require.NoError(t, err)
			require.Equal(t, http.StatusUnauthorized, resp.StatusCode())
			for _, role := range []string{"participant1", "consumer1", "agent1", "admin1"} {
				client := roleClient(t, url, role)
				defer client.Close()
				// Existing APIs still accept external Keycloak access tokens.
				testhelpers.MustGet[api.ServiceRes](t, client, "/services", seed.Service.ID)
				var result api.CmpLaunchTokenRes
				resp, err = client.R().SetBody(map[string]any{"ext": map[string]any{"order": "synthetic"}, "redirectUrl": "https://evil.example"}).SetResult(&result).Post(path)
				require.NoError(t, err)
				if role == "agent1" || role == "admin1" {
					require.Equal(t, http.StatusForbidden, resp.StatusCode(), resp.String())
					continue
				}
				if !enabled {
					require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode(), resp.String())
					continue
				}
				require.Equal(t, http.StatusOK, resp.StatusCode(), resp.String())
				require.Equal(t, "https://cmp.example/sso/landing", result.RedirectURL)
				parsed, err := jwt.ParseSigned(result.Token, []jose.SignatureAlgorithm{jose.ES256})
				require.NoError(t, err)
				var claims sso.LaunchClaims
				require.NoError(t, parsed.Claims(keys, &claims))
				require.NoError(t, claims.Validate(jwt.Expected{Issuer: "https://core.example", AnyAudience: jwt.Audience{"cmp-test"}, Time: time.Now()}))
				relation, participantID := "provider", seed.Provider.ID.String()
				if role == "consumer1" {
					relation, participantID = "consumer", seed.Consumer.ID.String()
				}
				require.Equal(t, relation, claims.Relation)
				require.Equal(t, participantID, claims.Subject)
				require.Equal(t, participantID, claims.ParticipantID)
				require.Equal(t, seed.Service.ID.String(), claims.ServiceID)
				require.Equal(t, "synthetic", claims.Ext["order"])
				require.Equal(t, int64(90), int64(*claims.Expiry-*claims.IssuedAt))
				require.Equal(t, result.ExpiresAt.Unix(), claims.Expiry.Time().Unix())
				resp, err = client.R().SetBody(map[string]any{"ext": map[string]any{"aud": "evil"}}).Post(path)
				require.NoError(t, err)
				require.Equal(t, http.StatusBadRequest, resp.StatusCode())
				// The new issuer does not authenticate incoming API requests.
				resp, err = publicClient.R().SetHeader("Authorization", "Bearer "+result.Token).Get(url + "/api/v1/services/" + seed.Service.ID.String())
				require.NoError(t, err)
				require.Equal(t, http.StatusForbidden, resp.StatusCode())
			}
			var events []domain.Event
			require.NoError(t, tdb.DB.Session(&gorm.Session{NewDB: true}).Where("type = ?", domain.EventTypeServiceCmpLaunched).Find(&events).Error)
			if enabled {
				require.Len(t, events, 2)
				for _, e := range events {
					require.Equal(t, seed.Service.ID, *e.EntityID)
					require.Len(t, e.Payload, 2)
					require.Equal(t, "cmp-test", e.Payload["audience"])
				}
			} else {
				require.Empty(t, events)
			}
		})
	}
}
