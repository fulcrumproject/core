//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/stretchr/testify/require"
)

func testServiceGroup(t *testing.T, env *Env) {
	t.Run("admin creates+gets+updates+deletes scoped to consumer", func(t *testing.T) {
		name := "sg-" + uniq()
		created := mustPost[api.CreateServiceGroupReq, api.ServiceGroupRes](t, env.AdminClient, "/service-groups", api.CreateServiceGroupReq{
			Name:       name,
			ConsumerID: env.Seed.Consumer.ID,
		})
		require.Equal(t, env.Seed.Consumer.ID, created.ConsumerID)

		got := mustGet[api.ServiceGroupRes](t, env.AdminClient, "/service-groups", created.ID)
		require.Equal(t, created.ID, got.ID)

		newName := "sg-renamed-" + uniq()
		updated := mustPatch[api.UpdateServiceGroupReq, api.ServiceGroupRes](t, env.AdminClient, "/service-groups", created.ID, api.UpdateServiceGroupReq{Name: &newName})
		require.Equal(t, newName, updated.Name)

		mustDelete(t, env.AdminClient, "/service-groups", created.ID)
	})

	t.Run("consumer creates own group", func(t *testing.T) {
		name := "sg-c-" + uniq()
		created := mustPost[api.CreateServiceGroupReq, api.ServiceGroupRes](t, env.ConsumerClient, "/service-groups", api.CreateServiceGroupReq{
			Name:       name,
			ConsumerID: env.Seed.Consumer.ID,
		})
		mustDelete(t, env.AdminClient, "/service-groups", created.ID)
	})

	t.Run("provider participant cannot create group for consumer", func(t *testing.T) {
		// participant1 is mapped to Provider; ConsumerID=Consumer is out of scope.
		resp, err := env.ParticipantClient.R().
			SetBody(api.CreateServiceGroupReq{
				Name:       "x-" + uniq(),
				ConsumerID: env.Seed.Consumer.ID,
			}).
			Post("/service-groups")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
