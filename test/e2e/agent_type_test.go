//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/stretchr/testify/require"
)

func testAgentType(t *testing.T, env *Env) {
	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "at-" + uniq()
		created := mustPost[api.CreateAgentTypeReq, api.AgentTypeRes](t, env.AdminClient, "/agent-types", api.CreateAgentTypeReq{
			Name: name,
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"apiEndpoint": {Type: "string", Label: "API Endpoint", Required: true},
				},
			},
			ConfigContentType: "text/plain",
			ConfigTemplate:    "[agent]\nendpoint={{.apiEndpoint}}\n",
			CmdTemplate:       "curl -fsSL {{.configUrl}} -H 'Authorization: Bearer {{.authToken}}'",
		})
		require.Equal(t, name, created.Name)

		got := mustGet[api.AgentTypeRes](t, env.AdminClient, "/agent-types", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, "[agent]\nendpoint={{.apiEndpoint}}\n", got.ConfigTemplate)

		newName := "at-renamed-" + uniq()
		updated := mustPatch[api.UpdateAgentTypeReq, api.AgentTypeRes](t, env.AdminClient, "/agent-types", created.ID, api.UpdateAgentTypeReq{Name: &newName})
		require.Equal(t, newName, updated.Name)

		page := mustList[api.AgentTypeRes](t, env.AdminClient, "/agent-types")
		require.GreaterOrEqual(t, page.TotalItems, int64(2), "seed (1) + new (1)")

		mustDelete(t, env.AdminClient, "/agent-types", created.ID)
	})

	t.Run("rejects configTemplate referencing unknown schema field", func(t *testing.T) {
		// configTemplate refs must exist in the schema; agent_type.http pins
		// this validation as a 400 case.
		resp, err := env.AdminClient.R().
			SetBody(api.CreateAgentTypeReq{
				Name: "bad-cfg-ref-" + uniq(),
				ConfigurationSchema: schema.Schema{
					Properties: map[string]schema.PropertyDefinition{
						"host": {Type: "string", Required: true},
					},
				},
				ConfigTemplate: "host={{.missing}}",
			}).
			Post("/agent-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("participant cannot create agent type", func(t *testing.T) {
		resp, err := env.ParticipantClient.R().
			SetBody(api.CreateAgentTypeReq{
				Name:                "p-" + uniq(),
				ConfigurationSchema: schema.Schema{},
			}).
			Post("/agent-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
