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

func testServicePoolValue(t *testing.T, env *Env) {
	name := "spv-" + testhelpers.Uniq()
	created := testhelpers.MustPost[api.CreateServicePoolValueReq, api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", api.CreateServicePoolValueReq{
		Name:          name,
		Value:         "10.0.0.1",
		ServicePoolID: env.Seed.ServicePool.ID,
	})
	require.Equal(t, name, created.Name)
	require.Equal(t, "10.0.0.1", created.Value)
	require.Equal(t, env.Seed.ServicePool.ID, created.ServicePoolID)
	require.NotEqual(t, properties.UUID{}, created.ID)
	require.False(t, time.Time(created.CreatedAt).IsZero())

	got := testhelpers.MustGet[api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", created.ID)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Name, got.Name)
	require.Equal(t, created.Value, got.Value)
	require.Equal(t, created.ServicePoolID, got.ServicePoolID)

	page := testhelpers.MustList[api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values")
	require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created pool value")

	testhelpers.MustDelete(t, env.AdminClient, "/service-pool-values", created.ID)
	testhelpers.AssertGone(t, env.AdminClient, "/service-pool-values", created.ID)
}
