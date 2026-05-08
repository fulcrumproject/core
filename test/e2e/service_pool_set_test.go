//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func testServicePoolSet(t *testing.T, env *Env) {
	name := "sps-" + testhelpers.Uniq()
	created := testhelpers.MustPost[api.CreateServicePoolSetReq, api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets", api.CreateServicePoolSetReq{
		Name:       name,
		ProviderID: env.Seed.Provider.ID,
	})
	require.Equal(t, name, created.Name)
	require.Equal(t, env.Seed.Provider.ID, created.ProviderID)
	require.NotEqual(t, properties.UUID{}, created.ID)
	require.False(t, time.Time(created.CreatedAt).IsZero())

	got := testhelpers.MustGet[api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets", created.ID)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Name, got.Name)
	require.Equal(t, created.ProviderID, got.ProviderID)

	newName := "sps-renamed-" + testhelpers.Uniq()
	updated := testhelpers.MustPatch[api.UpdateServicePoolSetReq, api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets", created.ID, api.UpdateServicePoolSetReq{Name: &newName})
	require.Equal(t, newName, updated.Name)
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, created.ProviderID, updated.ProviderID, "PATCH name-only must not change FK")

	page := testhelpers.MustList[api.ServicePoolSetRes](t, env.AdminClient, "/service-pool-sets")
	require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created pool set")

	testhelpers.MustDelete(t, env.AdminClient, "/service-pool-sets", created.ID)
	testhelpers.AssertGone(t, env.AdminClient, "/service-pool-sets", created.ID)
}
