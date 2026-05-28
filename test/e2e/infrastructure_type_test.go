//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func testInfrastructureType(t *testing.T, env *Env) {
	t.Run("admin creates, gets, updates, lists, deletes", func(t *testing.T) {
		name := "it-" + testhelpers.Uniq()
		cfgTpl := "endpoint={{.endpoint}}\n"
		cmdTpl := "curl -fsSL {{.configUrl}} -H 'Authorization: Bearer {{.authToken}}'"
		created := testhelpers.MustPost[api.CreateInfrastructureTypeReq, api.InfrastructureTypeRes](t, env.AdminClient, "/infrastructure-types", api.CreateInfrastructureTypeReq{
			Name: name,
			ConfigurationSchema: schema.Schema{
				Properties: map[string]schema.PropertyDefinition{
					"endpoint": {Type: "string", Label: "Endpoint", Required: true},
				},
			},
			ConfigTemplate: cfgTpl,
			CmdTemplate:    cmdTpl,
		})
		require.Equal(t, name, created.Name)
		require.Equal(t, cfgTpl, created.ConfigTemplate)
		require.Equal(t, cmdTpl, created.CmdTemplate)
		// ConfigContentType omitted in request → server defaults to text/plain.
		require.Equal(t, "text/plain", created.ConfigContentType)
		require.NotEqual(t, properties.UUID{}, created.ID)
		require.False(t, time.Time(created.CreatedAt).IsZero())

		got := testhelpers.MustGet[api.InfrastructureTypeRes](t, env.AdminClient, "/infrastructure-types", created.ID)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, created.Name, got.Name)
		require.Equal(t, created.ConfigTemplate, got.ConfigTemplate)
		require.Equal(t, created.CmdTemplate, got.CmdTemplate)
		require.Equal(t, created.ConfigContentType, got.ConfigContentType)

		newName := "it-renamed-" + testhelpers.Uniq()
		newContentType := "text/yaml"
		updated := testhelpers.MustPatch[api.UpdateInfrastructureTypeReq, api.InfrastructureTypeRes](t, env.AdminClient, "/infrastructure-types", created.ID, api.UpdateInfrastructureTypeReq{
			Name:              &newName,
			ConfigContentType: &newContentType,
		})
		require.Equal(t, newName, updated.Name)
		require.Equal(t, "text/yaml", updated.ConfigContentType)
		require.Equal(t, created.ID, updated.ID)
		require.Equal(t, created.ConfigTemplate, updated.ConfigTemplate, "PATCH must not touch unprovided fields")
		require.Equal(t, created.CmdTemplate, updated.CmdTemplate, "PATCH must not touch unprovided fields")

		page := testhelpers.MustList[api.InfrastructureTypeRes](t, env.AdminClient, "/infrastructure-types")
		require.True(t, testhelpers.ContainsID(page.Items, created.ID), "list must include just-created infrastructure type")

		testhelpers.MustDelete(t, env.AdminClient, "/infrastructure-types", created.ID)
		testhelpers.AssertGone(t, env.AdminClient, "/infrastructure-types", created.ID)
	})

	t.Run("rejects configTemplate referencing unknown schema field", func(t *testing.T) {
		resp, err := env.AdminClient.R().
			SetBody(api.CreateInfrastructureTypeReq{
				Name: "bad-cfg-ref-" + testhelpers.Uniq(),
				ConfigurationSchema: schema.Schema{
					Properties: map[string]schema.PropertyDefinition{
						"endpoint": {Type: "string", Required: true},
					},
				},
				ConfigTemplate: "endpoint={{.missing}}",
				CmdTemplate:    "curl {{.configUrl}} -H 'Authorization: Bearer {{.authToken}}'",
			}).
			Post("/infrastructure-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("rejects configTemplate set with empty cmdTemplate", func(t *testing.T) {
		resp, err := env.AdminClient.R().
			SetBody(api.CreateInfrastructureTypeReq{
				Name: "no-cmd-" + testhelpers.Uniq(),
				ConfigurationSchema: schema.Schema{
					Properties: map[string]schema.PropertyDefinition{
						"endpoint": {Type: "string"},
					},
				},
				ConfigTemplate: "endpoint={{.endpoint}}",
				// cmdTemplate intentionally empty → template-coupling rule fires
			}).
			Post("/infrastructure-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", resp.String())
	})

	t.Run("participant cannot create infrastructure type", func(t *testing.T) {
		resp, err := env.ProviderClient.R().
			SetBody(api.CreateInfrastructureTypeReq{
				Name:                "p-" + testhelpers.Uniq(),
				ConfigurationSchema: schema.Schema{},
			}).
			Post("/infrastructure-types")
		require.NoError(t, err)
		require.Equalf(t, http.StatusForbidden, resp.StatusCode(), "body: %s", resp.String())
	})
}
