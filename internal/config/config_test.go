package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg, _ := Builder().Build()

	// Test default values
	if cfg.Port != 3000 {
		t.Errorf("Expected default Port to be 3000, got %d", cfg.Port)
	}

	if cfg.LogConfig.Format != "text" {
		t.Errorf("Expected default LogConfig.Format to be 'text', got '%s'", cfg.LogConfig.Format)
	}

	if cfg.LogConfig.Level != "info" {
		t.Errorf("Expected default LogConfig.Level to be 'info', got '%s'", cfg.LogConfig.Level)
	}

	if cfg.JobConfig.Maintenance != 3*time.Minute {
		t.Errorf("Expected default JobConfig.Maintenance to be 3m, got %s", cfg.JobConfig.Maintenance)
	}

	if cfg.JobConfig.Retention != 3*24*time.Hour {
		t.Errorf("Expected default JobConfig.Retention to be 72h, got %s", cfg.JobConfig.Retention)
	}

	if cfg.JobConfig.Timeout != 5*time.Minute {
		t.Errorf("Expected default JobConfig.Timeout to be 5m, got %s", cfg.JobConfig.Timeout)
	}

	if cfg.AgentConfig.HealthTimeout != 5*time.Minute {
		t.Errorf("Expected default AgentConfig.HealthTimeout to be 5m, got %s", cfg.AgentConfig.HealthTimeout)
	}

	if cfg.DBConfig.Host != "localhost" {
		t.Errorf("Expected default DBConfig.Host to be 'localhost', got '%s'", cfg.DBConfig.Host)
	}

	if cfg.DBConfig.User != "fulcrum" {
		t.Errorf("Expected default DBConfig.User to be 'fulcrum', got '%s'", cfg.DBConfig.User)
	}

	if cfg.DBConfig.Password != "fulcrum_password" {
		t.Errorf("Expected default DBConfig.Password to be 'fulcrum_password', got '%s'", cfg.DBConfig.Password)
	}

	if cfg.DBConfig.Name != "fulcrum_db" {
		t.Errorf("Expected default DBConfig.Name to be 'fulcrum_db', got '%s'", cfg.DBConfig.Name)
	}

	if cfg.DBConfig.Port != 5432 {
		t.Errorf("Expected default DBConfig.Port to be 5432, got %d", cfg.DBConfig.Port)
	}

	if cfg.DBConfig.SSLMode != "disable" {
		t.Errorf("Expected default DBConfig.SSLMode to be 'disable', got '%s'", cfg.DBConfig.SSLMode)
	}

	if cfg.DBConfig.LogLevel != "warn" {
		t.Errorf("Expected default DBConfig.LogLevel to be 'warn', got '%s'", cfg.DBConfig.LogLevel)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary file with valid JSON config
	validJSON := `{
		"port": 8080,
		"log": {
			"format": "json",
			"level": "warn"
		},
		"job": {
			"maintenance": 600000000000,
			"retention": 172800000000000,
			"timeout": 900000000000
		},
		"agent": {
			"healthTimeout": 900000000000
		},
		"db": {
			"host": "testdb",
			"user": "tester",
			"password": "test_password",
			"name": "test_db",
			"port": 5433,
			"sslMode": "require",
			"logLevel": "error"
		}
	}`

	// Setup valid config test
	validConfigPath := filepath.Join(t.TempDir(), "valid_config.json")
	if err := os.WriteFile(validConfigPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test loading valid config
	cfg, err := Builder().LoadFile(&validConfigPath).Build()
	if err != nil {
		t.Fatalf("Failed to load config from file: %v", err)
	}

	// Check if values were loaded correctly
	if cfg.Port != 8080 {
		t.Errorf("Expected Port to be 8080, got %d", cfg.Port)
	}
	if cfg.LogConfig.Format != "json" {
		t.Errorf("Expected LogConfig.Format to be 'json', got '%s'", cfg.LogConfig.Format)
	}
	if cfg.LogConfig.Level != "warn" {
		t.Errorf("Expected LogConfig.Level to be 'warn', got '%s'", cfg.LogConfig.Level)
	}
	if cfg.JobConfig.Maintenance != 10*time.Minute {
		t.Errorf("Expected JobConfig.Maintenance to be 10m, got %s", cfg.JobConfig.Maintenance)
	}
	if cfg.JobConfig.Retention != 48*time.Hour {
		t.Errorf("Expected JobConfig.Retention to be 48h, got %s", cfg.JobConfig.Retention)
	}
	if cfg.JobConfig.Timeout != 15*time.Minute {
		t.Errorf("Expected JobConfig.Timeout to be 15m, got %s", cfg.JobConfig.Timeout)
	}
	if cfg.AgentConfig.HealthTimeout != 15*time.Minute {
		t.Errorf("Expected AgentConfig.HealthTimeout to be 15m, got %s", cfg.AgentConfig.HealthTimeout)
	}
	if cfg.DBConfig.Host != "testdb" {
		t.Errorf("Expected DBConfig.Host to be 'testdb', got '%s'", cfg.DBConfig.Host)
	}
	if cfg.DBConfig.Port != 5433 {
		t.Errorf("Expected DBConfig.Port to be 5433, got %d", cfg.DBConfig.Port)
	}
	if cfg.DBConfig.SSLMode != "require" {
		t.Errorf("Expected DBConfig.SSLMode to be 'require', got '%s'", cfg.DBConfig.SSLMode)
	}

	// Setup invalid JSON test
	invalidJSON := `{invalid json}`
	invalidConfigPath := filepath.Join(t.TempDir(), "invalid_config.json")
	if err := os.WriteFile(invalidConfigPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to create invalid test config file: %v", err)
	}

	// Test loading invalid JSON
	_, err = Builder().LoadFile(&invalidConfigPath).Build()
	if err == nil {
		t.Error("Expected error when loading invalid JSON, got nil")
	}

	// Test loading non-existent file
	nonExistentFilePath := "/path/to/nonexistent/config.json"
	_, err = Builder().LoadFile(&nonExistentFilePath).Build()
	if err == nil {
		t.Error("Expected error when loading non-existent file, got nil")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name         string
		mutateConfig func(*Config)
		wantErr      bool
		errContains  string
	}{
		{
			name:         "Valid default config",
			mutateConfig: func(c *Config) {},
			wantErr:      false,
		},
		{
			name:         "Invalid port",
			mutateConfig: func(c *Config) { c.Port = 0 },
			wantErr:      true,
			errContains:  "port cannot be 0",
		},
		{
			name:         "Invalid log format",
			mutateConfig: func(c *Config) { c.LogConfig.Format = "invalid" },
			wantErr:      true,
			errContains:  "invalid log format",
		},
		{
			name:         "Invalid log level",
			mutateConfig: func(c *Config) { c.LogConfig.Level = "invalid" },
			wantErr:      true,
			errContains:  "invalid log level",
		},
		{
			name:         "Invalid job maintenance",
			mutateConfig: func(c *Config) { c.JobConfig.Maintenance = 0 },
			wantErr:      true,
			errContains:  "job maintenance duration must be positive",
		},
		{
			name:         "Invalid job retention",
			mutateConfig: func(c *Config) { c.JobConfig.Retention = 0 },
			wantErr:      true,
			errContains:  "job retention duration must be positive",
		},
		{
			name:         "Invalid job timeout",
			mutateConfig: func(c *Config) { c.JobConfig.Timeout = 0 },
			wantErr:      true,
			errContains:  "job timeout duration must be positive",
		},
		{
			name:         "Invalid agent health timeout",
			mutateConfig: func(c *Config) { c.AgentConfig.HealthTimeout = 0 },
			wantErr:      true,
			errContains:  "agent health timeout must be positive",
		},
		{
			name:         "Empty DB host",
			mutateConfig: func(c *Config) { c.DBConfig.Host = "" },
			wantErr:      true,
			errContains:  "db host cannot be empty",
		},
		{
			name:         "Empty DB user",
			mutateConfig: func(c *Config) { c.DBConfig.User = "" },
			wantErr:      true,
			errContains:  "db user cannot be empty",
		},
		{
			name:         "Empty DB password",
			mutateConfig: func(c *Config) { c.DBConfig.Password = "" },
			wantErr:      true,
			errContains:  "db password cannot be empty",
		},
		{
			name:         "Empty DB name",
			mutateConfig: func(c *Config) { c.DBConfig.Name = "" },
			wantErr:      true,
			errContains:  "db name cannot be empty",
		},
		{
			name:         "DB port zero",
			mutateConfig: func(c *Config) { c.DBConfig.Port = 0 },
			wantErr:      true,
			errContains:  "db port must be between 1 and 65535",
		},
		{
			name:         "DB port too large",
			mutateConfig: func(c *Config) { c.DBConfig.Port = 65536 },
			wantErr:      true,
			errContains:  "db port must be between 1 and 65535",
		},
		{
			name:         "Invalid SSL mode",
			mutateConfig: func(c *Config) { c.DBConfig.SSLMode = "invalid" },
			wantErr:      true,
			errContains:  "invalid ssl mode",
		},
		{
			name:         "Invalid DB log level",
			mutateConfig: func(c *Config) { c.DBConfig.LogLevel = "invalid" },
			wantErr:      true,
			errContains:  "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := Builder().Build()
			tt.mutateConfig(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("Validate() error message does not contain expected text, got: %v, want to contain: %v", err.Error(), tt.errContains)
				}
			}
		})
	}
}

