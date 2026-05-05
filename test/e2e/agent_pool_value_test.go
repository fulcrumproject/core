//go:build e2e

package e2e

import (
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/stretchr/testify/require"
)

func testAgentPoolValue(t *testing.T, env *Env) {
	name := "apv-" + uniq()
	created := mustPost[api.CreateAgentPoolValueReq, api.AgentPoolValueRes](t, env.AdminClient, "/agent-pool-values", api.CreateAgentPoolValueReq{
		Name:        name,
		Value:       "10.0.0.1",
		AgentPoolID: env.Seed.AgentPool.ID,
	})
	require.Equal(t, env.Seed.AgentPool.ID, created.AgentPoolID)
	require.Equal(t, name, created.Name)

	got := mustGet[api.AgentPoolValueRes](t, env.AdminClient, "/agent-pool-values", created.ID)
	require.Equal(t, created.ID, got.ID)

	page := mustList[api.AgentPoolValueRes](t, env.AdminClient, "/agent-pool-values")
	require.GreaterOrEqual(t, page.TotalItems, int64(2))

	mustDelete(t, env.AdminClient, "/agent-pool-values", created.ID)
}
