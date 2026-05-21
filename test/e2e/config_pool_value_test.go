//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func testConfigPoolValue(t *testing.T, env *Env) {
	t.Run("admin crud on global pool", func(t *testing.T) {
		name := "apv-" + testhelpers.Uniq()
		created := testhelpers.MustPost[api.CreateConfigPoolValueReq, api.ConfigPoolValueRes](t, env.AdminClient, "/config-pool-values", api.CreateConfigPoolValueReq{
			Name:         name,
			Value:        "10.0.0.1",
			ConfigPoolID: env.Seed.ConfigPool.ID,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, "10.0.0.1", created.Value)
		require.Equal(t, env.Seed.ConfigPool.ID, created.ConfigPoolID)
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := testhelpers.MustGet[api.ConfigPoolValueRes](t, env.AdminClient, "/config-pool-values", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.Value, got.Value)
		require.Equal(t, created.ConfigPoolID, got.ConfigPoolID)

		page := testhelpers.MustList[api.ConfigPoolValueRes](t, env.AdminClient, "/config-pool-values")
		require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created pool value")

		testhelpers.MustDelete(t, env.AdminClient, "/config-pool-values", created.ID)
		testhelpers.AssertGone(t, env.AdminClient, "/config-pool-values", created.ID)
	})

	t.Run("participant cannot add value to global pool", func(t *testing.T) {
		resp, err := env.ProviderClient.R().
			SetBody(api.CreateConfigPoolValueReq{
				Name:         "v-" + testhelpers.Uniq(),
				Value:        "10.0.0.2",
				ConfigPoolID: env.Seed.ConfigPool.ID, // seeded ConfigPool is global (nil participant)
			}).
			Post("/config-pool-values")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("participant adds value to own pool", func(t *testing.T) {
		providerID := testhelpers.ProviderID

		// Provider creates a pool they own.
		pool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
			Name:          "ap-own-" + testhelpers.Uniq(),
			Type:          "type_own_" + testhelpers.Uniq(),
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
			ParticipantID: &providerID,
		})

		// Provider adds a value to their own pool — succeeds.
		val := testhelpers.MustPost[api.CreateConfigPoolValueReq, api.ConfigPoolValueRes](t, env.ProviderClient, "/config-pool-values", api.CreateConfigPoolValueReq{
			Name:         "v-" + testhelpers.Uniq(),
			Value:        "192.168.1.10",
			ConfigPoolID: pool.ID,
		})
		require.Equal(t, pool.ID, val.ConfigPoolID)

		// Consumer (another participant) cannot read the provider's value.
		resp, err := env.ConsumerClient.R().Get("/config-pool-values/" + val.ID.String())
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "consumer must not see provider's value. body: %s", resp.String())

		// Cleanup (in dependency order).
		testhelpers.MustDelete(t, env.ProviderClient, "/config-pool-values", val.ID)
		testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", pool.ID)
	})
}
