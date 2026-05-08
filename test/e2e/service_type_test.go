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

func testServiceType(t *testing.T, env *Env) {
	lifecycle := domain.LifecycleSchema{
		States: []domain.LifecycleState{
			{Name: "new"}, {Name: "creating"}, {Name: "created"},
		},
		Actions: []domain.LifecycleAction{
			{
				Name: "create",
				Transitions: []domain.LifecycleTransition{
					{From: "new", To: "creating"},
					{From: "creating", To: "created"},
				},
			},
		},
		InitialState:   "new",
		TerminalStates: []string{"created"},
	}

	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "st-" + testhelpers.Uniq()
		created := testhelpers.MustPost[api.CreateServiceTypeReq, api.ServiceTypeRes](t, env.AdminClient, "/service-types", api.CreateServiceTypeReq{
			Name: name,
			PropertySchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"region": {Type: "string", Label: "Region"},
				},
			},
			LifecycleSchema: lifecycle,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, "new", created.LifecycleSchema.InitialState, "lifecycle round-trips")
		require.Contains(t, created.PropertySchema.Properties, "region", "property schema round-trips")
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := testhelpers.MustGet[api.ServiceTypeRes](t, env.AdminClient, "/service-types", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.LifecycleSchema, got.LifecycleSchema)
		require.Equal(t, created.PropertySchema, got.PropertySchema)

		newName := "st-renamed-" + testhelpers.Uniq()
		updated := testhelpers.MustPatch[api.UpdateServiceTypeReq, api.ServiceTypeRes](t, env.AdminClient, "/service-types", created.ID, api.UpdateServiceTypeReq{Name: &newName})
		require.Equal(t, newName, updated.Name)
		require.Equal(t, created.ID, updated.ID)
		require.Equal(t, created.LifecycleSchema, updated.LifecycleSchema, "PATCH name-only must not change lifecycle")

		page := testhelpers.MustList[api.ServiceTypeRes](t, env.AdminClient, "/service-types")
		require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created service type")

		testhelpers.MustDelete(t, env.AdminClient, "/service-types", created.ID)
		testhelpers.AssertGone(t, env.AdminClient, "/service-types", created.ID)
	})

	t.Run("participant cannot create service type", func(t *testing.T) {
		resp, err := env.ProviderClient.R().
			SetBody(api.CreateServiceTypeReq{
				Name: "p-" + testhelpers.Uniq(),
				PropertySchema: schema.Schema{
					Properties: map[string]schema.PropertyDefinition{
						"region": {Type: "string"},
					},
				},
				LifecycleSchema: lifecycle,
			}).
			Post("/service-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
