//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/require"
)

func testServiceOption(t *testing.T, env *Env) {
	t.Run("admin creates+gets+updates+deletes scoped to provider", func(t *testing.T) {
		enabled := true
		name := "opt-" + uniq()
		created := mustPost[api.CreateServiceOptionReq, api.ServiceOptionRes](t, env.AdminClient, "/service-options", api.CreateServiceOptionReq{
			ProviderID:          env.Seed.Provider.ID,
			ServiceOptionTypeID: env.Seed.OptionType.ID,
			Name:                name,
			Value:               "small",
			Enabled:             &enabled,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, env.Seed.Provider.ID, created.ProviderID)
		require.Equal(t, env.Seed.OptionType.ID, created.ServiceOptionTypeID)
		require.True(t, created.Enabled)
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := mustGet[api.ServiceOptionRes](t, env.AdminClient, "/service-options", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.ProviderID, got.ProviderID)
		require.Equal(t, created.ServiceOptionTypeID, got.ServiceOptionTypeID)
		require.Equal(t, created.Enabled, got.Enabled)

		newName := "opt-renamed-" + uniq()
		updated := mustPatch[api.UpdateServiceOptionReq, api.ServiceOptionRes](t, env.AdminClient, "/service-options", created.ID, api.UpdateServiceOptionReq{Name: &newName})
		require.Equal(t, newName, updated.Name)
		require.Equal(t, created.ID, updated.ID)
		require.Equal(t, created.ProviderID, updated.ProviderID, "PATCH must not change FK")
		require.Equal(t, created.ServiceOptionTypeID, updated.ServiceOptionTypeID, "PATCH must not change FK")
		require.Equal(t, created.Enabled, updated.Enabled, "PATCH name-only must not flip enabled")

		page := mustList[api.ServiceOptionRes](t, env.AdminClient, "/service-options")
		require.True(t, containsID(page.Items, created.ID), "list must include just-created option")

		mustDelete(t, env.AdminClient, "/service-options", created.ID)
		assertGone(t, env.AdminClient, "/service-options", created.ID)
	})

	t.Run("participant cannot create option for another provider", func(t *testing.T) {
		// participant1 is mapped to Provider; creating an option scoped to
		// Consumer must 403 since it's not in their participant scope.
		resp, err := env.ProviderClient.R().
			SetBody(api.CreateServiceOptionReq{
				ProviderID:          env.Seed.Consumer.ID,
				ServiceOptionTypeID: env.Seed.OptionType.ID,
				Name:                "x-scope-" + uniq(),
				Value:               "v",
			}).
			Post("/service-options")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
