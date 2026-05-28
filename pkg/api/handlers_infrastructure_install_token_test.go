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

func TestBuildInfrastructureInstallURL(t *testing.T) {
	tests := []struct {
		base, token, want string
	}{
		{"http://localhost:8080", "abc", "http://localhost:8080/api/v1/infrastructures/install/abc/config"},
		{"http://localhost:8080/", "abc", "http://localhost:8080/api/v1/infrastructures/install/abc/config"},
		{"https://fulcrum.example.com///", "xyz", "https://fulcrum.example.com/api/v1/infrastructures/install/xyz/config"},
	}
	for _, tc := range tests {
		got := buildInfrastructureInstallURL(tc.base, tc.token)
		if got != tc.want {
			t.Errorf("buildInfrastructureInstallURL(%q,%q) = %q; want %q", tc.base, tc.token, got, tc.want)
		}
	}
}

// TestInfrastructureInstallSubPath_RouteAndURLAgree mirrors the agent-side
// route/URL agreement test: chi must resolve what buildInfrastructureInstallURL
// renders, so the URL builder and the route share one template.
func TestInfrastructureInstallSubPath_RouteAndURLAgree(t *testing.T) {
	r := chi.NewRouter()
	r.Get(InstallConfigSubPath, func(w http.ResponseWriter, r *http.Request) {})

	url := buildInfrastructureInstallURL("http://x", "the-token")
	path := strings.TrimPrefix(url, "http://x/api/v1/infrastructures")

	rctx := chi.NewRouteContext()
	if !r.Match(rctx, "GET", path) {
		t.Fatalf("chi.Match failed for %q", path)
	}
	if got := rctx.URLParam("token"); got != "the-token" {
		t.Errorf("token param: got %q; want %q", got, "the-token")
	}
}

// TestInfrastructureInstallFetch_RejectsAgentToken is the symmetric guard:
// a hashed-token lookup that returns an agent-bound row must 404 on the
// infrastructure mount.
func TestInfrastructureInstallFetch_RejectsAgentToken(t *testing.T) {
	querier := domain.NewMockInstallTokenQuerier(t)
	agentTok := &domain.InstallToken{
		EntityType: domain.InstallTokenEntityTypeAgent,
		EntityID:   properties.UUID(uuid.New()),
	}
	querier.EXPECT().FindByHashedToken(mock.Anything, mock.Anything).Return(agentTok, nil).Once()

	h := &InfrastructureInstallTokenHandler{querier: querier}

	r := chi.NewRouter()
	r.Get(InstallConfigSubPath, h.Fetch)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/install/some-token/config", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}
