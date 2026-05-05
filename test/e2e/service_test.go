//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/require"
)

func testService(t *testing.T, env *Env) {
	t.Run("admin lists services includes seed", func(t *testing.T) {
		page := mustList[api.ServiceRes](t, env.AdminClient, "/services")
		require.GreaterOrEqual(t, page.TotalItems, int64(1))
	})

	t.Run("admin creates+gets+updates service", func(t *testing.T) {
		agentID := env.Seed.Agent.ID
		name := "svc-" + uniq()
		created := mustPost[api.CreateServiceReq, api.ServiceRes](t, env.AdminClient, "/services", api.CreateServiceReq{
			GroupID:       env.Seed.Group.ID,
			AgentID:       &agentID,
			ServiceTypeID: env.Seed.ServiceType.ID,
			Name:          name,
			Properties:    properties.JSON{},
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, "creating", created.Status, "service starts in InitialState")
		require.Equal(t, env.Seed.Group.ID, created.GroupID)
		require.Equal(t, env.Seed.ServiceType.ID, created.ServiceTypeID)
		require.Equal(t, agentID, created.AgentID)
		require.Equal(t, env.Seed.Provider.ID, created.ProviderID, "provider derived from agent")
		require.Equal(t, env.Seed.Consumer.ID, created.ConsumerID, "consumer derived from group")
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())
		t.Cleanup(func() {
			// Best-effort cleanup; ignore status code (test may have transitioned/deleted).
			_, _ = env.AdminClient.R().
				SetPathParam("id", created.ID.String()).
				Delete("/services/{id}")
		})

		got := mustGet[api.ServiceRes](t, env.AdminClient, "/services", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.Status, got.Status)
		require.Equal(t, created.GroupID, got.GroupID)
		require.Equal(t, created.AgentID, got.AgentID)
		require.Equal(t, created.ServiceTypeID, got.ServiceTypeID)
		require.Equal(t, created.ProviderID, got.ProviderID)
		require.Equal(t, created.ConsumerID, got.ConsumerID)

		page := mustList[api.ServiceRes](t, env.AdminClient, "/services")
		require.True(t, containsID(page.Items, created.ID), "list must include just-created service")

		newName := "svc-renamed-" + uniq()
		updated := mustPatch[api.UpdateServiceReq, api.ServiceRes](t, env.AdminClient, "/services", created.ID, api.UpdateServiceReq{Name: &newName})
		require.Equal(t, newName, updated.Name)
		require.Equal(t, created.ID, updated.ID)
		require.Equal(t, created.Status, updated.Status, "PATCH must not transition status")
		require.Equal(t, created.GroupID, updated.GroupID, "PATCH must not change FK")
		require.Equal(t, created.AgentID, updated.AgentID, "PATCH must not change FK")
		require.Equal(t, created.ServiceTypeID, updated.ServiceTypeID, "PATCH must not change FK")
	})

	t.Run("rejects undefined action", func(t *testing.T) {
		// "restart" is not declared in the seed lifecycle. We hit the
		// undefined-action 400 on the seed service before any other test
		// mutates its state.
		resp, err := env.AdminClient.R().
			SetPathParam("id", env.Seed.Service.ID.String()).
			Post("/services/{id}/restart")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", resp.String())
	})

}
