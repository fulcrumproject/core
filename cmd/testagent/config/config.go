package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for the test agent
type Config struct {
	// Agent authentication
	AgentToken string `json:"agentToken"` // Authentication token for the agent

	// Fulcrum Core API connection
	FulcrumAPIURL string `json:"fulcrumApiUrl"`

	// Simulation parameters
	VMCount              int           `json:"vmCount"`              // Number of VMs to simulate
	VMOperationInterval  time.Duration `json:"vmOperationInterval"`  // How often to perform VM operations
	JobPollInterval      time.Duration `json:"jobPollInterval"`      // How often to poll for jobs
	MetricReportInterval time.Duration `json:"metricReportInterval"` // How often to report metrics

	// Simulation behavior
	OperationDelayMin time.Duration `json:"operationDelayMin"` // Minimum time for operation
	OperationDelayMax time.Duration `json:"operationDelayMax"` // Maximum time for operation
	ErrorRate         float64       `json:"errorRate"`         // Probability of operation failure (0.0-1.0)
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		AgentToken:           "", // Must be provided
		FulcrumAPIURL:        "http://localhost:3000",
		VMCount:              10,
		VMOperationInterval:  30 * time.Second,
		JobPollInterval:      5 * time.Second,
		MetricReportInterval: 60 * time.Second,
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
func (c *Config) LoadFromEnv() {
	// Map environment variables to config fields
	if v := os.Getenv("TESTAGENT_AGENT_TOKEN"); v != "" {
		c.AgentToken = v
	}
	if v := os.Getenv("TESTAGENT_FULCRUM_API_URL"); v != "" {
		c.FulcrumAPIURL = v
	}

	// Load numeric and duration parameters
	if v := os.Getenv("TESTAGENT_VM_COUNT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.VMCount = i
		}
	}
	if v := os.Getenv("TESTAGENT_VM_OPERATION_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.VMOperationInterval = d
		}
	}
	if v := os.Getenv("TESTAGENT_JOB_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.JobPollInterval = d
		}
	}
	if v := os.Getenv("TESTAGENT_METRIC_REPORT_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.MetricReportInterval = d
		}
	}
	if v := os.Getenv("TESTAGENT_OPERATION_DELAY_MIN"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.OperationDelayMin = d
		}
	}
	if v := os.Getenv("TESTAGENT_OPERATION_DELAY_MAX"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.OperationDelayMax = d
		}
	}
	if v := os.Getenv("TESTAGENT_ERROR_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.ErrorRate = f
		}
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.AgentToken == "" {
		return fmt.Errorf("agent token is required")
	}
	if c.FulcrumAPIURL == "" {
		return fmt.Errorf("Fulcrum API URL is required")
	}
	if c.OperationDelayMin > c.OperationDelayMax {
		return fmt.Errorf("minimum operation delay cannot be greater than maximum")
	}
	if c.ErrorRate < 0.0 || c.ErrorRate > 1.0 {
		return fmt.Errorf("error rate must be between 0.0 and 1.0")
	}
	return nil
}
