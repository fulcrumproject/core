//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/stretchr/testify/require"
)

// testServiceLifecycleScenario drives a service through create → start → stop
// → cold update → delete with a real ServiceType that exercises:
//   - pool-generated publicIp (auto-allocated on create, released on delete)
//   - agent-set internalIp on job completion
//   - state-authorized cpu (only mutable while Stopped)
//
// The scenario builds its own ServiceType + ServicePool because the seeded
// fixtures (seed.go) are intentionally minimal and don't model these contracts.
func testServiceLifecycleScenario(t *testing.T, env *Env) {
	const (
		publicIPValue = "185.123.45.11"
		internalIP    = "10.0.0.42"
	)

	poolType := "public_ip_" + uniq()

	pool := mustPost[api.CreateServicePoolReq, api.ServicePoolRes](t, env.AdminClient, "/service-pools", api.CreateServicePoolReq{
		Name:             "lifecycle-pool-" + uniq(),
		Type:             poolType,
		PropertyType:     "string",
		GeneratorType:    domain.PoolGeneratorList,
		ServicePoolSetID: env.Seed.PoolSet.ID,
	})

	poolValue := mustPost[api.CreateServicePoolValueReq, api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", api.CreateServicePoolValueReq{
		ServicePoolID: pool.ID,
		Name:          publicIPValue,
		Value:         publicIPValue,
	})

	svcType := mustPost[api.CreateServiceTypeReq, api.ServiceTypeRes](t, env.AdminClient, "/service-types", api.CreateServiceTypeReq{
		Name: "vm-lifecycle-" + uniq(),
		PropertySchema: schema.Schema{
			Properties: map[string]schema.PropertyDefinition{
				"cpu": {
					Type:     "integer",
					Label:    "CPU Cores",
					Required: true,
					Authorizers: []schema.AuthorizerConfig{
						{Type: "state", Config: map[string]any{"allowedStates": []string{"Stopped"}}},
					},
					Validators: []schema.ValidatorConfig{
						{Type: "enum", Config: map[string]any{"values": []int{1, 2, 4, 8, 16, 32}}},
					},
				},
				"image": {
					Type:      "string",
					Label:     "Image",
					Required:  true,
					Immutable: true,
				},
				"internalIp": {
					Type:      "string",
					Label:     "Internal IP",
					Immutable: true,
					Authorizers: []schema.AuthorizerConfig{
						{Type: "actor", Config: map[string]any{"actors": []string{"agent"}}},
					},
				},
				"publicIp": {
					Type:      "string",
					Label:     "Public IP",
					Immutable: true,
					Authorizers: []schema.AuthorizerConfig{
						{Type: "actor", Config: map[string]any{"actors": []string{"system"}}},
					},
					Generator: &schema.GeneratorConfig{
						Type:   "pool",
						Config: map[string]any{"poolType": poolType},
					},
				},
			},
		},
		LifecycleSchema: domain.LifecycleSchema{
			States: []domain.LifecycleState{
				{Name: "New"}, {Name: "Stopped"}, {Name: "Started"}, {Name: "Deleted"},
			},
			Actions: []domain.LifecycleAction{
				{Name: "create", RequestSchemaType: "properties", Transitions: []domain.LifecycleTransition{{From: "New", To: "Stopped"}}},
				{Name: "start", Transitions: []domain.LifecycleTransition{{From: "Stopped", To: "Started"}}},
				{Name: "stop", Transitions: []domain.LifecycleTransition{{From: "Started", To: "Stopped"}}},
				{Name: "update", RequestSchemaType: "properties", Transitions: []domain.LifecycleTransition{{From: "Stopped", To: "Stopped"}}},
				{Name: "delete", Transitions: []domain.LifecycleTransition{
					{From: "Stopped", To: "Deleted"},
					{From: "Started", To: "Deleted"},
				}},
			},
			InitialState:   "New",
			TerminalStates: []string{"Deleted"},
			RunningStates:  []string{"Started"},
		},
	})

	// Link the new ServiceType to the seeded AgentType so the seeded Agent
	// can serve jobs for services of this type.
	originalSvcTypeIDs := []properties.UUID{env.Seed.ServiceType.ID}
	mergedSvcTypeIDs := append([]properties.UUID{env.Seed.ServiceType.ID}, svcType.ID)
	mustPatch[api.UpdateAgentTypeReq, api.AgentTypeRes](t, env.AdminClient, "/agent-types", env.Seed.AgentType.ID, api.UpdateAgentTypeReq{
		ServiceTypeIds: &mergedSvcTypeIDs,
	})
	t.Cleanup(func() {
		_, _ = env.AdminClient.R().
			SetPathParam("id", env.Seed.AgentType.ID.String()).
			SetBody(api.UpdateAgentTypeReq{ServiceTypeIds: &originalSvcTypeIDs}).
			Patch("/agent-types/{id}")
	})

	// Dedicated group: /jobs/pending returns one job per group.
	group := mustPost[api.CreateServiceGroupReq, api.ServiceGroupRes](t, env.AdminClient, "/service-groups", api.CreateServiceGroupReq{
		Name:       "g-svc-lifecycle-" + uniq(),
		ConsumerID: env.Seed.Consumer.ID,
	})

	agentID := env.Seed.Agent.ID
	svc := mustPost[api.CreateServiceReq, api.ServiceRes](t, env.AdminClient, "/services", api.CreateServiceReq{
		GroupID:       group.ID,
		AgentID:       &agentID,
		ServiceTypeID: svcType.ID,
		Name:          "svc-lifecycle-" + uniq(),
		Properties: properties.JSON{
			"cpu":   4,
			"image": "ubuntu:20.04",
		},
	})
	require.Equal(t, "New", svc.Status, "service starts in InitialState")

	t.Run("create job: pool allocates publicIp, agent sets internalIp", func(t *testing.T) {
		instanceID := "vm-" + uniq()
		ip := internalIP
		claimAndComplete(t, env, svc.ID, "create", &api.CompleteJobReq{
			Properties:      &properties.JSON{"internalIp": ip},
			AgentInstanceID: &instanceID,
		})

		got := mustGet[api.ServiceRes](t, env.AdminClient, "/services", svc.ID)
		require.Equal(t, "Stopped", got.Status, "create transitions New→Stopped")
		require.NotNil(t, got.Properties)
		require.Equalf(t, publicIPValue, (*got.Properties)["publicIp"], "publicIp must be auto-allocated from pool: %v", *got.Properties)
		require.Equalf(t, internalIP, (*got.Properties)["internalIp"], "internalIp must be set by agent on completion: %v", *got.Properties)
	})

	t.Run("start job transitions to Started", func(t *testing.T) {
		actionService(t, env, svc.ID, "start", http.StatusOK)
		claimAndComplete(t, env, svc.ID, "start", &api.CompleteJobReq{})

		got := mustGet[api.ServiceRes](t, env.AdminClient, "/services", svc.ID)
		require.Equal(t, "Started", got.Status)
	})

	t.Run("PATCH cpu while Started is rejected by state authorizer", func(t *testing.T) {
		newCPU := properties.JSON{"cpu": 8}
		resp, err := env.AdminClient.R().
			SetPathParam("id", svc.ID.String()).
			SetBody(api.UpdateServiceReq{Properties: &newCPU}).
			Patch("/services/{id}")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "PATCH cpu while Started must 400: %s", resp.String())
	})

	t.Run("stop job transitions to Stopped", func(t *testing.T) {
		actionService(t, env, svc.ID, "stop", http.StatusOK)
		claimAndComplete(t, env, svc.ID, "stop", &api.CompleteJobReq{})

		got := mustGet[api.ServiceRes](t, env.AdminClient, "/services", svc.ID)
		require.Equal(t, "Stopped", got.Status)
	})

	t.Run("cold update while Stopped enqueues update job and applies new cpu", func(t *testing.T) {
		newProps := properties.JSON{"cpu": 8}
		mustPatch[api.UpdateServiceReq, api.ServiceRes](t, env.AdminClient, "/services", svc.ID, api.UpdateServiceReq{Properties: &newProps})
		claimAndComplete(t, env, svc.ID, "update", &api.CompleteJobReq{})

		got := mustGet[api.ServiceRes](t, env.AdminClient, "/services", svc.ID)
		require.Equal(t, "Stopped", got.Status, "update transitions Stopped→Stopped")
		require.NotNil(t, got.Properties)
		require.EqualValues(t, 8, (*got.Properties)["cpu"], "cold update must persist new cpu: %v", *got.Properties)
	})

	t.Run("delete releases publicIp back to the pool", func(t *testing.T) {
		mustDelete(t, env.AdminClient, "/services", svc.ID)
		claimAndComplete(t, env, svc.ID, "delete", &api.CompleteJobReq{})

		got := mustGet[api.ServiceRes](t, env.AdminClient, "/services", svc.ID)
		require.Equal(t, "Deleted", got.Status)

		gotPoolValue := mustGet[api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", poolValue.ID)
		require.Nilf(t, gotPoolValue.ServiceID, "publicIp pool value must be released after service delete (serviceId still set: %v)", gotPoolValue.ServiceID)
	})
}

