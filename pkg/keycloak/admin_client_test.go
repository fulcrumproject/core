package keycloak

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRealm        = "test-realm"
	testClientID     = "test-client"
	testClientSecret = "test-secret"
)

// setupTestClient creates an AdminClient backed by the given handler.
// The handler receives all requests (both token and admin API).
func setupTestClient(t *testing.T, handler http.Handler) *AdminClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg := &Config{
		KeycloakURL:  server.URL,
		Realm:        testRealm,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	}
	return NewAdminClient(cfg)
}

// tokenHandler responds to token requests with a valid access token.
func tokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AdminToken{AccessToken: "test-token", ExpiresIn: 300})
}

// newMux creates a mux with the token endpoint pre-registered.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/"+testRealm+"/protocol/openid-connect/token", tokenHandler)
	return mux
}

// jsonResponse writes a JSON response with the correct Content-Type header.
func jsonResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- ensureToken ---

func TestEnsureToken_CachesToken(t *testing.T) {
	tokenCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/"+testRealm+"/protocol/openid-connect/token", func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AdminToken{AccessToken: "cached-token", ExpiresIn: 300})
	})

	mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/u1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/u2", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	client := setupTestClient(t, mux)
	ctx := context.Background()

	// Two calls should only request one token
	_ = client.Delete(ctx, "u1")
	_ = client.Delete(ctx, "u2")

	assert.Equal(t, 1, tokenCalls, "token should be cached across requests")
}

func TestEnsureToken_FailureReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/"+testRealm+"/protocol/openid-connect/token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid credentials"))
	})

	client := setupTestClient(t, mux)
	err := client.Delete(context.Background(), "user-123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token request failed")
}

