//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func testServiceGroup(t *testing.T, env *Env) {
	t.Run("admin creates+gets+updates+deletes scoped to consumer", func(t *testing.T) {
		name := "sg-" + testhelpers.Uniq()
		created := testhelpers.MustPost[api.CreateServiceGroupReq, api.ServiceGroupRes](t, env.AdminClient, "/service-groups", api.CreateServiceGroupReq{
			Name:       name,
			ConsumerID: env.Seed.Consumer.ID,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, env.Seed.Consumer.ID, created.ConsumerID)
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero(), "createdAt populated")

		got := testhelpers.MustGet[api.ServiceGroupRes](t, env.AdminClient, "/service-groups", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.ConsumerID, got.ConsumerID)

		newName := "sg-renamed-" + testhelpers.Uniq()
		updated := testhelpers.MustPatch[api.UpdateServiceGroupReq, api.ServiceGroupRes](t, env.AdminClient, "/service-groups", created.ID, api.UpdateServiceGroupReq{Name: &newName})
		require.Equal(t, newName, updated.Name)
		require.Equal(t, created.ID, updated.ID)
		require.Equal(t, created.ConsumerID, updated.ConsumerID, "PATCH must not silently change FK")

		page := testhelpers.MustList[api.ServiceGroupRes](t, env.AdminClient, "/service-groups")
		require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created group")

		testhelpers.MustDelete(t, env.AdminClient, "/service-groups", created.ID)
		testhelpers.AssertGone(t, env.AdminClient, "/service-groups", created.ID)
	})

	t.Run("consumer creates own group", func(t *testing.T) {
		name := "sg-c-" + testhelpers.Uniq()
		created := testhelpers.MustPost[api.CreateServiceGroupReq, api.ServiceGroupRes](t, env.ConsumerClient, "/service-groups", api.CreateServiceGroupReq{
			Name:       name,
			ConsumerID: env.Seed.Consumer.ID,
		})
		testhelpers.MustDelete(t, env.AdminClient, "/service-groups", created.ID)
	})

	t.Run("provider participant cannot create group for consumer", func(t *testing.T) {
		// participant1 is mapped to Provider; ConsumerID=Consumer is out of scope.
		resp, err := env.ProviderClient.R().
			SetBody(api.CreateServiceGroupReq{
				Name:       "x-" + testhelpers.Uniq(),
				ConsumerID: env.Seed.Consumer.ID,
			}).
			Post("/service-groups")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
