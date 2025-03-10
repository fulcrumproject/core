package config

import (
	"os"
	"testing"
	"time"
)

// TestNestedStructEnvLoading tests that the loadEnvToStruct function can handle nested structs
func TestNestedStructEnvLoading(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("FULCRUM_PORT", "8080")
	os.Setenv("FULCRUM_JOB_MAINTENANCE_INTERVAL", "30m")
	os.Setenv("FULCRUM_AGENT_HEALTH_TIMEOUT", "10m")
	os.Setenv("FULCRUM_DB_HOST", "testhost")
	os.Setenv("FULCRUM_DB_PORT", "5433")

	// Create a config instance
	cfg := DefaultConfig()

	// Load from environment
	err := cfg.LoadFromEnv()
	if err != nil {
		t.Fatalf("Failed to load config from env: %v", err)
	}

	// Verify values were loaded correctly
	if cfg.Port != 8080 {
		t.Errorf("Expected Port to be 8080, got %d", cfg.Port)
	}

	expectedMaintenance := 30 * time.Minute
	if cfg.JobConfig.Maintenance != expectedMaintenance {
		t.Errorf("Expected JobConfig.Maintenance to be %s, got %s", expectedMaintenance, cfg.JobConfig.Maintenance)
	}

	expectedHealthTimeout := 10 * time.Minute
	if cfg.AgentConfig.HealthTimeout != expectedHealthTimeout {
		t.Errorf("Expected AgentConfig.HealthTimeout to be %s, got %s", expectedHealthTimeout, cfg.AgentConfig.HealthTimeout)
	}

	if cfg.DBConfig.Host != "testhost" {
		t.Errorf("Expected DBConfig.Host to be 'testhost', got '%s'", cfg.DBConfig.Host)
	}

	if cfg.DBConfig.Port != 5433 {
		t.Errorf("Expected DBConfig.Port to be 5433, got %d", cfg.DBConfig.Port)
	}

	// Clean up
	os.Unsetenv("FULCRUM_PORT")
	os.Unsetenv("FULCRUM_JOB_MAINTENANCE_DURATION")
	os.Unsetenv("FULCRUM_AGENT_HEALTH_TIMEOUT")
	os.Unsetenv("FULCRUM_DB_HOST")
	os.Unsetenv("FULCRUM_DB_PORT")
}
