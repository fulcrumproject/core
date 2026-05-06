//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/require"
)

func testAgentPool(t *testing.T, env *Env) {
	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "ap-" + uniq()
		typeVal := "type_" + uniq()
		created := mustPost[api.CreateAgentPoolReq, api.AgentPoolRes](t, env.AdminClient, "/agent-pools", api.CreateAgentPoolReq{
			Name:          name,
			Type:          typeVal,
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, typeVal, created.Type)
		require.Equal(t, "string", created.PropertyType)
		require.Equal(t, domain.PoolGeneratorList, created.GeneratorType)
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := mustGet[api.AgentPoolRes](t, env.AdminClient, "/agent-pools", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.Type, got.Type)
		require.Equal(t, created.PropertyType, got.PropertyType)
		require.Equal(t, created.GeneratorType, got.GeneratorType)

		newName := "ap-renamed-" + uniq()
		updated := mustPatch[api.UpdateAgentPoolReq, api.AgentPoolRes](t, env.AdminClient, "/agent-pools", created.ID, api.UpdateAgentPoolReq{Name: &newName})
		require.Equal(t, newName, updated.Name)
		require.Equal(t, created.ID, updated.ID)
		require.Equal(t, created.Type, updated.Type, "PATCH name-only must not change type")
		require.Equal(t, created.PropertyType, updated.PropertyType, "PATCH name-only must not change propertyType")
		require.Equal(t, created.GeneratorType, updated.GeneratorType, "PATCH name-only must not change generatorType")

		page := mustList[api.AgentPoolRes](t, env.AdminClient, "/agent-pools")
		require.True(t, containsID(page.Items, created.ID), "list must include just-created agent pool")

		mustDelete(t, env.AdminClient, "/agent-pools", created.ID)
		assertGone(t, env.AdminClient, "/agent-pools", created.ID)
	})

	t.Run("participant cannot create agent pool", func(t *testing.T) {
		resp, err := env.ProviderClient.R().
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