func claimAndComplete(t *testing.T, env *Env, svcID properties.UUID, expectedAction string, complete *api.CompleteJobReq) {
	t.Helper()
	var pending []*api.JobRes
	resp, err := env.AgentClient.R().SetResult(&pending).Get("/jobs/pending")
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode(), "/jobs/pending: %s", resp.String())

	job := findJobForService(pending, svcID)
	require.NotNilf(t, job, "no pending job for service %s (looking for action=%s)", svcID, expectedAction)
	require.Equalf(t, expectedAction, job.Action, "expected action=%s, got %s", expectedAction, job.Action)

	resp, err = env.AgentClient.R().
		SetPathParam("id", job.ID.String()).
		Post("/jobs/{id}/claim")
	require.NoError(t, err)
	require.Equalf(t, http.StatusNoContent, resp.StatusCode(), "claim %s job: %s", expectedAction, resp.String())

	resp, err = env.AgentClient.R().
		SetPathParam("id", job.ID.String()).
		SetBody(complete).
		Post("/jobs/{id}/complete")
	require.NoError(t, err)
	require.Equalf(t, http.StatusNoContent, resp.StatusCode(), "complete %s job: %s", expectedAction, resp.String())
}

func actionService(t *testing.T, env *Env, svcID properties.UUID, action string, want int) {
	t.Helper()
	resp, err := env.AdminClient.R().
		SetPathParam("id", svcID.String()).
		SetPathParam("action", action).
		Post("/services/{id}/{action}")
	require.NoError(t, err)
	require.Equalf(t, want, resp.StatusCode(), "POST /services/%s/%s: %s", svcID, action, resp.String())
}
