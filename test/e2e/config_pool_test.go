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

func testConfigPool(t *testing.T, env *Env) {
	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "ap-" + testhelpers.Uniq()
		typeVal := "type_" + testhelpers.Uniq()
		created := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.AdminClient, "/config-pools", api.CreateConfigPoolReq{
			Name:          name,
			Type:          typeVal,
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, typeVal, created.Type)
		require.Equal(t, "string", created.PropertyType)
		require.Equal(t, domain.PoolGeneratorList, created.GeneratorType)
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := testhelpers.MustGet[api.ConfigPoolRes](t, env.AdminClient, "/config-pools", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.Type, got.Type)
		require.Equal(t, created.PropertyType, got.PropertyType)
		require.Equal(t, created.GeneratorType, got.GeneratorType)

		newName := "ap-renamed-" + testhelpers.Uniq()
		updated := testhelpers.MustPatch[api.UpdateConfigPoolReq, api.ConfigPoolRes](t, env.AdminClient, "/config-pools", created.ID, api.UpdateConfigPoolReq{Name: &newName})
		require.Equal(t, newName, updated.Name)
		require.Equal(t, created.ID, updated.ID)
		require.Equal(t, created.Type, updated.Type, "PATCH name-only must not change type")
		require.Equal(t, created.PropertyType, updated.PropertyType, "PATCH name-only must not change propertyType")
		require.Equal(t, created.GeneratorType, updated.GeneratorType, "PATCH name-only must not change generatorType")

		page := testhelpers.MustList[api.ConfigPoolRes](t, env.AdminClient, "/config-pools")
		require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created config pool")

		testhelpers.MustDelete(t, env.AdminClient, "/config-pools", created.ID)
		testhelpers.AssertGone(t, env.AdminClient, "/config-pools", created.ID)
	})

	t.Run("participant cannot create global pool", func(t *testing.T) {
		// nil ParticipantID = "global" — only admin can create one.
		resp, err := env.ProviderClient.R().
			SetBody(api.CreateConfigPoolReq{
				Name:          "p-" + testhelpers.Uniq(),
				Type:          "type_" + testhelpers.Uniq(),
				PropertyType:  "string",
				GeneratorType: domain.PoolGeneratorList,
			}).
			Post("/config-pools")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("participant cannot create pool owned by another participant", func(t *testing.T) {
		consumerID := testhelpers.ConsumerID
		resp, err := env.ProviderClient.R().
			SetBody(api.CreateConfigPoolReq{
				Name:          "p-" + testhelpers.Uniq(),
				Type:          "type_" + testhelpers.Uniq(),
				PropertyType:  "string",
				GeneratorType: domain.PoolGeneratorList,
				ParticipantID: &consumerID,
			}).
			Post("/config-pools")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("participant creates, gets, deletes pool owned by self", func(t *testing.T) {
		providerID := testhelpers.ProviderID
		name := "ap-own-" + testhelpers.Uniq()
		typeVal := "type_own_" + testhelpers.Uniq()
		created := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
			Name:          name,
			Type:          typeVal,
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
			ParticipantID: &providerID,
		})
		require.Equal(t, name, created.Name)
		require.NotNil(t, created.ParticipantID)
		require.Equal(t, providerID, *created.ParticipantID)

		// Participant can GET own pool.
		got := testhelpers.MustGet[api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", created.ID)
		require.Equal(t, created.ID, got.ID)

		testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", created.ID)
		testhelpers.AssertGone(t, env.ProviderClient, "/config-pools", created.ID)
	})

	t.Run("two participants can have a pool with the same type", func(t *testing.T) {
		providerID := testhelpers.ProviderID
		consumerID := testhelpers.ConsumerID
		sharedType := "type_shared_" + testhelpers.Uniq()

		// Admin creates one for the consumer (admin may create on any participant's behalf).
		consumerPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.AdminClient, "/config-pools", api.CreateConfigPoolReq{
			Name:          "consumer-pool-" + testhelpers.Uniq(),
			Type:          sharedType,
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
			ParticipantID: &consumerID,
		})

		// Provider creates their own with the same Type — must succeed.
		providerPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
			Name:          "provider-pool-" + testhelpers.Uniq(),
			Type:          sharedType,
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
			ParticipantID: &providerID,
		})
		require.NotEqual(t, consumerPool.ID, providerPool.ID)

		t.Cleanup(func() {
			testhelpers.MustDelete(t, env.AdminClient, "/config-pools", providerPool.ID)
			testhelpers.MustDelete(t, env.AdminClient, "/config-pools", consumerPool.ID)
		})

		// Provider cannot see the consumer's pool.
		resp, err := env.ProviderClient.R().Get("/config-pools/" + consumerPool.ID.String())
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "provider must not see other participant's pool. body: %s", resp.String())
	})
}
