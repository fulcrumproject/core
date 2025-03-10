package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"
)

var (
	// Validate SSL mode (common values: disable, require, verify-ca, verify-full)
	validSSLModes = map[string]bool{
		"disable":     true,
		"require":     true,
		"verify-ca":   true,
		"verify-full": true,
	}

	// Validate log level (silent, error, warn, info)
	validLogLevels = map[string]bool{
		"silent": true,
		"error":  true,
		"warn":   true,
		"info":   true,
	}
)

// Fulcrum configuration
type Config struct {
	Port        uint        `json:"port" env:"FULCRUM_PORT"`
	JobConfig   JobConfig   `json:"job"`
	AgentConfig AgentConfig `json:"agent"`
	DBConfig    DBConfig    `json:"db"`
}

// Fulcrum Agent configuration
type AgentConfig struct {
	HealthTimeout time.Duration `json:"healthTimeout" env:"FULCRUM_AGENT_HEALTH_TIMEOUT"`
}

// Fulcrum Job configuration
type JobConfig struct {
	Maintenance   time.Duration `json:"maintenance" env:"FULCRUM_JOB_MAINTENANCE_DURATION"`
	RetentionDays int           `json:"retention" env:"FULCRUM_JOB_RETENTION_DURATION"`
	TimeoutMins   int           `json:"timeout" env:"FULCRUM_JOB_TIMEOUT_DURATION"`
}

// Fulcrum DB configuration
type DBConfig struct {
	Host     string `json:"host" env:"FULCRUM_DB_HOST"`
	User     string `json:"user" env:"FULCRUM_DB_USER"`
	Password string `json:"password" env:"FULCRUM_DB_PASSWORD"`
	Name     string `json:"name" env:"FULCRUM_DB_NAME"`
	Port     int    `json:"port" env:"FULCRUM_DB_PORT"`
	SSLMode  string `json:"sslMode" env:"FULCRUM_DB_SSL_MODE"`
	LogLevel string `json:"logLevel" env:"FULCRUM_DB_LOG_LEVEL"`
}

// DSN returns the PostgreSQL connection string
func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode,
	)
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Port: 3000,
		JobConfig: JobConfig{
			Maintenance:   10 * time.Minute,
			RetentionDays: 7,
			TimeoutMins:   15,
		},
		AgentConfig: AgentConfig{
			HealthTimeout: 5 * time.Minute,
		},
		DBConfig: DBConfig{
			Host:     "localhost",
			User:     "fulcrum",
			Password: "fulcrum_password",
			Name:     "fulcrum_db",
			Port:     5432,
			SSLMode:  "disable",
			LogLevel: "warn",
		},
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
	// Process base config fields
	if err := loadEnvToStruct(c); err != nil {
		return err
	}
	// Process nested structs
	if err := loadEnvToStruct(&c.JobConfig); err != nil {
		return err
	}
	if err := loadEnvToStruct(&c.AgentConfig); err != nil {
		return err
	}
	if err := loadEnvToStruct(&c.DBConfig); err != nil {
		return err
	}
	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate main config
	if c.Port == 0 {
		return fmt.Errorf("port cannot be 0")
	}

	// Validate job config
	if c.JobConfig.Maintenance <= 0 {
		return fmt.Errorf("job maintenance duration must be positive")
	}
	if c.JobConfig.RetentionDays <= 0 {
		return fmt.Errorf("job retention duration must be positive")
	}
	if c.JobConfig.TimeoutMins <= 0 {
		return fmt.Errorf("job timeout duration must be positive")
	}

	// Validate agent config
	if c.AgentConfig.HealthTimeout <= 0 {
		return fmt.Errorf("agent health timeout must be positive")
	}

	// Validate DB config
	if c.DBConfig.Host == "" {
		return fmt.Errorf("db host cannot be empty")
	}
	if c.DBConfig.User == "" {
		return fmt.Errorf("db user cannot be empty")
	}
	if c.DBConfig.Password == "" {
		return fmt.Errorf("db password cannot be empty")
	}
	if c.DBConfig.Name == "" {
		return fmt.Errorf("db name cannot be empty")
	}
	if c.DBConfig.Port <= 0 || c.DBConfig.Port > 65535 {
		return fmt.Errorf("db port must be between 1 and 65535")
	}
	if !validSSLModes[c.DBConfig.SSLMode] {
		return fmt.Errorf("invalid ssl mode: %s", c.DBConfig.SSLMode)
	}
	if c.DBConfig.LogLevel != "" && !validLogLevels[c.DBConfig.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.DBConfig.LogLevel)
	}

	return nil
}

// loadEnvToStruct loads environment variables into struct fields based on tags
func loadEnvToStruct(target interface{}) error {
	v := reflect.ValueOf(target).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get env tag or skip if not present
		envVar, ok := field.Tag.Lookup("env")
		if !ok || envVar == "" {
			continue
		}

		// Get value from environment or skip if empty
		envValue := os.Getenv(envVar)
		if envValue == "" {
			continue
		}

		// Set field value based on type
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(envValue)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Type == reflect.TypeOf(time.Duration(0)) {
				// Handle time.Duration
				duration, err := time.ParseDuration(envValue)
				if err != nil {
					return fmt.Errorf("invalid duration value for %s: %w", envVar, err)
				}
				fieldValue.SetInt(int64(duration))
			} else {
				// Handle regular integers
				val, err := strconv.ParseInt(envValue, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid integer value for %s: %w", envVar, err)
				}
				fieldValue.SetInt(val)
			}

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val, err := strconv.ParseUint(envValue, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid unsigned integer value for %s: %w", envVar, err)
			}
			fieldValue.SetUint(val)

		case reflect.Float32, reflect.Float64:
			val, err := strconv.ParseFloat(envValue, 64)
			if err != nil {
				return fmt.Errorf("invalid float value for %s: %w", envVar, err)
			}
			fieldValue.SetFloat(val)

		case reflect.Bool:
			val, err := strconv.ParseBool(envValue)
			if err != nil {
				return fmt.Errorf("invalid boolean value for %s: %w", envVar, err)
			}
			fieldValue.SetBool(val)
		}
	}

	return nil
}
