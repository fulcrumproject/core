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

func uniq() string { return properties.NewUUID().String()[:8] }

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

func mustDelete(t *testing.T, c *resty.Client, path string, id properties.UUID) {
	t.Helper()
	resp, err := c.R().
		SetPathParam("id", id.String()).
		Delete(path + "/{id}")
	require.NoError(t, err)
	require.Equalf(t, http.StatusNoContent, resp.StatusCode(), "delete %s: %s", path, resp.String())
}

func mustGet[T any](t *testing.T, c *resty.Client, path string, id properties.UUID) *T {
	t.Helper()
	var out T
	resp, err := c.R().
		SetPathParam("id", id.String()).
		SetResult(&out).
		Get(path + "/{id}")
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode(), "get %s: %s", path, resp.String())
	return &out
}

func mustPost[TReq any, TRes any](t *testing.T, c *resty.Client, path string, req TReq) *TRes {
	t.Helper()
	var out TRes
	resp, err := c.R().
		SetBody(req).
		SetResult(&out).
		Post(path)
	require.NoError(t, err)
	require.Equalf(t, http.StatusCreated, resp.StatusCode(), "create %s: %s", path, resp.String())
	return &out
}

func mustPatch[TReq any, TRes any](t *testing.T, c *resty.Client, path string, id properties.UUID, req TReq) *TRes {
	t.Helper()
	var out TRes
	resp, err := c.R().
		SetPathParam("id", id.String()).
		SetBody(req).
		SetResult(&out).
		Patch(path + "/{id}")
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode(), "patch %s: %s", path, resp.String())
	return &out
}

func mustList[T any](t *testing.T, c *resty.Client, path string) *api.PageRes[T] {
	t.Helper()
	var out api.PageRes[T]
	resp, err := c.R().SetResult(&out).Get(path)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode(), "list %s: %s", path, resp.String())
	return &out
}
