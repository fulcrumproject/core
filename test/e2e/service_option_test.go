//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
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
		require.Equal(t, env.Seed.Provider.ID, created.ProviderID)
		require.True(t, created.Enabled)

		got := mustGet[api.ServiceOptionRes](t, env.AdminClient, "/service-options", created.ID)
		require.Equal(t, created.ID, got.ID)

		newName := "opt-renamed-" + uniq()
		updated := mustPatch[api.UpdateServiceOptionReq, api.ServiceOptionRes](t, env.AdminClient, "/service-options", created.ID, api.UpdateServiceOptionReq{Name: &newName})
		require.Equal(t, newName, updated.Name)

		mustDelete(t, env.AdminClient, "/service-options", created.ID)
	})

	t.Run("participant cannot create option for another provider", func(t *testing.T) {
		// participant1 is mapped to Provider; creating an option scoped to
		// Consumer must 403 since it's not in their participant scope.
		resp, err := env.ParticipantClient.R().
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
