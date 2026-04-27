package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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
		got := buildInstallURL(tc.base, tc.token)
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

	url := buildInstallURL("http://x", "the-token")
	path := strings.TrimPrefix(url, "http://x/api/v1/agents")

	rctx := chi.NewRouteContext()
	if !r.Match(rctx, "GET", path) {
		t.Fatalf("chi.Match failed for %q", path)
	}
	if got := rctx.URLParam("token"); got != "the-token" {
		t.Errorf("token param: got %q; want %q", got, "the-token")
	}
}
