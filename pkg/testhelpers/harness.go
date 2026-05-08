package testhelpers

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

	"github.com/stretchr/testify/require"
)

// DBSource yields the PostgreSQL DSN for the per-test database. Callers in
// each repo provide an adapter over their own TestDB type.
type DBSource interface {
	DSN() string
}

// Config configures Start. BinaryPath is relative to the consumer module
// root (e.g. "./cmd/fulcrum"). ExtraEnv is appended last and wins on conflict
// with the harness defaults.
type Config struct {
	BinaryPath string
	DB         DBSource
	Realm      Realm
	ExtraEnv   []string
	Cover      bool
}

// Start execs the configured binary against cfg.DB, waits for /ready, and
// registers cleanup. Returns the API base URL and the health server base URL
// (both without trailing slash; the API URL has no /api/v1 prefix).
func Start(t *testing.T, cfg Config) (apiURL, healthURL string) {
	t.Helper()

	apiPort := FreePort(t)
	healthPort := FreePort(t)

	coverDir := os.Getenv("GOCOVERDIR")
	if coverDir == "" {
		coverDir = t.TempDir()
	}
	require.NoError(t, os.MkdirAll(coverDir, 0o755))

	env := append(os.Environ(),
		"FULCRUM_PORT="+strconv.Itoa(apiPort),
		"FULCRUM_HEALTH_PORT="+strconv.Itoa(healthPort),
		"FULCRUM_DB_DSN="+cfg.DB.DSN(),
		"FULCRUM_AUTHENTICATORS=oauth,token",
		"FULCRUM_OAUTH_KEYCLOAK_URL="+cfg.Realm.URL,
		"FULCRUM_OAUTH_REALM="+cfg.Realm.Name,
		"FULCRUM_OAUTH_CLIENT_ID="+cfg.Realm.ClientID,
		"FULCRUM_OAUTH_CLIENT_SECRET="+cfg.Realm.Secret,
		fmt.Sprintf("FULCRUM_PUBLIC_BASE_URL=http://127.0.0.1:%d", apiPort),
		"FULCRUM_API_SERVER=true",
		"FULCRUM_JOB_MAINTENANCE=false",
		"FULCRUM_AGENT_MAINTENANCE=false",
		"GOCOVERDIR="+coverDir,
	)
	env = append(env, cfg.ExtraEnv...)

	args := []string{"run"}
	if cfg.Cover {
		args = append(args, "-cover", "-coverpkg=./...")
	}
	args = append(args, cfg.BinaryPath)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = RepoRoot(t)
	cmd.Env = env
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	require.NoError(t, cmd.Start(), "failed to start %s", cfg.BinaryPath)

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

	WaitReady(t, healthPort, 60*time.Second)
	return fmt.Sprintf("http://127.0.0.1:%d", apiPort),
		fmt.Sprintf("http://127.0.0.1:%d", healthPort)
}

func FreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// RepoRoot walks up from the caller's source file until it finds a go.mod,
// returning that directory. This resolves to the *consumer* module root,
// regardless of where this package sits inside core.
func RepoRoot(t *testing.T) string {
	t.Helper()
	// Skip past testhelpers frames to the test file invoking us.
	for skip := 1; skip < 16; skip++ {
		_, file, _, ok := runtime.Caller(skip)
		if !ok {
			break
		}
		if filepath.Base(filepath.Dir(file)) == "testhelpers" {
			continue
		}
		dir, err := filepath.Abs(filepath.Dir(file))
		require.NoError(t, err)
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	t.Fatalf("testhelpers.RepoRoot: no go.mod found walking up from caller")
	return ""
}

func WaitReady(t *testing.T, healthPort int, timeout time.Duration) {
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
