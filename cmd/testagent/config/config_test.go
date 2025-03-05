package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromEnv(t *testing.T) {
	// Helper function to set and clean up environment variables
	setEnv := func(key, value string) {
		os.Setenv(key, value)
	}

	cleanEnv := func() {
		vars := []string{
			"TESTAGENT_AGENT_TOKEN",
			"TESTAGENT_FULCRUM_API_URL",
			"TESTAGENT_VM_OPERATION_INTERVAL",
			"TESTAGENT_JOB_POLL_INTERVAL",
			"TESTAGENT_METRIC_REPORT_INTERVAL",
			"TESTAGENT_OPERATION_DELAY_MIN",
			"TESTAGENT_OPERATION_DELAY_MAX",
			"TESTAGENT_ERROR_RATE",
		}
		for _, v := range vars {
			os.Unsetenv(v)
		}
	}

	// Clean up after all tests
	defer cleanEnv()

	tests := []struct {
		name      string
		setupEnv  func()
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid environment variables",
			setupEnv: func() {
				setEnv("TESTAGENT_AGENT_TOKEN", "test-token")
				setEnv("TESTAGENT_FULCRUM_API_URL", "http://test-url")
				setEnv("TESTAGENT_VM_OPERATION_INTERVAL", "10s")
				setEnv("TESTAGENT_JOB_POLL_INTERVAL", "15s")
				setEnv("TESTAGENT_METRIC_REPORT_INTERVAL", "60s")
				setEnv("TESTAGENT_OPERATION_DELAY_MIN", "2s")
				setEnv("TESTAGENT_OPERATION_DELAY_MAX", "30s")
				setEnv("TESTAGENT_ERROR_RATE", "0.25")
			},
			wantError: false,
		},
		{
			name: "invalid VM operation interval",
			setupEnv: func() {
				setEnv("TESTAGENT_VM_OPERATION_INTERVAL", "invalid")
			},
			wantError: true,
			errorMsg:  "invalid VM operation interval",
		},
		{
			name: "invalid job poll interval",
			setupEnv: func() {
				setEnv("TESTAGENT_JOB_POLL_INTERVAL", "invalid")
			},
			wantError: true,
			errorMsg:  "invalid job poll interval",
		},
		{
			name: "invalid metric report interval",
			setupEnv: func() {
				setEnv("TESTAGENT_METRIC_REPORT_INTERVAL", "invalid")
			},
			wantError: true,
			errorMsg:  "invalid metric report interval",
		},
		{
			name: "invalid operation delay min",
			setupEnv: func() {
				setEnv("TESTAGENT_OPERATION_DELAY_MIN", "invalid")
			},
			wantError: true,
			errorMsg:  "invalid operation delay minimum",
		},
		{
			name: "invalid operation delay max",
			setupEnv: func() {
				setEnv("TESTAGENT_OPERATION_DELAY_MAX", "invalid")
			},
			wantError: true,
			errorMsg:  "invalid operation delay maximum",
		},
		{
			name: "invalid error rate",
			setupEnv: func() {
				setEnv("TESTAGENT_ERROR_RATE", "invalid")
			},
			wantError: true,
			errorMsg:  "invalid error rate",
		},
		{
			name: "empty environment variables",
			setupEnv: func() {
				// No variables set
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			cleanEnv()

			// Set up test environment
			tt.setupEnv()

			// Create a new config with defaults
			cfg := DefaultConfig()

			// Load from environment
			err := cfg.LoadFromEnv()

			// Check if error matches expectation
			if tt.wantError && err == nil {
				t.Errorf("LoadFromEnv() expected error but got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("LoadFromEnv() unexpected error: %v", err)
			}

			if tt.wantError && err != nil {
				if tt.errorMsg != "" && !containsStr(err.Error(), tt.errorMsg) {
					t.Errorf("LoadFromEnv() error message does not contain expected string: got %v, want to contain %v", err, tt.errorMsg)
				}
			}

			// Additional checks for the valid case
			if tt.name == "valid environment variables" && err == nil {
				// Check that values were properly parsed
				if cfg.AgentToken != "test-token" {
					t.Errorf("Expected AgentToken to be 'test-token', got '%s'", cfg.AgentToken)
				}

				if cfg.FulcrumAPIURL != "http://test-url" {
					t.Errorf("Expected FulcrumAPIURL to be 'http://test-url', got '%s'", cfg.FulcrumAPIURL)
				}

				if cfg.VMUpdateInterval != 10*time.Second {
					t.Errorf("Expected VMUpdateInterval to be 10s, got %v", cfg.VMUpdateInterval)
				}

				if cfg.ErrorRate != 0.25 {
					t.Errorf("Expected ErrorRate to be 0.25, got %f", cfg.ErrorRate)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
