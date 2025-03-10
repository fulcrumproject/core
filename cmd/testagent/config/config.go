package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	internalConfig "fulcrumproject.org/core/internal/config"
)

// Config holds the configuration for the test agent
type Config struct {
	// Agent authentication
	AgentToken string `json:"agentToken" env:"AGENT_TOKEN"` // Authentication token for the agent

	// Fulcrum Core API connection
	FulcrumAPIURL string `json:"fulcrumApiUrl" env:"FULCRUM_API_URL"`

	// Simulation parameters
	VMUpdateInterval     time.Duration `json:"vmUpdateInterval" env:"VM_OPERATION_INTERVAL"`      // How often to perform VM operations
	JobPollInterval      time.Duration `json:"jobPollInterval" env:"JOB_POLL_INTERVAL"`           // How often to poll for jobs
	MetricReportInterval time.Duration `json:"metricReportInterval" env:"METRIC_REPORT_INTERVAL"` // How often to report metrics

	// Simulation behavior
	OperationDelayMin time.Duration `json:"operationDelayMin" env:"OPERATION_DELAY_MIN"` // Minimum time for operation
	OperationDelayMax time.Duration `json:"operationDelayMax" env:"OPERATION_DELAY_MAX"` // Maximum time for operation
	ErrorRate         float64       `json:"errorRate" env:"ERROR_RATE"`                  // Probability of operation failure (0.0-1.0)
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		AgentToken:           "", // Must be provided
		FulcrumAPIURL:        "http://localhost:3000",
		VMUpdateInterval:     5 * time.Second,
		JobPollInterval:      5 * time.Second,
		MetricReportInterval: 30 * time.Second,
		OperationDelayMin:    2 * time.Second,
		OperationDelayMax:    10 * time.Second,
		ErrorRate:            0.05, // 5% chance of failure
	}
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(filepath string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// LoadFromEnv overrides configuration with environment variables
func (c *Config) LoadFromEnv() error {
	return internalConfig.LoadEnvToStruct(c, "TESTAGENT_", "env")
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.AgentToken == "" {
		return fmt.Errorf("agent token is required")
	}
	if c.FulcrumAPIURL == "" {
		return fmt.Errorf("the Fulcrum API URL is required")
	}
	if c.OperationDelayMin > c.OperationDelayMax {
		return fmt.Errorf("minimum operation delay cannot be greater than maximum")
	}
	if c.ErrorRate < 0.0 || c.ErrorRate > 1.0 {
		return fmt.Errorf("error rate must be between 0.0 and 1.0")
	}
	return nil
}
