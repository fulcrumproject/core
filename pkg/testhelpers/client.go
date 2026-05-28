package testhelpers

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

func NewClient(serverURL, authToken string) *resty.Client {
	return resty.New().
		SetBaseURL(serverURL + "/api/v1").
		SetAuthToken(authToken)
}

// Uniq returns 8 hex chars suitable as a unique suffix in test fixture names.
// Uses the random/clock-seq tail of a UUIDv7, not the timestamp prefix — back-to-back
// calls within the same millisecond would otherwise collide.
func Uniq() string {
	s := properties.NewUUID().String()
	return s[len(s)-8:]
}

func MustGet[T any](t *testing.T, c *resty.Client, path string, id properties.UUID) *T {
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

func MustPost[TReq any, TRes any](t *testing.T, c *resty.Client, path string, req TReq) *TRes {
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

func MustPatch[TReq any, TRes any](t *testing.T, c *resty.Client, path string, id properties.UUID, req TReq) *TRes {
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

func MustList[T any](t *testing.T, c *resty.Client, path string) *api.PageRes[T] {
	t.Helper()
	var out api.PageRes[T]
	resp, err := c.R().SetResult(&out).Get(path)
	require.NoError(t, err)
	require.Equalf(t, http.StatusOK, resp.StatusCode(), "list %s: %s", path, resp.String())
	return &out
}

func MustDelete(t *testing.T, c *resty.Client, path string, id properties.UUID) {
	t.Helper()
	resp, err := c.R().
		SetPathParam("id", id.String()).
		Delete(path + "/{id}")
	require.NoError(t, err)
	require.Equalf(t, http.StatusNoContent, resp.StatusCode(), "delete %s: %s", path, resp.String())
}

// AssertGone asserts the row at path/{id} is no longer accessible. Accepts
// either 404 or 403: this API's authz middleware can't determine scope on a
// missing row and returns 403 with a "resource not found" body — semantically
// "gone from your perspective". Either status proves the delete happened.
func AssertGone(t *testing.T, c *resty.Client, path string, id properties.UUID) {
	t.Helper()
	resp, err := c.R().
		SetPathParam("id", id.String()).
		Get(path + "/{id}")
	require.NoError(t, err)
	code := resp.StatusCode()
	require.Truef(t,
		code == http.StatusNotFound || code == http.StatusForbidden,
		"GET %s/%s after delete: expected 404 or 403, got %d: %s", path, id, code, resp.String())
	if code == http.StatusForbidden {
		require.Containsf(t, resp.String(), "not found",
			"GET %s/%s returned 403 but body lacks 'not found' marker: %s", path, id, resp.String())
	}
}

// ContainsID reports whether items contains a *Res whose top-level ID field
// equals id. All e2e *Res types declare ID as properties.UUID at struct top
// level (no shared base type), so a reflect-based extractor keeps call sites
// flat across entity tests without forcing a GetID() method on each Res.
func ContainsID[T any](items []*T, id properties.UUID) bool {
	for _, it := range items {
		v := reflect.ValueOf(it).Elem().FieldByName("ID")
		if v.IsValid() {
			if got, ok := v.Interface().(properties.UUID); ok && got == id {
				return true
			}
		}
	}
	return false
}