func TestLogLevelConversion(t *testing.T) {
	tests := []struct {
		level    string
		expected slog.Level
	}{
		{"silent", slog.Level(99)},
		{"error", slog.LevelError},
		{"warn", slog.LevelWarn},
		{"info", slog.LevelInfo},
		{"", slog.LevelInfo},        // Empty defaults to info
		{"unknown", slog.LevelInfo}, // Unknown defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			logCfg := LogConfig{Level: tt.level}
			if got := logCfg.GetLogLevel(); got != tt.expected {
				t.Errorf("LogConfig.GetLogLevel() = %v, want %v", got, tt.expected)
			}

			dbCfg := DBConfig{LogLevel: tt.level}
			if got := dbCfg.GetLogLevel(); got != tt.expected {
				t.Errorf("DBConfig.GetLogLevel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDBConfigDSN(t *testing.T) {
	dbConfig := DBConfig{
		Host:     "testhost",
		User:     "testuser",
		Password: "testpass",
		Name:     "testdb",
		Port:     5432,
		SSLMode:  "require",
	}

	expected := "host=testhost user=testuser password=testpass dbname=testdb port=5432 sslmode=require"
	if got := dbConfig.DSN(); got != expected {
		t.Errorf("DBConfig.DSN() = %v, want %v", got, expected)
	}
}

func TestLoadFromEnvOverridesDefaults(t *testing.T) {
	// Already tested in env_helper_test.go, but adding a more specific test here
	// Set environment variables for testing
	os.Setenv("FULCRUM_PORT", "9090")
	os.Setenv("FULCRUM_LOG_FORMAT", "json")
	os.Setenv("FULCRUM_LOG_LEVEL", "error")
	os.Setenv("FULCRUM_DB_SSL_MODE", "verify-full")
	os.Setenv("FULCRUM_DB_LOG_FORMAT", "json")

	// Create a config instance
	cfg, err := Builder().WithEnv().Build()
	if err != nil {
		t.Fatalf("Failed to load config from env: %v", err)
	}

	// Verify values were loaded correctly
	if cfg.Port != 9090 {
		t.Errorf("Expected Port to be 9090, got %d", cfg.Port)
	}

	if cfg.LogConfig.Format != "json" {
		t.Errorf("Expected LogConfig.Format to be 'json', got '%s'", cfg.LogConfig.Format)
	}

	if cfg.LogConfig.Level != "error" {
		t.Errorf("Expected LogConfig.Level to be 'error', got '%s'", cfg.LogConfig.Level)
	}

	if cfg.DBConfig.SSLMode != "verify-full" {
		t.Errorf("Expected DBConfig.SSLMode to be 'verify-full', got '%s'", cfg.DBConfig.SSLMode)
	}

	// Clean up
	os.Unsetenv("FULCRUM_PORT")
	os.Unsetenv("FULCRUM_LOG_FORMAT")
	os.Unsetenv("FULCRUM_LOG_LEVEL")
	os.Unsetenv("FULCRUM_DB_SSL_MODE")
	os.Unsetenv("FULCRUM_DB_LOG_FORMAT")
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
