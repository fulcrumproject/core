//go:build e2e

package e2e

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/stretchr/testify/require"
)

func testServicePoolSet(t *testing.T, env *Env) {
	name := "sps-" + uniq()
	created := mustPost[api.CreateServicePoolSetReq, api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets", api.CreateServicePoolSetReq{
		Name:       name,
		ProviderID: env.Seed.Provider.ID,
	})
	require.Equal(t, env.Seed.Provider.ID, created.ProviderID)

	got := mustGet[api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets", created.ID)
	require.Equal(t, created.ID, got.ID)

	newName := "sps-renamed-" + uniq()
	updated := mustPatch[api.UpdateServicePoolSetReq, api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets", created.ID, api.UpdateServicePoolSetReq{Name: &newName})
	require.Equal(t, newName, updated.Name)

	page := mustList[api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets")
	require.GreaterOrEqual(t, page.TotalItems, int64(2))

	mustDelete(t, env.AdminClient, "/service-pool-sets", created.ID)
}
