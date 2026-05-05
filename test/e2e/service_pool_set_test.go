//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/require"
)

func testServicePoolSet(t *testing.T, env *Env) {
	name := "sps-" + uniq()
	created := mustPost[api.CreateServicePoolSetReq, api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets", api.CreateServicePoolSetReq{
		Name:       name,
		ProviderID: env.Seed.Provider.ID,
	})
	require.Equal(t, name, created.Name)
	require.Equal(t, env.Seed.Provider.ID, created.ProviderID)
	require.NotEqual(t, properties.UUID{}, created.ID)
	require.False(t, time.Time(created.CreatedAt).IsZero())

	got := mustGet[api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets", created.ID)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Name, got.Name)
	require.Equal(t, created.ProviderID, got.ProviderID)

	newName := "sps-renamed-" + uniq()
	updated := mustPatch[api.UpdateServicePoolSetReq, api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets", created.ID, api.UpdateServicePoolSetReq{Name: &newName})
	require.Equal(t, newName, updated.Name)
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, created.ProviderID, updated.ProviderID, "PATCH name-only must not change FK")

	page := mustList[api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets")
	require.True(t, containsID(page.Items, created.ID), "list must include just-created pool set")

	mustDelete(t, env.AdminClient, "/service-pool-sets", created.ID)
	assertGone(t, env.AdminClient, "/service-pool-sets", created.ID)
}
