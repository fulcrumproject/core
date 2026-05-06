//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/database"
	"github.com/stretchr/testify/require"
)

const (
	keycloakURL   = "http://localhost:8080"
	keycloakRealm = "fulcrum"
	oauthClientID = "fulcrum-api"
	oauthSecret   = "secret"

	vaultKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, this, _, _ := runtime.Caller(0)
	root, err := filepath.Abs(filepath.Join(filepath.Dir(this), "..", ".."))
	require.NoError(t, err)
	return root
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// startServer execs the fulcrum binary against tdb, waits for /ready, and
// registers cleanup. Returns the API base URL and the health server base URL
// (both without trailing slash; API URL has no /api/v1 prefix).
func startServer(t *testing.T, tdb *database.TestDB) (apiURL, healthURL string) {
	t.Helper()

	apiPort := freePort(t)
	healthPort := freePort(t)

	coverDir := os.Getenv("GOCOVERDIR")
	if coverDir == "" {
		coverDir = t.TempDir()
	}
	require.NoError(t, os.MkdirAll(coverDir, 0o755))

	env := append(os.Environ(),
		"FULCRUM_PORT="+strconv.Itoa(apiPort),
		"FULCRUM_HEALTH_PORT="+strconv.Itoa(healthPort),
		"FULCRUM_DB_DSN="+tdb.DSN,
		"FULCRUM_METRIC_DB_DSN="+tdb.DSN,
		"FULCRUM_SCHEDULER_LOCKER_DB_DSN="+tdb.DSN,
		"FULCRUM_AUTHENTICATORS=oauth,token",
		"FULCRUM_OAUTH_KEYCLOAK_URL="+keycloakURL,
		"FULCRUM_OAUTH_REALM="+keycloakRealm,
		"FULCRUM_OAUTH_CLIENT_ID="+oauthClientID,
		"FULCRUM_OAUTH_CLIENT_SECRET="+oauthSecret,
		fmt.Sprintf("FULCRUM_PUBLIC_BASE_URL=http://127.0.0.1:%d", apiPort),
		"FULCRUM_VAULT_ENCRYPTION_KEY="+vaultKey,
		"FULCRUM_API_SERVER=true",
		"FULCRUM_JOB_MAINTENANCE=false",
		"FULCRUM_AGENT_MAINTENANCE=false",
		"FULCRUM_KEYCLOAK_ADMIN=true",
		"GOCOVERDIR="+coverDir,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "go", "run", "-cover", "-coverpkg=./...", "./cmd/fulcrum")
	cmd.Dir = repoRoot(t)
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	require.NoError(t, cmd.Start(), "failed to start fulcrum binary")

	t.Cleanup(func() {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(20 * time.Second):
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			cancel()
			<-done
		}
	})

	waitReady(t, healthPort, 60*time.Second)
	return fmt.Sprintf("http://127.0.0.1:%d", apiPort),
		fmt.Sprintf("http://127.0.0.1:%d", healthPort)
}

func waitReady(t *testing.T, healthPort int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://127.0.0.1:%d/ready", healthPort)
	client := &http.Client{Timeout: 1 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("server not ready at %s after %s", url, timeout)
}
