//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

func mustCreateInstallToken(t *testing.T, c *resty.Client, agentID properties.UUID) *api.InstallTokenRes {
	t.Helper()
	var out api.InstallTokenRes
	resp, err := c.R().
		SetPathParam("id", agentID.String()).
		SetResult(&out).
		Post("/agents/{id}/install-command")
	require.NoError(t, err)
	require.Equalf(t, http.StatusCreated, resp.StatusCode(), "create install command: %s", resp.String())
	return &out
}

func mustCreateInfraInstallToken(t *testing.T, c *resty.Client, infraID properties.UUID) *api.InstallTokenRes {
	t.Helper()
	var out api.InstallTokenRes
	resp, err := c.R().
		SetPathParam("id", infraID.String()).
		SetResult(&out).
		Post("/infrastructures/{id}/install-command")
	require.NoError(t, err)
	require.Equalf(t, http.StatusCreated, resp.StatusCode(), "create install command: %s", resp.String())
	return &out
}
