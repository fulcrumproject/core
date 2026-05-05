//go:build e2e

package e2e

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/stretchr/testify/require"
)

func testServicePoolValue(t *testing.T, env *Env) {
	name := "spv-" + uniq()
	created := mustPost[api.CreateServicePoolValueReq, api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", api.CreateServicePoolValueReq{
		Name:          name,
		Value:         "10.0.0.1",
		ServicePoolID: env.Seed.ServicePool.ID,
	})
	require.Equal(t, env.Seed.ServicePool.ID, created.ServicePoolID)

	got := mustGet[api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", created.ID)
	require.Equal(t, created.ID, got.ID)

	page := mustList[api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values")
	require.GreaterOrEqual(t, page.TotalItems, int64(2))

	mustDelete(t, env.AdminClient, "/service-pool-values", created.ID)
}
