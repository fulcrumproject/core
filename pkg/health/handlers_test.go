package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockChecker implements the Checker interface for testing
type MockChecker struct {
	healthResult    CheckResult
	readinessResult CheckResult
}

func (m *MockChecker) CheckHealth(ctx context.Context) CheckResult {
	return m.healthResult
}

func (m *MockChecker) CheckReadiness(ctx context.Context) CheckResult {
	return m.readinessResult
}

func TestHealthHandler_UP(t *testing.T) {
	// Setup
	mockChecker := &MockChecker{
		healthResult: CheckResult{Status: StatusUP},
	}
	handler := NewHandler(mockChecker)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.HealthHandler(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "UP", response.Status)
}

func TestHealthHandler_DOWN(t *testing.T) {
	// Setup
	mockChecker := &MockChecker{
		healthResult: CheckResult{
			Status: StatusDOWN,
			Error:  "Database connection failed",
		},
	}
	handler := NewHandler(mockChecker)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.HealthHandler(w, req)

	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "DOWN", response.Status)
}

func TestReadinessHandler_UP(t *testing.T) {
	// Setup
	mockChecker := &MockChecker{
		readinessResult: CheckResult{Status: StatusUP},
	}
	handler := NewHandler(mockChecker)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ReadinessHandler(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "UP", response.Status)
}

func TestReadinessHandler_DOWN(t *testing.T) {
	// Setup
	mockChecker := &MockChecker{
		readinessResult: CheckResult{
			Status: StatusDOWN,
			Error:  "Authentication service unavailable",
		},
	}
	handler := NewHandler(mockChecker)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	// Execute
	handler.ReadinessHandler(w, req)

	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "DOWN", response.Status)
}
