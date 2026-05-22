//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

// testServiceCreateFailureScenario covers two failure paths on /jobs/{id}/fail
// for a create job:
//
//   1. Lifecycle DOES define an OnError transition into a terminal state.
//      Service transitions to creationFailed and any pool-allocated property
//      values are released back to the pool.
//
//   2. Lifecycle does NOT define an error transition for the action.
//      Fail must still succeed (HTTP 200), the job is marked Failed, and the
//      service stays in its current state so the operator can retry/delete.
//      (Pool values stay allocated because the service is not in a terminal state.)
func testServiceCreateFailureScenario(t *testing.T, env *Env) {
	t.Run("OnError into terminal: releases pool values", func(t *testing.T) {
		testCreateFailureTerminalReleasesPool(t, env)
	})
	t.Run("no error transition: Fail still 200, service unchanged, pool retained", func(t *testing.T) {
		testCreateFailureNoTransitionTolerant(t, env)
	})
}

func testCreateFailureTerminalReleasesPool(t *testing.T, env *Env) {
	const publicIPValue = "185.123.45.21"

	poolType := "public_ip_fail_" + testhelpers.Uniq()

	pool := testhelpers.MustPost[api.CreateServicePoolReq, api.ServicePoolRes](t, env.AdminClient, "/service-pools", api.CreateServicePoolReq{
		Name:             "fail-terminal-pool-" + testhelpers.Uniq(),
		Type:             poolType,
		PropertyType:     "string",
		GeneratorType:    domain.PoolGeneratorList,
		ServicePoolSetID: env.Seed.ServicePoolSet.ID,
	})

	poolValue := testhelpers.MustPost[api.CreateServicePoolValueReq, api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", api.CreateServicePoolValueReq{
		ServicePoolID: pool.ID,
		Name:          publicIPValue,
		Value:         publicIPValue,
	})

	svcType := testhelpers.MustPost[api.CreateServiceTypeReq, api.ServiceTypeRes](t, env.AdminClient, "/service-types", api.CreateServiceTypeReq{
		Name: "vm-fail-terminal-" + testhelpers.Uniq(),
		PropertySchema: schema.Schema{
			Properties: map[string]schema.PropertyDefinition{
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
				{Name: "creating"}, {Name: "created"}, {Name: "creationFailed"}, {Name: "deleted"},
			},
			Actions: []domain.LifecycleAction{
				{Name: "create", Transitions: []domain.LifecycleTransition{
					{From: "creating", To: "created"},
					{From: "creating", To: "creationFailed", OnError: true},
				}},
				{Name: "delete", Transitions: []domain.LifecycleTransition{
					{From: "created", To: "deleted"},
				}},
			},
			InitialState:   "creating",
			TerminalStates: []string{"deleted", "creationFailed"},
		},
	})

	wireServiceTypeToAgentType(t, env, svcType.ID)

	group := testhelpers.MustPost[api.CreateServiceGroupReq, api.ServiceGroupRes](t, env.AdminClient, "/service-groups", api.CreateServiceGroupReq{
		Name:       "g-fail-terminal-" + testhelpers.Uniq(),
		ConsumerID: env.Seed.Consumer.ID,
	})

	agentID := env.Seed.Agent.ID
	svc := testhelpers.MustPost[api.CreateServiceReq, api.ServiceRes](t, env.AdminClient, "/services", api.CreateServiceReq{
		GroupID:       group.ID,
		AgentID:       &agentID,
		ServiceTypeID: svcType.ID,
		Name:          "svc-fail-terminal-" + testhelpers.Uniq(),
		Properties:    properties.JSON{},
	})
	require.Equal(t, "creating", svc.Status)
	require.NotNil(t, svc.Properties)
	require.Equalf(t, publicIPValue, (*svc.Properties)["publicIp"], "publicIp must be allocated from pool on create: %v", *svc.Properties)

	gotPoolValueAfterCreate := testhelpers.MustGet[api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", poolValue.ID)
	require.NotNilf(t, gotPoolValueAfterCreate.ServiceID, "pool value must be allocated to the service after create")
	require.Equal(t, svc.ID, *gotPoolValueAfterCreate.ServiceID)

	claimAndFail(t, env, svc.ID, "create", "Failed to create VM: simulated error")

	got := testhelpers.MustGet[api.ServiceRes](t, env.AdminClient, "/services", svc.ID)
	require.Equalf(t, "creationFailed", got.Status, "service must transition to creationFailed via OnError, got %s", got.Status)

	gotPoolValueAfterFail := testhelpers.MustGet[api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", poolValue.ID)
	require.Nilf(t, gotPoolValueAfterFail.ServiceID, "pool value must be released after create fails into a terminal state (serviceId still set: %v)", gotPoolValueAfterFail.ServiceID)
}

func testCreateFailureNoTransitionTolerant(t *testing.T, env *Env) {
	const publicIPValue = "185.123.45.22"

	poolType := "public_ip_fail_no_transition_" + testhelpers.Uniq()

	pool := testhelpers.MustPost[api.CreateServicePoolReq, api.ServicePoolRes](t, env.AdminClient, "/service-pools", api.CreateServicePoolReq{
		Name:             "fail-no-transition-pool-" + testhelpers.Uniq(),
		Type:             poolType,
		PropertyType:     "string",
		GeneratorType:    domain.PoolGeneratorList,
		ServicePoolSetID: env.Seed.ServicePoolSet.ID,
	})

	poolValue := testhelpers.MustPost[api.CreateServicePoolValueReq, api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", api.CreateServicePoolValueReq{
		ServicePoolID: pool.ID,
		Name:          publicIPValue,
		Value:         publicIPValue,
	})

	// Lifecycle has no OnError transition for create.
	svcType := testhelpers.MustPost[api.CreateServiceTypeReq, api.ServiceTypeRes](t, env.AdminClient, "/service-types", api.CreateServiceTypeReq{
		Name: "vm-fail-no-transition-" + testhelpers.Uniq(),
		PropertySchema: schema.Schema{
			Properties: map[string]schema.PropertyDefinition{
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
				{Name: "creating"}, {Name: "created"}, {Name: "deleted"},
			},
			Actions: []domain.LifecycleAction{
				{Name: "create", Transitions: []domain.LifecycleTransition{
					{From: "creating", To: "created"},
				}},
				{Name: "delete", Transitions: []domain.LifecycleTransition{
					{From: "created", To: "deleted"},
				}},
			},
			InitialState:   "creating",
			TerminalStates: []string{"deleted"},
		},
	})

	wireServiceTypeToAgentType(t, env, svcType.ID)

	group := testhelpers.MustPost[api.CreateServiceGroupReq, api.ServiceGroupRes](t, env.AdminClient, "/service-groups", api.CreateServiceGroupReq{
		Name:       "g-fail-no-transition-" + testhelpers.Uniq(),
		ConsumerID: env.Seed.Consumer.ID,
	})

	agentID := env.Seed.Agent.ID
	svc := testhelpers.MustPost[api.CreateServiceReq, api.ServiceRes](t, env.AdminClient, "/services", api.CreateServiceReq{
		GroupID:       group.ID,
		AgentID:       &agentID,
		ServiceTypeID: svcType.ID,
		Name:          "svc-fail-no-transition-" + testhelpers.Uniq(),
		Properties:    properties.JSON{},
	})
	require.Equal(t, "creating", svc.Status)

	jobID := claimAndFail(t, env, svc.ID, "create", "Failed to create VM: simulated error")

	got := testhelpers.MustGet[api.ServiceRes](t, env.AdminClient, "/services", svc.ID)
	require.Equalf(t, "creating", got.Status, "service must stay in current state when lifecycle has no error transition, got %s", got.Status)

	jobAfter := testhelpers.MustGet[api.JobRes](t, env.AdminClient, "/jobs", jobID)
	require.Equal(t, domain.JobFailed, jobAfter.Status, "job must be marked Failed even when service state is unchanged")

	gotPoolValue := testhelpers.MustGet[api.ServicePoolValueRes](t, env.AdminClient, "/service-pool-values", poolValue.ID)
	require.NotNilf(t, gotPoolValue.ServiceID, "pool value must remain allocated when service is not in a terminal state")
}

