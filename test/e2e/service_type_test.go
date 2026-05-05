//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/stretchr/testify/require"
)

func testServiceType(t *testing.T, env *Env) {
	lifecycle := domain.LifecycleSchema{
		States: []domain.LifecycleState{
			{Name: "new"}, {Name: "creating"}, {Name: "created"},
		},
		Actions: []domain.LifecycleAction{
			{
				Name: "create",
				Transitions: []domain.LifecycleTransition{
					{From: "new", To: "creating"},
					{From: "creating", To: "created"},
				},
			},
		},
		InitialState:   "new",
		TerminalStates: []string{"created"},
	}

	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "st-" + uniq()
		created := mustPost[api.CreateServiceTypeReq, api.ServiceTypeRes](t, env.AdminClient, "/service-types", api.CreateServiceTypeReq{
			Name: name,
			PropertySchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"region": {Type: "string", Label: "Region"},
				},
			},
			LifecycleSchema: lifecycle,
		})
		require.Equal(t, name, created.Name)

		got := mustGet[api.ServiceTypeRes](t, env.AdminClient, "/service-types", created.ID)
		require.Equal(t, created.ID, got.ID)

		newName := "st-renamed-" + uniq()
		updated := mustPatch[api.UpdateServiceTypeReq, api.ServiceTypeRes](t, env.AdminClient, "/service-types", created.ID, api.UpdateServiceTypeReq{Name: &newName})
		require.Equal(t, newName, updated.Name)

		page := mustList[api.ServiceTypeRes](t, env.AdminClient, "/service-types")
		require.GreaterOrEqual(t, page.TotalItems, int64(2))

		mustDelete(t, env.AdminClient, "/service-types", created.ID)
	})

	t.Run("participant cannot create service type", func(t *testing.T) {
		resp, err := env.ParticipantClient.R().
			SetBody(api.CreateServiceTypeReq{
				Name: "p-" + uniq(),
				PropertySchema: schema.Schema{
					Properties: map[string]schema.PropertyDefinition{
						"region": {Type: "string"},
					},
				},
				LifecycleSchema: lifecycle,
			}).
			Post("/service-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
