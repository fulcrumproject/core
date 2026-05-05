//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
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

		got := mustGet[api.ParticipantRes](t, env.AdminClient, "/participants", created.ID)
		require.Equal(t, created.ID, got.ID)

		newName := "p-renamed-" + uniq()
		var updated api.ParticipantRes
		resp, err := env.AdminClient.R().
			SetPathParam("id", created.ID.String()).
			SetBody(api.UpdateParticipantReq{Name: &newName}).
			SetResult(&updated).
			Patch("/participants/{id}")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.Equal(t, newName, updated.Name)

		var page api.PageRes[api.ParticipantRes]
		resp, err = env.AdminClient.R().SetResult(&page).Get("/participants")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.GreaterOrEqual(t, page.TotalItems, int64(3), "seed (2) + new (1)")

		mustDelete(t, env.AdminClient, "/participants", created.ID)
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
		resp, err := env.ParticipantClient.R().
			SetPathParam("id", env.Seed.Consumer.ID.String()).
			Get("/participants/{id}")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
