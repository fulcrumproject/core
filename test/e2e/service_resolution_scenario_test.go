//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

// testServiceResolutionScenario covers agent resolution when a service is created
// without an explicit agentId: the most-recently-seen online agent supporting the
// service type wins, and creation fails when no supporting agent is online.
//
// A dedicated agent type + service type + agent are used so only this agent can ever
// serve the type, making the resolved choice deterministic regardless of seed state.
func testServiceResolutionScenario(t *testing.T, env *Env) {
	svcType := testhelpers.MustPost[api.CreateServiceTypeReq, api.ServiceTypeRes](t, env.AdminClient, "/service-types", api.CreateServiceTypeReq{
		Name: "vm-resolution-" + testhelpers.Uniq(),
		PropertySchema: schema.Schema{Properties: map[string]schema.PropertyDefinition{
			"hostname": {Type: "string", Label: "Hostname"},
		}},
		LifecycleSchema: domain.LifecycleSchema{
			States: []domain.LifecycleState{
				{Name: "creating"}, {Name: "created"}, {Name: "deleted"},
			},
			Actions: []domain.LifecycleAction{
				{Name: "create", Transitions: []domain.LifecycleTransition{{From: "creating", To: "created"}}},
				{Name: "delete", Transitions: []domain.LifecycleTransition{{From: "created", To: "deleted"}}},
			},
			InitialState:   "creating",
			TerminateState: "deleted",
			TerminalStates: []string{"deleted"},
		},
	})

	// Agent type that supports only the dedicated service type.
	agentType := testhelpers.MustPost[api.CreateAgentTypeReq, api.AgentTypeRes](t, env.AdminClient, "/agent-types", api.CreateAgentTypeReq{
		Name:           "resolution-agent-type-" + testhelpers.Uniq(),
		ServiceTypeIds: []properties.UUID{svcType.ID},
		ConfigurationSchema: schema.Schema{Properties: map[string]schema.PropertyDefinition{
			"endpoint": {Type: "string", Label: "Endpoint"},
		}},
		ConfigContentType: "text/plain",
	})

	agent := testhelpers.MustPost[api.CreateAgentReq, api.AgentRes](t, env.AdminClient, "/agents", api.CreateAgentReq{
		Name:        "resolution-agent-" + testhelpers.Uniq(),
		ProviderID:  env.Seed.Provider.ID,
		AgentTypeID: agentType.ID,
	})

	group := testhelpers.MustPost[api.CreateServiceGroupReq, api.ServiceGroupRes](t, env.AdminClient, "/service-groups", api.CreateServiceGroupReq{
		Name:       "g-resolution-" + testhelpers.Uniq(),
		ConsumerID: env.Seed.Consumer.ID,
	})

	// Agent client so we can drive the agent's own status endpoint.
	expireAt := time.Now().Add(time.Hour)
	token := testhelpers.MustPost[api.CreateTokenReq, api.TokenRes](t, env.AdminClient, "/tokens", api.CreateTokenReq{
		Name:     "resolution-agent-token-" + testhelpers.Uniq(),
		Role:     auth.RoleAgent,
		ScopeID:  &agent.ID,
		ExpireAt: &expireAt,
	})
	agentClient := testhelpers.NewClient(env.ServerURL, token.Value)

	createWithoutAgentID := func() api.CreateServiceReq {
		return api.CreateServiceReq{
			GroupID:       group.ID,
			ServiceTypeID: svcType.ID,
			Name:          "svc-resolution-" + testhelpers.Uniq(),
			Properties:    properties.JSON{},
		}
	}

	t.Run("no online agent: create without agentId fails", func(t *testing.T) {
		// Agent was just created (status New), so no agent supporting the type is online.
		resp, err := env.AdminClient.R().SetBody(createWithoutAgentID()).Post("/services")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "expected 400 with no online agent: %s", resp.String())
		require.Containsf(t, resp.String(), "no online agent found", "unexpected error body: %s", resp.String())
	})

	t.Run("online agent is auto-picked", func(t *testing.T) {
		resp, err := agentClient.R().
			SetBody(api.UpdateAgentStatusReq{Status: domain.AgentConnected}).
			Put("/agents/me/status")
		require.NoError(t, err)
		require.Equalf(t, http.StatusOK, resp.StatusCode(), "connect agent: %s", resp.String())

		svc := testhelpers.MustPost[api.CreateServiceReq, api.ServiceRes](t, env.AdminClient, "/services", createWithoutAgentID())
		require.Equal(t, agent.ID, svc.AgentID, "auto-resolution must pick the only online agent")
		require.Equal(t, "creating", svc.Status)
	})
}
