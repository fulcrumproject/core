package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoints_Integration(t *testing.T) {
	// Setup dependencies with mock authenticators
	healthyAuth := &MockAuthenticator{healthError: nil}
	deps := &PrimaryDependencies{
		DB:             nil, // This will cause health checks to fail
		Authenticators: []auth.Authenticator{healthyAuth},
	}

	// Create health checker and handler
	checker := NewHealthChecker(deps)
	handler := NewHandler(checker)

	// Setup router
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Get("/healthz", handler.HealthHandler)
	r.Get("/ready", handler.ReadinessHandler)

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Health endpoint with unhealthy dependencies",
			endpoint:       "/healthz",
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   `{"status":"DOWN"}`,
		},
		{
			name:           "Readiness endpoint with unhealthy dependencies",
			endpoint:       "/ready",
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   `{"status":"DOWN"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			w := httptest.NewRecorder()

			// Execute
			r.ServeHTTP(w, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Assert response body
			var response Res
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "UP", response.Status)
			} else {
				assert.Equal(t, "DOWN", response.Status)
			}

			// Assert content type
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		})
	}
}

func TestHealthEndpoints_HealthyDependencies(t *testing.T) {
	// This test would require a real database connection for full integration
	// For now, we'll test with healthy authenticators and skip DB check

	healthyAuth := &MockAuthenticator{healthError: nil}
	deps := &PrimaryDependencies{
		DB:             nil, // Still nil, but we can modify the checker for this test
		Authenticators: []auth.Authenticator{healthyAuth},
	}

	// Create a custom checker that skips DB check for this test
	checker := &testHealthChecker{
		deps:        deps,
		skipDBCheck: true,
	}
	handler := NewHandler(checker)

	// Setup router
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Get("/healthz", handler.HealthHandler)
	r.Get("/ready", handler.ReadinessHandler)

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
	}{
		{
			name:           "Health endpoint with healthy dependencies",
			endpoint:       "/healthz",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Readiness endpoint with healthy dependencies",
			endpoint:       "/ready",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			w := httptest.NewRecorder()

			// Execute
			r.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response Res
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "UP", response.Status)
		})
	}
}

// testHealthChecker is a test-specific checker that can skip certain checks
type testHealthChecker struct {
	deps        *PrimaryDependencies
	skipDBCheck bool
}

func (t *testHealthChecker) CheckHealth(ctx context.Context) CheckResult {
	return t.check(ctx)
}

func (t *testHealthChecker) CheckReadiness(ctx context.Context) CheckResult {
	return t.check(ctx)
}

func (t *testHealthChecker) check(ctx context.Context) CheckResult {
	// Skip DB check if configured
	if !t.skipDBCheck {
		if t.deps.DB == nil {
			return CheckResult{
				Status: StatusDOWN,
				Error:  "Database check failed: database connection is nil",
			}
		}
	}

	// Check authenticators
	for i, authenticator := range t.deps.Authenticators {
		if err := authenticator.Health(ctx); err != nil {
			return CheckResult{
				Status: StatusDOWN,
				Error:  fmt.Sprintf("authenticator %d health check failed: %v", i, err),
			}
		}
	}

	return CheckResult{
		Status: StatusUP,
	}
}
