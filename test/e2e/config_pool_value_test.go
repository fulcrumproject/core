//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
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

	t.Run("agent under a provider allocates value from global pool", func(t *testing.T) {
		// Seed an extra value into the global ConfigPool so this subtest doesn't
		// race the seeded one (which other subtests/scenarios may have consumed).
		val := testhelpers.MustPost[api.CreateConfigPoolValueReq, api.ConfigPoolValueRes](t, env.AdminClient, "/config-pool-values", api.CreateConfigPoolValueReq{
			Name:         "global-ip-" + testhelpers.Uniq(),
			Value:        "10.0.99." + testhelpers.Uniq()[:2],
			ConfigPoolID: env.Seed.ConfigPool.ID,
		})

		// Fresh AgentType whose Configuration schema auto-allocates from the
		// seeded global ConfigPool (type "internalIp", no participantId).
		at := testhelpers.MustPost[api.CreateAgentTypeReq, api.AgentTypeRes](t, env.AdminClient, "/agent-types", api.CreateAgentTypeReq{
			Name: "at-pool-global-" + testhelpers.Uniq(),
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"poolIp": {
						Type:  "string",
						Label: "Pool IP",
						Generator: &schema.GeneratorConfig{
							Type:   "pool",
							Config: map[string]any{"poolType": env.Seed.ConfigPool.Type},
						},
					},
				},
			},
		})
		t.Cleanup(func() { testhelpers.MustDelete(t, env.AdminClient, "/agent-types", at.ID) })

		cfg := properties.JSON{}
		agent := testhelpers.MustPost[api.CreateAgentReq, api.AgentRes](t, env.AdminClient, "/agents", api.CreateAgentReq{
			Name:          "agent-pool-" + testhelpers.Uniq(),
			ProviderID:    env.Seed.Provider.ID,
			AgentTypeID:   at.ID,
			Configuration: &cfg,
		})
		t.Cleanup(func() { testhelpers.MustDelete(t, env.AdminClient, "/agents", agent.ID) })

		require.NotNil(t, agent.Configuration, "agent must surface its resolved Configuration")
		got := (*agent.Configuration)["poolIp"]
		require.NotEmptyf(t, got, "poolIp must be allocated from the global pool: %v", *agent.Configuration)

		// The allocated value must correspond to one of the global pool's values
		// (the seeded one or the one we just added).
		require.Containsf(t, []any{val.Value, "10.0.0.10"}, got, "allocated value %v not from global pool", got)
	})

	t.Run("agent under a provider allocates value from provider-owned pool", func(t *testing.T) {
		providerID := testhelpers.ProviderID
		poolType := "provider-pool-" + testhelpers.Uniq()

		providerPool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
			Name:          "own-" + poolType,
			Type:          poolType,
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
			ParticipantID: &providerID,
		})
		providerValue := testhelpers.MustPost[api.CreateConfigPoolValueReq, api.ConfigPoolValueRes](t, env.ProviderClient, "/config-pool-values", api.CreateConfigPoolValueReq{
			Name:         "p-" + testhelpers.Uniq(),
			Value:        "192.168.99.5",
			ConfigPoolID: providerPool.ID,
		})

		at := testhelpers.MustPost[api.CreateAgentTypeReq, api.AgentTypeRes](t, env.AdminClient, "/agent-types", api.CreateAgentTypeReq{
			Name: "at-pool-provider-" + testhelpers.Uniq(),
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"poolIp": {
						Type:  "string",
						Label: "Pool IP",
						Generator: &schema.GeneratorConfig{
							Type:   "pool",
							Config: map[string]any{"poolType": poolType},
						},
					},
				},
			},
		})
		t.Cleanup(func() { testhelpers.MustDelete(t, env.AdminClient, "/agent-types", at.ID) })

		cfg := properties.JSON{}
		agent := testhelpers.MustPost[api.CreateAgentReq, api.AgentRes](t, env.AdminClient, "/agents", api.CreateAgentReq{
			Name:          "agent-pool-provider-" + testhelpers.Uniq(),
			ProviderID:    providerID,
			AgentTypeID:   at.ID,
			Configuration: &cfg,
		})
		t.Cleanup(func() {
			testhelpers.MustDelete(t, env.AdminClient, "/agents", agent.ID)
			testhelpers.MustDelete(t, env.ProviderClient, "/config-pool-values", providerValue.ID)
			testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", providerPool.ID)
		})

		require.NotNil(t, agent.Configuration)
		require.Equalf(t, providerValue.Value, (*agent.Configuration)["poolIp"],
			"agent must allocate from provider-owned pool (got %v)", (*agent.Configuration)["poolIp"])
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

	t.Run("participant cannot add value to another participant's pool", func(t *testing.T) {
		providerID := testhelpers.ProviderID

		// Provider creates a pool they own.
		pool := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
			Name:          "p-cross-" + testhelpers.Uniq(),
			Type:          "type_cross_" + testhelpers.Uniq(),
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
			ParticipantID: &providerID,
		})
		t.Cleanup(func() { testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", pool.ID) })

		// Consumer (different participant) tries to POST a value into the provider's pool — must 403.
		resp, err := env.ConsumerClient.R().
			SetBody(api.CreateConfigPoolValueReq{
				Name:         "v-cross-" + testhelpers.Uniq(),
				Value:        "10.0.0.99",
				ConfigPoolID: pool.ID,
			}).
			Post("/config-pool-values")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "consumer must not POST a value into provider's pool. body: %s", resp.String())
	})

	t.Run("schema allocation is isolated between two providers with same-typed pools", func(t *testing.T) {
		providerID := testhelpers.ProviderID
		consumerID := testhelpers.ConsumerID
		sharedType := "iso-type-" + testhelpers.Uniq()

		// Provider-owned pool of type T + its value.
		poolA := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.ProviderClient, "/config-pools", api.CreateConfigPoolReq{
			Name:          "iso-prov-" + testhelpers.Uniq(),
			Type:          sharedType,
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
			ParticipantID: &providerID,
		})
		valueA := testhelpers.MustPost[api.CreateConfigPoolValueReq, api.ConfigPoolValueRes](t, env.ProviderClient, "/config-pool-values", api.CreateConfigPoolValueReq{
			Name:         "a-" + testhelpers.Uniq(),
			Value:        "10.0.0.1",
			ConfigPoolID: poolA.ID,
		})

		// Consumer-owned pool of the same type T + its value.
		poolB := testhelpers.MustPost[api.CreateConfigPoolReq, api.ConfigPoolRes](t, env.AdminClient, "/config-pools", api.CreateConfigPoolReq{
			Name:          "iso-cons-" + testhelpers.Uniq(),
			Type:          sharedType,
			PropertyType:  "string",
			GeneratorType: domain.PoolGeneratorList,
			ParticipantID: &consumerID,
		})
		valueB := testhelpers.MustPost[api.CreateConfigPoolValueReq, api.ConfigPoolValueRes](t, env.AdminClient, "/config-pool-values", api.CreateConfigPoolValueReq{
			Name:         "b-" + testhelpers.Uniq(),
			Value:        "10.0.0.2",
			ConfigPoolID: poolB.ID,
		})

		at := testhelpers.MustPost[api.CreateAgentTypeReq, api.AgentTypeRes](t, env.AdminClient, "/agent-types", api.CreateAgentTypeReq{
			Name: "at-iso-" + testhelpers.Uniq(),
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"poolIp": {
						Type:  "string",
						Label: "Pool IP",
						Generator: &schema.GeneratorConfig{
							Type:   "pool",
							Config: map[string]any{"poolType": sharedType},
						},
					},
				},
			},
		})
		t.Cleanup(func() { testhelpers.MustDelete(t, env.AdminClient, "/agent-types", at.ID) })

		cfg := properties.JSON{}
		agentA := testhelpers.MustPost[api.CreateAgentReq, api.AgentRes](t, env.AdminClient, "/agents", api.CreateAgentReq{
			Name:          "iso-agent-a-" + testhelpers.Uniq(),
			ProviderID:    providerID,
			AgentTypeID:   at.ID,
			Configuration: &cfg,
		})
		agentB := testhelpers.MustPost[api.CreateAgentReq, api.AgentRes](t, env.AdminClient, "/agents", api.CreateAgentReq{
			Name:          "iso-agent-b-" + testhelpers.Uniq(),
			ProviderID:    consumerID,
			AgentTypeID:   at.ID,
			Configuration: &cfg,
		})
		t.Cleanup(func() {
			testhelpers.MustDelete(t, env.AdminClient, "/agents", agentA.ID)
			testhelpers.MustDelete(t, env.AdminClient, "/agents", agentB.ID)
			testhelpers.MustDelete(t, env.ProviderClient, "/config-pool-values", valueA.ID)
			testhelpers.MustDelete(t, env.ProviderClient, "/config-pools", poolA.ID)
			testhelpers.MustDelete(t, env.AdminClient, "/config-pool-values", valueB.ID)
			testhelpers.MustDelete(t, env.AdminClient, "/config-pools", poolB.ID)
		})

		require.NotNil(t, agentA.Configuration)
		require.Equalf(t, valueA.Value, (*agentA.Configuration)["poolIp"],
			"provider's agent must allocate from provider's pool, not consumer's (got %v)", (*agentA.Configuration)["poolIp"])

		require.NotNil(t, agentB.Configuration)
		require.Equalf(t, valueB.Value, (*agentB.Configuration)["poolIp"],
			"consumer's agent must allocate from consumer's pool, not provider's (got %v)", (*agentB.Configuration)["poolIp"])
	})
}
