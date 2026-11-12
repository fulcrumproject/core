// Tests generic JWT issuance against real Core, PostgreSQL and Keycloak.
// All signing keys, credentials and resource scopes are isolated synthetic data.

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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/database"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/jwtissuer"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"resty.dev/v3"
)

func jwtSigningConfig(t *testing.T) jwtissuer.Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	return jwtissuer.Config{SigningKey: string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})), KeyID: "generic-e2e", Issuer: "https://core.example"}
}

func jwtConfigFile(t *testing.T, cfg jwtissuer.Config) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{"jwt": cfg})
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "core.json")
	require.NoError(t, os.WriteFile(path, data, 0600))
	return path
}

func jwtEnv() []string {
	return []string{"FULCRUM_JWT_SIGNING_KEY=", "FULCRUM_JWT_KEY_ID=", "FULCRUM_JWT_ISSUER=", "FULCRUM_JWT_PARTICIPANT_TTL=", "FULCRUM_JWT_AGENT_TTL="}
}

func TestJWTInvalidBoot(t *testing.T) {
	for name, cfg := range map[string]jwtissuer.Config{
		"partial":                  {KeyID: "test"},
		"invalid key":              {SigningKey: "never-log-this-key", KeyID: "test", Issuer: "https://core.example"},
		"participant negative TTL": {ParticipantTTL: -time.Second},
		"agent negative TTL":       {AgentTTL: -time.Second},
		"fractional TTL":           {AgentTTL: time.Millisecond},
		"policy without signing":   {AgentTTL: time.Minute},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "go", "run", "./cmd/fulcrum", "-config", jwtConfigFile(t, cfg))
			cmd.Dir = testhelpers.RepoRoot(t)
			cmd.Env = append(os.Environ(), jwtEnv()...)
			cmd.Env = append(cmd.Env, "FULCRUM_DB_DSN=host="+t.TempDir()+" user=synthetic dbname=unavailable", "FULCRUM_PUBLIC_BASE_URL=http://127.0.0.1:3000")
			out, err := cmd.CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(out), "Invalid JWT configuration")
			require.NotContains(t, string(out), "Failed to initialize database")
			require.NotContains(t, string(out), "never-log-this-key")
			require.NotContains(t, string(out), "PRIVATE KEY")
		})
	}
}

