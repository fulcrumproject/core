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

func testServiceOptionType(t *testing.T, env *Env) {
	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "sot-" + uniq()
		typeVal := "size_" + uniq()
		created := mustPost[api.CreateServiceOptionTypeReq, api.ServiceOptionTypeRes](t, env.AdminClient, "/service-option-types", api.CreateServiceOptionTypeReq{
			Name:        name,
			Type:        typeVal,
			Description: "test option type",
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, typeVal, created.Type)
		require.Equal(t, "test option type", created.Description)
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := mustGet[api.ServiceOptionTypeRes](t, env.AdminClient, "/service-option-types", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.Type, got.Type)
		require.Equal(t, created.Description, got.Description)

		newDesc := "renamed"
		updated := mustPatch[api.UpdateServiceOptionTypeReq, api.ServiceOptionTypeRes](t, env.AdminClient, "/service-option-types", created.ID, api.UpdateServiceOptionTypeReq{Description: &newDesc})
		require.Equal(t, newDesc, updated.Description)
		require.Equal(t, created.ID, updated.ID)
		require.Equal(t, created.Name, updated.Name, "PATCH description-only must not change name")
		require.Equal(t, created.Type, updated.Type, "PATCH description-only must not change type")

		page := mustList[api.ServiceOptionTypeRes](t, env.AdminClient, "/service-option-types")
		require.True(t, containsID(page.Items, created.ID), "list must include just-created option type")

		mustDelete(t, env.AdminClient, "/service-option-types", created.ID)
		assertGone(t, env.AdminClient, "/service-option-types", created.ID)
	})

	t.Run("participant cannot create option type", func(t *testing.T) {
		resp, err := env.ProviderClient.R().
			SetBody(api.CreateServiceOptionTypeReq{
				Name: "p-" + uniq(),
				Type: "size_" + uniq(),
			}).
			Post("/service-option-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