// wireServiceTypeToAgentType appends svcTypeID to the seeded AgentType's service-type
// list so the seeded Agent will serve jobs for it. Restores the original list on cleanup.
func wireServiceTypeToAgentType(t *testing.T, env *Env, svcTypeID properties.UUID) {
	t.Helper()
	originalSvcTypeIDs := []properties.UUID{env.Seed.ServiceType.ID}
	mergedSvcTypeIDs := append([]properties.UUID{env.Seed.ServiceType.ID}, svcTypeID)
	testhelpers.MustPatch[api.UpdateAgentTypeReq, api.AgentTypeRes](t, env.AdminClient, "/agent-types", env.Seed.AgentType.ID, api.UpdateAgentTypeReq{
		ServiceTypeIds: &mergedSvcTypeIDs,
	})
	t.Cleanup(func() {
		_, _ = env.AdminClient.R().
			SetPathParam("id", env.Seed.AgentType.ID.String()).
			SetBody(api.UpdateAgentTypeReq{ServiceTypeIds: &originalSvcTypeIDs}).
			Patch("/agent-types/{id}")
	})
}

// claimAndFail picks the pending job for svcID, asserts its action, claims it,
// then posts /jobs/{id}/fail with errorMessage. Returns the job ID.
func claimAndFail(t *testing.T, env *Env, svcID properties.UUID, expectedAction, errorMessage string) properties.UUID {
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
		SetBody(api.FailJobReq{ErrorMessage: errorMessage}).
		Post("/jobs/{id}/fail")
	require.NoError(t, err)
	require.Equalf(t, http.StatusNoContent, resp.StatusCode(), "fail %s job: %s", expectedAction, resp.String())

	return job.ID
}
