//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/stretchr/testify/require"
)

func testAgentPool(t *testing.T, env *Env) {
	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "ap-" + uniq()
		created := mustPost[api.CreateAgentPoolReq, api.AgentPoolRes](t, env.AdminClient, "/agent-pools", api.CreateAgentPoolReq{
			Name:          name,
			Type:          "type_" + uniq(),
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
		})
		require.Equal(t, name, created.Name)

		got := mustGet[api.AgentPoolRes](t, env.AdminClient, "/agent-pools", created.ID)
		require.Equal(t, created.ID, got.ID)

		newName := "ap-renamed-" + uniq()
		updated := mustPatch[api.UpdateAgentPoolReq, api.AgentPoolRes](t, env.AdminClient, "/agent-pools", created.ID, api.UpdateAgentPoolReq{Name: &newName})
		require.Equal(t, newName, updated.Name)

		page := mustList[api.AgentPoolRes](t, env.AdminClient, "/agent-pools")
		require.GreaterOrEqual(t, page.TotalItems, int64(2))

		mustDelete(t, env.AdminClient, "/agent-pools", created.ID)
	})

	t.Run("participant cannot create agent pool", func(t *testing.T) {
		resp, err := env.ParticipantClient.R().
			SetBody(api.CreateAgentPoolReq{
				Name:          "p-" + uniq(),
				Type:          "type_" + uniq(),
				PropertyType:  "string",
				GeneratorType: domain.PoolGeneratorList,
			}).
			Post("/agent-pools")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
