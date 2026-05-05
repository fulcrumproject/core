//go:build e2e

package e2e

import (
	"net/http"
	"strings"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

// testAgentInstallScenario walks the full install-token lifecycle on a fresh
// agent built with a real config template, and asserts the public install URL
// returns the rendered configuration with the persistent secret resolved end
// to end (vault round-trip + template render).
func testAgentInstallScenario(t *testing.T, env *Env) {
	const (
		apiEndpoint = "https://api.example.com"
		apiKey      = "super-secret-api-key-42"
		maxRetries  = 5
	)

	agentType := mustPost[api.CreateAgentTypeReq, api.AgentTypeRes](t, env.AdminClient, "/agent-types", api.CreateAgentTypeReq{
		Name: "install-scenario-type-" + uniq(),
		ConfigurationSchema: schema.Schema{
			Properties: map[string]schema.PropertyDefinition{
				"apiEndpoint": {Type: "string", Label: "API Endpoint", Required: true},
				"apiKey":      {Type: "string", Label: "API Key", Required: true, Secret: &schema.SecretConfig{Type: "persistent"}},
				"maxRetries":  {Type: "integer", Label: "Max Retries", Default: 3},
			},
		},
		ConfigContentType: "text/plain",
		ConfigTemplate:    "[agent]\nendpoint={{.apiEndpoint}}\napi_key={{.apiKey}}\nretries={{.maxRetries}}\n",
		CmdTemplate:       "curl -fsSL {{.configUrl}} -H 'Authorization: Bearer {{.authToken}}' -o /tmp/agent.conf",
	})

	cfg := properties.JSON{
		"apiEndpoint": apiEndpoint,
		"apiKey":      apiKey,
		"maxRetries":  maxRetries,
	}
	agent := mustPost[api.CreateAgentReq, api.AgentRes](t, env.AdminClient, "/agents", api.CreateAgentReq{
		Name:          "install-scenario-agent-" + uniq(),
		ProviderID:    env.Seed.Provider.ID,
		AgentTypeID:   agentType.ID,
		Tags:          []string{"install-scenario"},
		Configuration: &cfg,
	})

	t.Cleanup(func() {
		mustDelete(t, env.AdminClient, "/agents", agent.ID)
		assertGone(t, env.AdminClient, "/agents", agent.ID)
		mustDelete(t, env.AdminClient, "/agent-types", agentType.ID)
	})

	assertRendered := func(t *testing.T, body string) {
		t.Helper()
		require.Containsf(t, body, "endpoint="+apiEndpoint, "rendered config missing endpoint line: %s", body)
		require.Containsf(t, body, "api_key="+apiKey, "rendered config missing resolved api_key (persistent secret): %s", body)
		require.Containsf(t, body, "retries=5", "rendered config missing retries line: %s", body)
	}

	var currentURL string

	t.Run("mint install command and fetch rendered config", func(t *testing.T) {
		minted := mustCreateInstallToken(t, env.AdminClient, agent.ID)
		require.NotEmpty(t, minted.URL)
		require.Containsf(t, minted.InstallCommand, minted.URL, "installCommand should embed the install URL: %s", minted.InstallCommand)

		body := fetchInstall(t, env.AdminClient, minted.URL, http.StatusOK)
		assertRendered(t, body)
		t.Logf("rendered install config:\n%s", body)

		currentURL = minted.URL
	})

	t.Run("regenerate rotates URL, secret still resolves", func(t *testing.T) {
		require.NotEmpty(t, currentURL, "previous subtest must run first")

		var rotated api.InstallTokenRes
		resp, err := env.AdminClient.R().
			SetPathParam("id", agent.ID.String()).
			SetResult(&rotated).
			Post("/agents/{id}/install-command/regenerate")
		require.NoError(t, err)
		require.Equalf(t, http.StatusOK, resp.StatusCode(), "regenerate: %s", resp.String())
		require.NotEqual(t, currentURL, rotated.URL, "regenerate must rotate the URL")

		fetchInstall(t, env.AdminClient, currentURL, http.StatusNotFound)
		assertRendered(t, fetchInstall(t, env.AdminClient, rotated.URL, http.StatusOK))

		currentURL = rotated.URL
	})

	t.Run("revoke invalidates URL and re-mint produces a fresh working URL", func(t *testing.T) {
		require.NotEmpty(t, currentURL, "previous subtests must run first")

		resp, err := env.AdminClient.R().
			SetPathParam("id", agent.ID.String()).
			Delete("/agents/{id}/install-command")
		require.NoError(t, err)
		require.Equalf(t, http.StatusNoContent, resp.StatusCode(), "revoke: %s", resp.String())

		fetchInstall(t, env.AdminClient, currentURL, http.StatusNotFound)

		fresh := mustCreateInstallToken(t, env.AdminClient, agent.ID)
		require.NotEqual(t, currentURL, fresh.URL)
		assertRendered(t, fetchInstall(t, env.AdminClient, fresh.URL, http.StatusOK))
	})
}

// fetchInstall GETs the absolute install URL. resty still applies the bearer
// token even when the URL bypasses BaseURL, which the route requires.
func fetchInstall(t *testing.T, c *resty.Client, url string, want int) string {
	t.Helper()
	resp, err := c.R().Get(url)
	require.NoError(t, err)
	require.Equalf(t, want, resp.StatusCode(), "GET %s: %s", url, resp.String())
	if want == http.StatusOK {
		require.True(t, strings.HasPrefix(resp.Header().Get("Content-Type"), "text/plain"),
			"expected text/plain, got %q", resp.Header().Get("Content-Type"))
	}
	return resp.String()
}
