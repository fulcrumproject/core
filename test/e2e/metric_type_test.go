//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/stretchr/testify/require"
)

func testMetricType(t *testing.T, env *Env) {
	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "mt-" + uniq()
		created := mustPost[api.CreateMetricTypeReq, api.MetricTypeRes](t, env.AdminClient, "/metric-types", api.CreateMetricTypeReq{
			Name:       name,
			EntityType: domain.MetricEntityTypeService,
		})
		require.Equal(t, name, created.Name)

		got := mustGet[api.MetricTypeRes](t, env.AdminClient, "/metric-types", created.ID)
		require.Equal(t, created.ID, got.ID)

		newName := "mt-renamed-" + uniq()
		updated := mustPatch[api.UpdateMetricTypeReq, api.MetricTypeRes](t, env.AdminClient, "/metric-types", created.ID, api.UpdateMetricTypeReq{Name: &newName})
		require.Equal(t, newName, updated.Name)

		page := mustList[api.MetricTypeRes](t, env.AdminClient, "/metric-types")
		require.GreaterOrEqual(t, page.TotalItems, int64(2))

		mustDelete(t, env.AdminClient, "/metric-types", created.ID)
	})

	t.Run("participant cannot create metric type", func(t *testing.T) {
		resp, err := env.ParticipantClient.R().
			SetBody(api.CreateMetricTypeReq{
				Name:       "p-" + uniq(),
				EntityType: domain.MetricEntityTypeService,
			}).
			Post("/metric-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
