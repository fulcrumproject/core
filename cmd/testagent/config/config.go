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
	VMUpdateInterval     time.Duration `json:"vmUpdateInterval"`     // How often to perform VM operations
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
	// Map environment variables to config fields
	if v := os.Getenv("TESTAGENT_AGENT_TOKEN"); v != "" {
		c.AgentToken = v
	}

	if v := os.Getenv("TESTAGENT_FULCRUM_API_URL"); v != "" {
		c.FulcrumAPIURL = v
	}

	// Load numeric and duration parameters
	if v := os.Getenv("TESTAGENT_VM_OPERATION_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid VM operation interval: %w", err)
		}
		c.VMUpdateInterval = d
	}

	if v := os.Getenv("TESTAGENT_JOB_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid job poll interval: %w", err)
		}
		c.JobPollInterval = d
	}

	if v := os.Getenv("TESTAGENT_METRIC_REPORT_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid metric report interval: %w", err)
		}
		c.MetricReportInterval = d
	}

	if v := os.Getenv("TESTAGENT_OPERATION_DELAY_MIN"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid operation delay minimum: %w", err)
		}
		c.OperationDelayMin = d
	}

	if v := os.Getenv("TESTAGENT_OPERATION_DELAY_MAX"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid operation delay maximum: %w", err)
		}
		c.OperationDelayMax = d
	}

	if v := os.Getenv("TESTAGENT_ERROR_RATE"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("invalid error rate: %w", err)
		}
		c.ErrorRate = f
	}

	return nil
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
