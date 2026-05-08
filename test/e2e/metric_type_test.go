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

func testMetricType(t *testing.T, env *Env) {
	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "mt-" + testhelpers.Uniq()
		created := testhelpers.MustPost[api.CreateMetricTypeReq, api.MetricTypeRes](t, env.AdminClient, "/metric-types", api.CreateMetricTypeReq{
			Name:       name,
			EntityType: domain.MetricEntityTypeService,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, domain.MetricEntityTypeService, created.EntityType)
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := testhelpers.MustGet[api.MetricTypeRes](t, env.AdminClient, "/metric-types", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.EntityType, got.EntityType)

		newName := "mt-renamed-" + testhelpers.Uniq()
		updated := testhelpers.MustPatch[api.UpdateMetricTypeReq, api.MetricTypeRes](t, env.AdminClient, "/metric-types", created.ID, api.UpdateMetricTypeReq{Name: &newName})
		require.Equal(t, newName, updated.Name)
		require.Equal(t, created.ID, updated.ID)
		require.Equal(t, created.EntityType, updated.EntityType, "PATCH name-only must not change entityType")

		page := testhelpers.MustList[api.MetricTypeRes](t, env.AdminClient, "/metric-types")
		require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created metric type")

		testhelpers.MustDelete(t, env.AdminClient, "/metric-types", created.ID)
		testhelpers.AssertGone(t, env.AdminClient, "/metric-types", created.ID)
	})

	t.Run("participant cannot create metric type", func(t *testing.T) {
		resp, err := env.ProviderClient.R().
			SetBody(api.CreateMetricTypeReq{
				Name:       "p-" + testhelpers.Uniq(),
				EntityType: domain.MetricEntityTypeService,
			}).
			Post("/metric-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