func TestJWTIssueRoute(t *testing.T) {
	for _, mode := range []string{"disabled", "defaults", "custom lifetimes"} {
		t.Run(mode, func(t *testing.T) {
			tdb := database.NewTestDB(t)
			t.Cleanup(func() { tdb.Cleanup(t) })
			var seed *testhelpers.CoreFixtures
			require.NoError(t, tdb.DB.Transaction(func(tx *gorm.DB) error { var err error; seed, err = testhelpers.SeedCore(tx); return err }))
			db := func() *gorm.DB { return tdb.DB.Session(&gorm.Session{NewDB: true}) }
			// Native opaque agent credentials require both scopes. Do not alter Core auth.
			require.NoError(t, db().Model(seed.AgentToken).Update("participant_id", seed.Provider.ID).Error)
			providerToken := &domain.Token{Name: "synthetic provider credential", Role: auth.RoleParticipant, ParticipantID: &seed.Provider.ID, HashedValue: domain.HashTokenValue("e2e-provider-token"), ExpireAt: time.Now().Add(time.Hour)}
			require.NoError(t, db().Create(providerToken).Error)
			otherAgent := &domain.Agent{BaseEntity: domain.BaseEntity{ID: properties.NewUUID()}, ProviderID: seed.Provider.ID, AgentTypeID: seed.AgentType.ID, Name: "other assigned agent", Status: domain.AgentNew}
			require.NoError(t, db().Create(otherAgent).Error)
			otherService := *seed.Service
			otherService.ID = properties.NewUUID()
			otherService.AgentID = otherAgent.ID
			require.NoError(t, db().Create(&otherService).Error)
			foreignService := *seed.Service
			foreignService.ID = properties.NewUUID()
			foreignService.ProviderID = seed.Consumer.ID
			require.NoError(t, db().Create(&foreignService).Error)
			signing := jwtissuer.Config{}
			if mode != "disabled" {
				signing = jwtSigningConfig(t)
			}
			if mode == "custom lifetimes" {
				signing.ParticipantTTL = 45 * time.Second
				signing.AgentTTL = 2 * time.Minute
			}
			env := append(jwtEnv(), "FULCRUM_METRIC_DB_DSN="+tdb.DSN(), "FULCRUM_SCHEDULER_LOCKER_DB_DSN="+tdb.DSN())
			url, _ := testhelpers.Start(t, testhelpers.Config{BinaryPath: "./cmd/fulcrum", ConfigPath: jwtConfigFile(t, signing), DB: tdb, Realm: coreRealm, Cover: true, ExtraEnv: env})
			public := resty.New()
			defer public.Close()
			resp, err := public.R().Get(url + "/.well-known/jwks.json")
			require.NoError(t, err)
			require.Equal(t, 200, resp.StatusCode())
			var keys jose.JSONWebKeySet
			require.NoError(t, json.Unmarshal(resp.Bytes(), &keys))
			if mode == "disabled" {
				require.Empty(t, keys.Keys)
			} else {
				require.Len(t, keys.Keys, 1)
				require.True(t, keys.Keys[0].IsPublic())
				require.Equal(t, "generic-e2e", keys.Keys[0].KeyID)
			}
			var rawKeys struct {
				Keys []map[string]any `json:"keys"`
			}
			require.NoError(t, json.Unmarshal(resp.Bytes(), &rawKeys))
			for _, key := range rawKeys.Keys {
				for _, field := range []string{"d", "p", "q", "k"} {
					require.NotContains(t, key, field)
				}
			}
			resp, err = public.R().SetBody(map[string]any{"audience": "recipient"}).Post(url + "/api/v1/tokens/issue")
			require.NoError(t, err)
			require.Equal(t, 401, resp.StatusCode())
			successes := 0
			for _, caller := range []struct {
				name, role, relation, credential string
				participantID, agentID           properties.UUID
			}{
				{name: "provider Keycloak", role: "participant", relation: "provider", credential: "participant1", participantID: seed.Provider.ID},
				{name: "consumer Keycloak", role: "participant", relation: "consumer", credential: "consumer1", participantID: seed.Consumer.ID},
				{name: "agent Keycloak", role: "agent", relation: "provider", credential: "agent1", participantID: seed.Provider.ID, agentID: seed.Agent.ID},
				{name: "provider opaque", role: "participant", relation: "provider", credential: "e2e-provider-token", participantID: seed.Provider.ID},
				{name: "agent opaque", role: "agent", relation: "provider", credential: "e2e-agent-token", participantID: seed.Provider.ID, agentID: seed.Agent.ID},
			} {
				t.Run(caller.name, func(t *testing.T) {
					var client *resty.Client
					if strings.HasPrefix(caller.credential, "e2e-") {
						client = testhelpers.NewClient(url, caller.credential)
					} else {
						client = roleClient(t, url, caller.credential)
					}
					defer client.Close()
					// Native APIs continue accepting both incoming credential kinds.
					resp, err := client.R().Get("/services/" + seed.Service.ID.String())
					require.NoError(t, err)
					require.Equal(t, 200, resp.StatusCode(), resp.String())
					for _, withService := range []bool{false, true} {
						audience := "recipient"
						if !withService {
							audience = url + "/api/v1"
						} // Core-like audiences still cannot authenticate back.
						body := map[string]any{"audience": audience, "ext": map[string]any{"order": "synthetic", "nested": map[string]any{"role": "not certified"}}}
						if withService {
							body["serviceId"] = seed.Service.ID.String()
						}
						var result api.IssueTokenRes
						var rawResponse map[string]any
						resp, err = client.R().SetBody(body).SetResult(&rawResponse).Post("/tokens/issue?sub=forged&role=admin&exp=9999999999")
						require.NoError(t, err)
						if mode == "disabled" {
							require.Equal(t, 503, resp.StatusCode(), resp.String())
							continue
						}
						require.Equal(t, 200, resp.StatusCode(), resp.String())
						successes++
						require.Equal(t, "no-store", resp.Header().Get("Cache-Control"))
						require.Len(t, rawResponse, 2)
						responseJSON, err := json.Marshal(rawResponse)
						require.NoError(t, err)
						require.NoError(t, json.Unmarshal(responseJSON, &result))
						parsed, err := jwt.ParseSigned(result.Token, []jose.SignatureAlgorithm{jose.ES256})
						require.NoError(t, err)
						var claims jwtissuer.Claims
						require.NoError(t, parsed.Claims(keys, &claims))
						subject := caller.participantID.String()
						ttl := int64(90)
						if caller.role == "agent" {
							subject = caller.agentID.String()
							ttl = 300
						}
						if mode == "custom lifetimes" {
							ttl = 45
							if caller.role == "agent" {
								ttl = 120
							}
						}
						require.NoError(t, claims.Validate(jwt.Expected{Issuer: signing.Issuer, Subject: subject, AnyAudience: jwt.Audience{audience}, Time: time.Now()}))
						require.Equal(t, caller.role, claims.Role)
						require.Equal(t, caller.participantID.String(), claims.ParticipantID)
						require.Equal(t, ttl, int64(*claims.Expiry-*claims.IssuedAt))
						require.Equal(t, *claims.IssuedAt, *claims.NotBefore)
						require.NotEmpty(t, claims.ID)
						require.Equal(t, result.ExpiresAt.Unix(), claims.Expiry.Time().Unix())
						require.Equal(t, "synthetic", claims.Ext["order"])
						var raw map[string]any
						require.NoError(t, parsed.Claims(keys, &raw))
						for _, key := range []string{"app", "roles", "redirectUrl"} {
							require.NotContains(t, raw, key)
						}
						if caller.role == "agent" {
							require.Equal(t, caller.agentID.String(), claims.AgentID)
						} else {
							require.NotContains(t, raw, "agent_id")
						}
						if withService {
							require.Equal(t, seed.Service.ID.String(), claims.ServiceID)
							require.Equal(t, caller.relation, claims.Relation)
						} else {
							require.NotContains(t, raw, "service_id")
							require.NotContains(t, raw, "relation")
						}
						for _, path := range []string{"/services/" + seed.Service.ID.String(), "/tokens"} {
							resp, err = public.R().SetHeader("Authorization", "Bearer "+result.Token).Get(url + "/api/v1" + path)
							require.NoError(t, err)
							require.Equal(t, 403, resp.StatusCode(), resp.String())
						}
					}
				})
			}
			provider, agent, admin := roleClient(t, url, "participant1"), roleClient(t, url, "agent1"), roleClient(t, url, "admin1")
			defer provider.Close()
			defer agent.Close()
			defer admin.Close()
			resp, err = admin.R().SetBody(map[string]any{"audience": "recipient"}).Post("/tokens/issue")
			require.NoError(t, err)
			require.Equal(t, 403, resp.StatusCode(), resp.String())
			// No specialized alias is left; native lifecycle fallback must never issue.
			resp, err = agent.R().Post("/agents/me/access-token")
			require.NoError(t, err)
			require.Equal(t, 404, resp.StatusCode(), resp.String())
			resp, err = provider.R().SetBody(map[string]any{}).Post("/services/" + seed.Service.ID.String() + "/cmp-launch-token")
			require.NoError(t, err)
			require.NotEqual(t, 200, resp.StatusCode(), resp.String())
			require.NotContains(t, resp.String(), `"token":`)
			for _, body := range []map[string]any{
				{"audience": "recipient", "serviceId": otherService.ID.String()},
				{"audience": "recipient", "serviceId": foreignService.ID.String()},
			} {
				resp, err = agent.R().SetBody(body).Post("/tokens/issue")
				require.NoError(t, err)
				require.Equal(t, 403, resp.StatusCode(), resp.String())
			}
			resp, err = provider.R().SetBody(map[string]any{"audience": "recipient", "serviceId": foreignService.ID.String()}).Post("/tokens/issue")
			require.NoError(t, err)
			require.Equal(t, 403, resp.StatusCode(), resp.String())
			for _, body := range []string{`{"audience":"recipient","role":"admin"}`, `{"audience":"recipient","ext":{"agent_id":"forged"}}`, `{"audience":"recipient","ext":{"relation":"provider"}}`, `{"audience":"recipient","ext":{"blob":"` + strings.Repeat("x", 4096) + `"}}`} {
				resp, err = provider.R().SetBody(body).Post("/tokens/issue")
				require.NoError(t, err)
				require.Equal(t, 400, resp.StatusCode(), resp.String())
			}
			// Malformed opaque scopes cannot impersonate a valid caller even when authenticated.
			for _, tc := range []struct {
				name     string
				role     auth.Role
				pid, aid *properties.UUID
			}{
				{name: "participant with agent", role: auth.RoleParticipant, pid: &seed.Provider.ID, aid: &seed.Agent.ID},
				{name: "agent provider mismatch", role: auth.RoleAgent, pid: &seed.Consumer.ID, aid: &seed.Agent.ID},
				{name: "agent missing participant", role: auth.RoleAgent, aid: &seed.Agent.ID},
				{name: "agent missing agent", role: auth.RoleAgent, pid: &seed.Provider.ID},
				{name: "admin scoped as agent", role: auth.RoleAdmin, pid: &seed.Provider.ID, aid: &seed.Agent.ID},
			} {
				credential := &domain.Token{Name: tc.name, Role: tc.role, ParticipantID: tc.pid, AgentID: tc.aid, HashedValue: domain.HashTokenValue(tc.name), ExpireAt: time.Now().Add(time.Hour)}
				require.NoError(t, db().Create(credential).Error)
				client := testhelpers.NewClient(url, tc.name)
				resp, err = client.R().SetBody(map[string]any{"audience": "recipient"}).Post("/tokens/issue")
				client.Close()
				require.NoError(t, err)
				require.Equal(t, 403, resp.StatusCode(), resp.String())
			}
			var events []domain.Event
			require.NoError(t, db().Where("type = ?", domain.EventTypeTokenIssued).Find(&events).Error)
			require.Len(t, events, successes)
			for _, event := range events {
				require.NotNil(t, event.ParticipantID)
				require.NotEmpty(t, event.InitiatorID)
				expected := 2
				if _, ok := event.Payload["relation"]; ok {
					expected = 3
				}
				require.Len(t, event.Payload, expected)
				require.Contains(t, event.Payload, "role")
				require.Contains(t, event.Payload, "audience")
			}
			if mode == "disabled" {
				return
			}
			require.NoError(t, db().Exec(`CREATE FUNCTION reject_jwt_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.type = 'token.issued' THEN RAISE EXCEPTION 'synthetic audit failure'; END IF; RETURN NEW; END $$`).Error)
			require.NoError(t, db().Exec(`CREATE TRIGGER reject_jwt_audit BEFORE INSERT ON events FOR EACH ROW EXECUTE FUNCTION reject_jwt_audit()`).Error)
			resp, err = provider.R().SetBody(map[string]any{"audience": "recipient"}).Post("/tokens/issue")
			require.NoError(t, err)
			require.Equal(t, 500, resp.StatusCode(), resp.String())
			require.NotContains(t, resp.String(), `"token":`)
			require.NoError(t, db().Delete(&domain.Agent{}, seed.Agent.ID).Error)
			resp, err = agent.R().SetBody(map[string]any{"audience": "recipient"}).Post("/tokens/issue")
			require.NoError(t, err)
			require.Equal(t, 404, resp.StatusCode(), resp.String())
			require.NotContains(t, resp.String(), `"token":`)
			require.NoError(t, db().Delete(&domain.Participant{}, seed.Provider.ID).Error)
			resp, err = provider.R().SetBody(map[string]any{"audience": "recipient"}).Post("/tokens/issue")
			require.NoError(t, err)
			require.Equal(t, 404, resp.StatusCode(), resp.String())
			require.NotContains(t, resp.String(), `"token":`)
		})
	}
}
