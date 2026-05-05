//go:build e2e

package e2e

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/stretchr/testify/require"
)

func testServicePool(t *testing.T, env *Env) {
	name := "sp-" + uniq()
	created := mustPost[api.CreateServicePoolReq, api.ServicePoolRes](t, env.AdminClient, "/service-pools", api.CreateServicePoolReq{
		Name:             name,
		Type:             "type_" + uniq(),
		PropertyType:     "string",
		GeneratorType:    domain.PoolGeneratorList,
		ServicePoolSetID: env.Seed.PoolSet.ID,
	})
	require.Equal(t, name, created.Name)

	got := mustGet[api.ServicePoolRes](t, env.AdminClient, "/service-pools", created.ID)
	require.Equal(t, created.ID, got.ID)

	newName := "sp-renamed-" + uniq()
	updated := mustPatch[api.UpdateServicePoolReq, api.ServicePoolRes](t, env.AdminClient, "/service-pools", created.ID, api.UpdateServicePoolReq{Name: &newName})
	require.Equal(t, newName, updated.Name)

	page := mustList[api.ServicePoolRes](t, env.AdminClient, "/service-pools")
	require.GreaterOrEqual(t, page.TotalItems, int64(2))

	mustDelete(t, env.AdminClient, "/service-pools", created.ID)
}
