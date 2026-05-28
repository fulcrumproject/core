//go:build e2e

package e2e

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

// testInfrastructureInstallScenario walks the full install-token lifecycle on
// a fresh infrastructure with a real config template and a persistent secret,
// and asserts:
//   - the public install URL renders the resolved config end to end
//   - the bootstrap bearer can hit /infrastructures/me and that /me resolves
//     to the bound row (i.e. the AgentID coordinate carries the infra id)
//   - the install URL is on the infrastructure prefix, and the same URL under
//     the /agents/install prefix 404s (cross-mount isolation)
//   - regenerate rotates the URL: the old one 404s, the new one serves
//
// Note: this scenario does NOT assert that the bootstrap bearer is blocked
// from reading sibling infrastructures owned by the same provider. The
// bootstrap token also carries ParticipantID = provider, and the existing
// DefaultObjectScope grants access whenever caller.ParticipantID equals
// target.ProviderID — same behavior as the agent bootstrap token. Tightening
// that to strict self-reference is a separate change that would affect both
// entity types and isn't in this phase's scope.
func testInfrastructureInstallScenario(t *testing.T, env *Env) {
	const (
		endpoint = "https://infra.example.com"
		apiKey   = "super-secret-infra-api-key"
	)

	infraType := testhelpers.MustPost[api.CreateInfrastructureTypeReq, api.InfrastructureTypeRes](t, env.AdminClient, "/infrastructure-types", api.CreateInfrastructureTypeReq{
		Name: "install-scenario-it-" + testhelpers.Uniq(),
		ConfigurationSchema: schema.Schema{
			Properties: map[string]schema.PropertyDefinition{
				"endpoint": {Type: "string", Label: "Endpoint", Required: true},
				"apiKey":   {Type: "string", Label: "API Key", Required: true, Secret: &schema.SecretConfig{Type: "persistent"}},
			},
		},
		ConfigContentType: "text/plain",
		ConfigTemplate:    "[infra]\nendpoint={{.endpoint}}\napi_key={{.apiKey}}\n",
		CmdTemplate:       "curl -fsSL {{.configUrl}} -H 'Authorization: Bearer {{.authToken}}' -o /tmp/infra.conf",
	})
	// Register the type cleanup BEFORE creating any infrastructure so the LIFO
	// stack runs the infra deletes first; otherwise the type delete races them
	// and fails with "N dependent infrastructure(s) exist".
	t.Cleanup(func() {
		testhelpers.MustDelete(t, env.AdminClient, "/infrastructure-types", infraType.ID)
	})

	mkInfra := func(t *testing.T, name string) *api.InfrastructureRes {
		t.Helper()
		cfg := properties.JSON{"endpoint": endpoint, "apiKey": apiKey}
		infra := testhelpers.MustPost[api.CreateInfrastructureReq, api.InfrastructureRes](t, env.AdminClient, "/infrastructures", api.CreateInfrastructureReq{
			Name:                 name + "-" + testhelpers.Uniq(),
			ProviderID:           env.Seed.Provider.ID,
			InfrastructureTypeID: infraType.ID,
			Tags:                 []string{"install-scenario"},
			Configuration:        &cfg,
		})
		t.Cleanup(func() {
			testhelpers.MustDelete(t, env.AdminClient, "/infrastructures", infra.ID)
			testhelpers.AssertGone(t, env.AdminClient, "/infrastructures", infra.ID)
		})
		return infra
	}

	infra := mkInfra(t, "install-scenario-infra")

	assertRendered := func(t *testing.T, body string) {
		t.Helper()
		require.Containsf(t, body, "endpoint="+endpoint, "rendered config missing endpoint line: %s", body)
		require.Containsf(t, body, "api_key="+apiKey, "rendered config missing resolved api_key (persistent secret): %s", body)
	}

	var (
		currentURL   string
		bootstrapTok string
	)

	t.Run("mint install command and fetch rendered config", func(t *testing.T) {
		minted := mustCreateInfraInstallToken(t, env.AdminClient, infra.ID)
		require.NotEmpty(t, minted.URL)
		require.Containsf(t, minted.InstallCommand, minted.URL, "installCommand should embed the install URL: %s", minted.InstallCommand)
		require.Contains(t, minted.URL, "/infrastructures/install/", "install URL should be on the infrastructure prefix")

		body := fetchInstall(t, env.AdminClient, minted.URL, http.StatusOK)
		assertRendered(t, body)
		t.Logf("rendered install config:\n%s", body)

		currentURL = minted.URL
		bootstrapTok = extractBearerToken(t, minted.InstallCommand)
		require.NotEmpty(t, bootstrapTok, "bootstrap token must be embedded in installCommand")
	})

	t.Run("bootstrap bearer resolves /me to this infrastructure", func(t *testing.T) {
		require.NotEmpty(t, bootstrapTok, "previous subtest must run first")

		bootstrapClient := testhelpers.NewClient(env.ServerURL, bootstrapTok)

		// /me resolves the identity's AgentID coordinate, which the install
		// flow set to this infrastructure's ID — proves the bootstrap token's
		// scope was wired through Phase 2's AgentID-as-self-reference.
		var me api.InfrastructureRes
		resp, err := bootstrapClient.R().SetResult(&me).Get("/infrastructures/me")
		require.NoError(t, err)
		require.Equalf(t, http.StatusOK, resp.StatusCode(), "GET /infrastructures/me: %s", resp.String())
		require.Equal(t, infra.ID, me.ID)
	})

	t.Run("regenerate rotates URL, secret still resolves", func(t *testing.T) {
		require.NotEmpty(t, currentURL, "previous subtests must run first")

		var rotated api.InstallTokenRes
		resp, err := env.AdminClient.R().
			SetPathParam("id", infra.ID.String()).
			SetResult(&rotated).
			Post("/infrastructures/{id}/install-command/regenerate")
		require.NoError(t, err)
		require.Equalf(t, http.StatusOK, resp.StatusCode(), "regenerate: %s", resp.String())
		require.NotEqual(t, currentURL, rotated.URL, "regenerate must rotate the URL")

		fetchInstall(t, env.AdminClient, currentURL, http.StatusNotFound)
		assertRendered(t, fetchInstall(t, env.AdminClient, rotated.URL, http.StatusOK))

		// Cross-entity isolation: the rotated infrastructure URL must not
		// resolve under the /agents/install prefix even with admin auth.
		agentMountURL := strings.Replace(rotated.URL, "/infrastructures/install/", "/agents/install/", 1)
		fetchInstall(t, env.AdminClient, agentMountURL, http.StatusNotFound)

		currentURL = rotated.URL
	})

	t.Run("revoke invalidates URL", func(t *testing.T) {
		require.NotEmpty(t, currentURL, "previous subtests must run first")

		resp, err := env.AdminClient.R().
			SetPathParam("id", infra.ID.String()).
			Delete("/infrastructures/{id}/install-command")
		require.NoError(t, err)
		require.Equalf(t, http.StatusNoContent, resp.StatusCode(), "revoke: %s", resp.String())

		fetchInstall(t, env.AdminClient, currentURL, http.StatusNotFound)
	})
}

// extractBearerToken pulls the bearer-token value out of a rendered cmdText.
// The cmd template renders Authorization: Bearer <token> inside the curl
// command, single-quoted; this regex tolerates either quote style.
var bearerRe = regexp.MustCompile(`Authorization: Bearer ([A-Za-z0-9_\-+/=]+)`)

func extractBearerToken(t *testing.T, cmdText string) string {
	t.Helper()
	m := bearerRe.FindStringSubmatch(cmdText)
	require.Lenf(t, m, 2, "could not locate Bearer token in installCommand: %s", cmdText)
	return m[1]
}
