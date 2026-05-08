//go:build e2e

package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testHealth(t *testing.T, env *Env) {
	client := &http.Client{Timeout: 2 * time.Second}

	t.Run("/healthz returns 200 (liveness)", func(t *testing.T) {
		resp, err := client.Get(env.HealthURL + "/healthz")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("/ready returns 200 (readiness)", func(t *testing.T) {
		resp, err := client.Get(env.HealthURL + "/ready")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
