//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/require"
)

func testJob(t *testing.T, env *Env) {
	// /jobs/pending returns at most one job per service group, so use a
	// dedicated group to keep this suite's job findable.
	group := mustPost[api.CreateServiceGroupReq, api.ServiceGroupRes](t, env.AdminClient, "/service-groups", api.CreateServiceGroupReq{
		Name:       "g-jobs-" + uniq(),
		ConsumerID: env.Seed.Consumer.ID,
	})

	aid := env.Seed.Agent.ID
	svc := mustPost[api.CreateServiceReq, api.ServiceRes](t, env.AdminClient, "/services", api.CreateServiceReq{
		GroupID:       group.ID,
		AgentID:       &aid,
		ServiceTypeID: env.Seed.ServiceType.ID,
		Name:          "svc-job-" + uniq(),
		Properties:    properties.JSON{},
	})

	t.Run("admin lists jobs includes the dispatched job", func(t *testing.T) {
		page := mustList[api.JobRes](t, env.AdminClient, "/jobs")
		require.GreaterOrEqual(t, page.TotalItems, int64(1))
	})

	t.Run("agent /pending shows the create job", func(t *testing.T) {
		var pending []*api.JobRes
		resp, err := env.AgentClient.R().SetResult(&pending).Get("/jobs/pending")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.NotEmpty(t, pending)
		require.NotNil(t, findJobForService(pending, svc.ID), "expected pending job for svc %s", svc.ID)
	})

	t.Run("non-agent client cannot list /pending", func(t *testing.T) {
		resp, err := env.AdminClient.R().Get("/jobs/pending")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("agent claims+completes job → service transitions to created", func(t *testing.T) {
		var pending []*api.JobRes
		_, err := env.AgentClient.R().SetResult(&pending).Get("/jobs/pending")
		require.NoError(t, err)
		job := findJobForService(pending, svc.ID)
		require.NotNil(t, job, "no pending job for svc %s", svc.ID)
		require.Equal(t, "create", job.Action)

		resp, err := env.AgentClient.R().
			SetPathParam("id", job.ID.String()).
			Post("/jobs/{id}/claim")
		require.NoError(t, err)
		require.Equalf(t, http.StatusNoContent, resp.StatusCode(), "claim: %s", resp.String())

		resp, err = env.AgentClient.R().
			SetPathParam("id", job.ID.String()).
			SetBody(api.CompleteJobReq{}).
			Post("/jobs/{id}/complete")
		require.NoError(t, err)
		require.Equalf(t, http.StatusNoContent, resp.StatusCode(), "complete: %s", resp.String())

		after := mustGet[api.ServiceRes](t, env.AdminClient, "/services", svc.ID)
		require.Equal(t, "created", after.Status, "service should advance after job completion")
	})
}

func findJobForService(jobs []*api.JobRes, serviceID properties.UUID) *api.JobRes {
	for _, j := range jobs {
		if j.ServiceID == serviceID {
			return j
		}
	}
	return nil
}
