//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/stretchr/testify/require"
)

func testServiceOptionType(t *testing.T, env *Env) {
	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "sot-" + uniq()
		created := mustPost[api.CreateServiceOptionTypeReq, api.ServiceOptionTypeRes](t, env.AdminClient, "/service-option-types", api.CreateServiceOptionTypeReq{
			Name:        name,
			Type:        "size_" + uniq(),
			Description: "test option type",
		})
		require.Equal(t, name, created.Name)

		got := mustGet[api.ServiceOptionTypeRes](t, env.AdminClient, "/service-option-types", created.ID)
		require.Equal(t, created.ID, got.ID)

		newDesc := "renamed"
		updated := mustPatch[api.UpdateServiceOptionTypeReq, api.ServiceOptionTypeRes](t, env.AdminClient, "/service-option-types", created.ID, api.UpdateServiceOptionTypeReq{Description: &newDesc})
		require.Equal(t, newDesc, updated.Description)

		page := mustList[api.ServiceOptionTypeRes](t, env.AdminClient, "/service-option-types")
		require.GreaterOrEqual(t, page.TotalItems, int64(2))

		mustDelete(t, env.AdminClient, "/service-option-types", created.ID)
	})

	t.Run("participant cannot create option type", func(t *testing.T) {
		resp, err := env.ParticipantClient.R().
			SetBody(api.CreateServiceOptionTypeReq{
				Name: "p-" + uniq(),
				Type: "size_" + uniq(),
			}).
			Post("/service-option-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
