//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func testInfrastructure(t *testing.T, env *Env) {
	// Shared InfrastructureType for the subtests below — built once so the
	// configuration round-trips against a known schema.
	mkType := func(t *testing.T) properties.UUID {
		t.Helper()
		it := testhelpers.MustPost[api.CreateInfrastructureTypeReq, api.InfrastructureTypeRes](t, env.AdminClient, "/infrastructure-types", api.CreateInfrastructureTypeReq{
			Name: "it-for-infra-" + testhelpers.Uniq(),
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint":   {Type: "string", Label: "Endpoint", Required: true},
					"maxRetries": {Type: "integer", Label: "Max Retries", Default: 3},
				},
			},
		})
		t.Cleanup(func() { testhelpers.MustDelete(t, env.AdminClient, "/infrastructure-types", it.ID) })
		return it.ID
	}

	t.Run("admin creates, gets, patches, deletes infrastructure", func(t *testing.T) {
		itID := mkType(t)

		name := "infra-" + testhelpers.Uniq()
		cfg := properties.JSON{
			"endpoint":   "https://example.invalid",
			"maxRetries": 5,
		}
		created := testhelpers.MustPost[api.CreateInfrastructureReq, api.InfrastructureRes](t, env.AdminClient, "/infrastructures", api.CreateInfrastructureReq{
			Name:                 name,
			ProviderID:           env.Seed.Provider.ID,
			InfrastructureTypeID: itID,
			Tags:                 []string{"region:eu"},
			Configuration:        &cfg,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, env.Seed.Provider.ID, created.ProviderID)
		require.Equal(t, itID, created.InfrastructureTypeID)
		require.Equal(t, []string{"region:eu"}, created.Tags)
		require.NotNil(t, created.Configuration, "configuration must round-trip")
		require.Equal(t, "https://example.invalid", (*created.Configuration)["endpoint"])
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := testhelpers.MustGet[api.InfrastructureRes](t, env.AdminClient, "/infrastructures", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.InfrastructureTypeID, got.InfrastructureTypeID)
		require.Equal(t, "https://example.invalid", (*got.Configuration)["endpoint"])

		newName := "infra-renamed-" + testhelpers.Uniq()
		newCfg := properties.JSON{
			"endpoint":   "https://renamed.invalid",
			"maxRetries": 7,
		}
		updated := testhelpers.MustPatch[api.UpdateInfrastructureReq, api.InfrastructureRes](t, env.AdminClient, "/infrastructures", created.ID, api.UpdateInfrastructureReq{
			Name:          &newName,
			Configuration: &newCfg,
		})
		require.Equal(t, newName, updated.Name)
		require.Equal(t, "https://renamed.invalid", (*updated.Configuration)["endpoint"])
		// Tags weren't in the PATCH body — must survive untouched.
		require.Equal(t, created.Tags, updated.Tags, "PATCH must not touch unprovided fields")

		page := testhelpers.MustList[api.InfrastructureRes](t, env.AdminClient, "/infrastructures")
		require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created infrastructure")

		testhelpers.MustDelete(t, env.AdminClient, "/infrastructures", created.ID)
		testhelpers.AssertGone(t, env.AdminClient, "/infrastructures", created.ID)
	})

	t.Run("rejects configuration with bad type", func(t *testing.T) {
		itID := mkType(t)

		badCfg := properties.JSON{
			"endpoint":   "https://x",
			"maxRetries": "not-an-integer",
		}
		resp, err := env.AdminClient.R().
			SetBody(api.CreateInfrastructureReq{
				Name:                 "infra-bad-" + testhelpers.Uniq(),
				ProviderID:           env.Seed.Provider.ID,
				InfrastructureTypeID: itID,
				Configuration:        &badCfg,
			}).
			Post("/infrastructures")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("rejects configuration missing required property", func(t *testing.T) {
		itID := mkType(t)

		missingReq := properties.JSON{
			"maxRetries": 5,
		}
		resp, err := env.AdminClient.R().
			SetBody(api.CreateInfrastructureReq{
				Name:                 "infra-miss-" + testhelpers.Uniq(),
				ProviderID:           env.Seed.Provider.ID,
				InfrastructureTypeID: itID,
				Configuration:        &missingReq,
			}).
			Post("/infrastructures")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("delete blocked by dependent agent", func(t *testing.T) {
		itID := mkType(t)

		// AgentType bound to the same IT so the Agent below can attach to the
		// Infrastructure below.
		at := testhelpers.MustPost[api.CreateAgentTypeReq, api.AgentTypeRes](t, env.AdminClient, "/agent-types", api.CreateAgentTypeReq{
			Name:                  "at-infra-block-" + testhelpers.Uniq(),
			InfrastructureTypeIds: []properties.UUID{itID},
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"placeholder": {Type: "string"},
				},
			},
		})
		t.Cleanup(func() { testhelpers.MustDelete(t, env.AdminClient, "/agent-types", at.ID) })

		infra := testhelpers.MustPost[api.CreateInfrastructureReq, api.InfrastructureRes](t, env.AdminClient, "/infrastructures", api.CreateInfrastructureReq{
			Name:                 "infra-blocked-" + testhelpers.Uniq(),
			ProviderID:           env.Seed.Provider.ID,
			InfrastructureTypeID: itID,
			Configuration:        &properties.JSON{"endpoint": "https://x.invalid"},
		})
		t.Cleanup(func() { testhelpers.MustDelete(t, env.AdminClient, "/infrastructures", infra.ID) })

		ag := testhelpers.MustPost[api.CreateAgentReq, api.AgentRes](t, env.AdminClient, "/agents", api.CreateAgentReq{
			Name:             "agent-blocks-infra-" + testhelpers.Uniq(),
			ProviderID:       env.Seed.Provider.ID,
			AgentTypeID:      at.ID,
			InfrastructureID: &infra.ID,
		})
		t.Cleanup(func() { testhelpers.MustDelete(t, env.AdminClient, "/agents", ag.ID) })

		// Block: Infrastructure can't be deleted while the Agent still binds it.
		resp, err := env.AdminClient.R().
			SetPathParam("id", infra.ID.String()).
			Delete("/infrastructures/{id}")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "expected dependency block, body: %s", resp.String())
	})
}
