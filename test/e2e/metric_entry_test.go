//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/stretchr/testify/require"
)

func testMetricEntry(t *testing.T, env *Env) {
	svcID := env.Seed.Service.ID
	resourceID := "cpu-" + uniq()

	t.Run("agent creates metric entry for service", func(t *testing.T) {
		entry := mustPost[api.CreateMetricEntryReq, api.MetricEntryRes](t, env.AgentClient, "/metric-entries", api.CreateMetricEntryReq{
			ServiceID:    &svcID,
			ResourceID:   resourceID,
			Value:        42.5,
			TypeName:     env.Seed.MetricType.Name,
			MetricTypeID: env.Seed.MetricType.ID,
		})
		require.Equal(t, svcID, entry.ServiceID)
		require.Equal(t, resourceID, entry.ResourceID)
		require.InDelta(t, 42.5, entry.Value, 1e-9)
	})

	t.Run("admin lists entries", func(t *testing.T) {
		page := mustList[api.MetricEntryRes](t, env.AdminClient, "/metric-entries")
		require.GreaterOrEqual(t, page.TotalItems, int64(1))
	})

	t.Run("/resource-ids returns the seeded resource", func(t *testing.T) {
		page := mustList[string](t, env.AdminClient, "/metric-entries/resource-ids")
		require.GreaterOrEqual(t, page.TotalItems, int64(1))
	})

	t.Run("/aggregate returns 200 for valid query", func(t *testing.T) {
		// Use a wide window so we definitely catch the entry created above.
		end := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
		start := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
		resp, err := env.AdminClient.R().
			SetPathParam("serviceId", svcID.String()).
			SetPathParam("resourceId", resourceID).
			SetPathParam("typeId", env.Seed.MetricType.ID.String()).
			SetQueryParams(map[string]string{
				"aggregateType": "max",
				"bucket":        "hour",
				"start":         start,
				"end":           end,
			}).
			Get("/metric-entries/aggregate/{serviceId}/{resourceId}/{typeId}")
		require.NoError(t, err)
		require.Equalf(t, http.StatusOK, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("create without serviceId or agentInstanceId is rejected", func(t *testing.T) {
		// Use AgentClient so we hit the 400 (missing identifiers), not the 403
		// from authz blocking non-agent creates.
		resp, err := env.AgentClient.R().
			SetBody(api.CreateMetricEntryReq{
				ResourceID:   resourceID,
				Value:        1.0,
				TypeName:     env.Seed.MetricType.Name,
				MetricTypeID: env.Seed.MetricType.ID,
			}).
			Post("/metric-entries")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", resp.String())
	})
}
