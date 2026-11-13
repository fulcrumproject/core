// Tests administrative bootstrap across real Core process restarts on one DB.
// Native token APIs secure and revoke credentials without resurrecting defaults.

//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/database"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/testhelpers"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBootstrapRestart(t *testing.T) {
	tdb := database.NewTestDB(t)
	t.Cleanup(func() { tdb.Cleanup(t) })
	db := tdb.DB.Session(&gorm.Session{NewDB: true})
	t.Logf("same persistent owned database for every restart: %s", tdb.DBName)
	start := func(t *testing.T) string {
		t.Helper()
		url, _ := testhelpers.Start(t, testhelpers.Config{
			BinaryPath: "./cmd/fulcrum", DB: tdb, Cover: true,
			ExtraEnv: []string{
				"FULCRUM_AUTHENTICATORS=token",
				"FULCRUM_METRIC_DB_DSN=" + tdb.DSN(),
				"FULCRUM_SCHEDULER_LOCKER_DB_DSN=" + tdb.DSN(),
			},
		})
		return url
	}
	denied := func(t *testing.T, url, credential string) {
		t.Helper()
		client := testhelpers.NewClient(url, credential)
		defer client.Close()
		resp, err := client.R().Get("/tokens")
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, resp.StatusCode())
	}
	snapshot := func(t *testing.T) []domain.Token {
		t.Helper()
		var tokens []domain.Token
		require.NoError(t, db.Order("id").Find(&tokens).Error)
		return tokens
	}
	var secured, spare *api.TokenRes
	var before []domain.Token

	// Each subtest's registered cleanup stops its process before the next boot.
	if !t.Run("fresh setup and supported regeneration", func(t *testing.T) {
		url := start(t)
		initial := testhelpers.NewClient(url, "change-me")
		defer initial.Close()
		page := testhelpers.MustList[api.TokenRes](t, initial, "/tokens")
		require.Len(t, page.Items, 1)
		var regenerated api.TokenRes
		resp, err := initial.R().SetResult(&regenerated).Post("/tokens/" + page.Items[0].ID.String() + "/regenerate")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.NotEmpty(t, regenerated.Value)
		require.NotEqual(t, "change-me", regenerated.Value)
		require.Equal(t, page.Items[0].ID, regenerated.ID)
		denied(t, url, "change-me")
		client := testhelpers.NewClient(url, regenerated.Value)
		defer client.Close()
		name, expiry := "secured first credential", time.Now().UTC().Add(48*time.Hour).Truncate(time.Second)
		secured = testhelpers.MustPatch[api.UpdateTokenReq, api.TokenRes](t, client, "/tokens", regenerated.ID, api.UpdateTokenReq{Name: &name, ExpireAt: &expiry})
		secured.Value = regenerated.Value
		spare = testhelpers.MustPost[api.CreateTokenReq, api.TokenRes](t, client, "/tokens", api.CreateTokenReq{Name: "owned spare credential", Role: auth.RoleAdmin})
		before = snapshot(t)
		require.Len(t, before, 2)
		t.Log("fresh bootstrap: one credential; regenerated default denied; name/expiry updated; spare created")
	}) {
		return
	}
	if !t.Run("restart preserves secured credentials then revoke initial", func(t *testing.T) {
		url := start(t)
		require.Equal(t, before, snapshot(t), "restart must preserve IDs, hashes, names, expiry and timestamps")
		denied(t, url, "change-me")
		client := testhelpers.NewClient(url, secured.Value)
		defer client.Close()
		got := testhelpers.MustGet[api.TokenRes](t, client, "/tokens", secured.ID)
		require.Equal(t, secured.Name, got.Name)
		require.Equal(t, secured.ExpireAt, got.ExpireAt)
		testhelpers.MustDelete(t, client, "/tokens", secured.ID)
		denied(t, url, secured.Value)
		before = snapshot(t)
		require.Len(t, before, 1)
		t.Log("restart: default denied; regenerated credential preserved; initial credential revoked through DELETE")
	}) {
		return
	}
	if !t.Run("restart after initial revocation then revoke all", func(t *testing.T) {
		url := start(t)
		require.Equal(t, before, snapshot(t))
		denied(t, url, "change-me")
		denied(t, url, secured.Value)
		client := testhelpers.NewClient(url, spare.Value)
		defer client.Close()
		testhelpers.MustGet[api.TokenRes](t, client, "/tokens", spare.ID)
		testhelpers.MustDelete(t, client, "/tokens", spare.ID)
		denied(t, url, spare.Value)
		require.Empty(t, snapshot(t))
		t.Log("restart after initial revocation: no default; spare preserved then all credentials revoked")
	}) {
		return
	}
	t.Run("restart with no credentials does not recover access", func(t *testing.T) {
		url := start(t)
		denied(t, url, "change-me")
		denied(t, url, secured.Value)
		denied(t, url, spare.Value)
		require.Empty(t, snapshot(t))
		var guards, deletions int64
		require.NoError(t, db.Table("bootstrap_states").Count(&guards).Error)
		require.EqualValues(t, 1, guards)
		require.NoError(t, db.Model(&domain.Event{}).Where("type = ?", domain.EventTypeTokenDeleted).Count(&deletions).Error)
		require.EqualValues(t, 2, deletions)
		t.Log("restart after total revocation: zero tokens, durable guard, two native deletion events; every former credential denied")
	})
}
