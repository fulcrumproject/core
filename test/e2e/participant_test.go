//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/require"
)

func testParticipant(t *testing.T, env *Env) {
	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "p-" + uniq()
		created := mustPost[api.CreateParticipantReq, api.ParticipantRes](t, env.AdminClient, "/participants", api.CreateParticipantReq{
			Name:   name,
			Status: domain.ParticipantEnabled,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, domain.ParticipantEnabled, created.Status)
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := mustGet[api.ParticipantRes](t, env.AdminClient, "/participants", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.Status, got.Status)

		newName := "p-renamed-" + uniq()
		updated := mustPatch[api.UpdateParticipantReq, api.ParticipantRes](t, env.AdminClient, "/participants", created.ID, api.UpdateParticipantReq{Name: &newName})
		require.Equal(t, newName, updated.Name)
		require.Equal(t, created.ID, updated.ID)
		require.Equal(t, created.Status, updated.Status, "PATCH must not silently change status")

		page := mustList[api.ParticipantRes](t, env.AdminClient, "/participants")
		require.True(t, containsID(page.Items, created.ID), "list must include just-created participant")

		mustDelete(t, env.AdminClient, "/participants", created.ID)
		assertGone(t, env.AdminClient, "/participants", created.ID)
	})

	t.Run("create rejects invalid status", func(t *testing.T) {
		resp, err := env.AdminClient.R().
			SetBody(api.CreateParticipantReq{Name: "bad-" + uniq(), Status: domain.ParticipantStatus("Bogus")}).
			Post("/participants")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("participant cannot read another participant", func(t *testing.T) {
		// participant1 maps to provider (Seed.Provider). Reading consumer
		// (Seed.Consumer) is out of scope and must 403.
		resp, err := env.ProviderClient.R().
			SetPathParam("id", env.Seed.Consumer.ID.String()).
			Get("/participants/{id}")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
