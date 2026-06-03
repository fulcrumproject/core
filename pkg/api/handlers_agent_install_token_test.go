package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBuildInstallURL(t *testing.T) {
	tests := []struct {
		base, token, want string
	}{
		{"http://localhost:8080", "abc", "http://localhost:8080/api/v1/agents/install/abc/config"},
		{"http://localhost:8080/", "abc", "http://localhost:8080/api/v1/agents/install/abc/config"},
		{"https://fulcrum.example.com///", "xyz", "https://fulcrum.example.com/api/v1/agents/install/xyz/config"},
	}
	for _, tc := range tests {
		got := buildInstallURL(agentInstallConfigFullPath, tc.base, tc.token)
		if got != tc.want {
			t.Errorf("buildInstallURL(%q,%q) = %q; want %q", tc.base, tc.token, got, tc.want)
		}
	}
}

// TestInstallConfigSubPath_RouteAndURLAgree guards against the two consumers
// of the install-config path drifting apart. If chi can resolve the URL that
// buildInstallURL renders, the route and the URL builder share the same
// template — exactly what InstallConfigSubPath promises.
func TestInstallConfigSubPath_RouteAndURLAgree(t *testing.T) {
	r := chi.NewRouter()
	r.Get(InstallConfigSubPath, func(w http.ResponseWriter, r *http.Request) {})

	url := buildInstallURL(agentInstallConfigFullPath, "http://x", "the-token")
	path := strings.TrimPrefix(url, "http://x/api/v1/agents")

	rctx := chi.NewRouteContext()
	if !r.Match(rctx, "GET", path) {
		t.Fatalf("chi.Match failed for %q", path)
	}
	if got := rctx.URLParam("token"); got != "the-token" {
		t.Errorf("token param: got %q; want %q", got, "the-token")
	}
}

// TestAgentInstallFetch_RejectsInfrastructureToken keeps the /agents/install
// mount strictly typed: a hashed-token lookup that returns an
// infrastructure-bound row must 404, exactly as it does for an unknown token.
// Mirrors the cross-entity guard in the infrastructure handler below.
func TestAgentInstallFetch_RejectsInfrastructureToken(t *testing.T) {
	querier := domain.NewMockInstallTokenQuerier(t)
	infraTok := &domain.InstallToken{
		EntityType: domain.InstallTokenEntityTypeInfrastructure,
		EntityID:   properties.UUID(uuid.New()),
	}
	querier.EXPECT().FindByHashedToken(mock.Anything, mock.Anything).Return(infraTok, nil).Once()

	h := &AgentInstallTokenHandler{querier: querier}

	r := chi.NewRouter()
	r.Get(InstallConfigSubPath, h.Fetch)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/install/some-token/config", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}
