package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	tag       = "env"
	envPrefix = "FULCRUM_"
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

	// Validate log formats (text, json)
	validLogFormats = map[string]bool{
		"text": true,
		"json": true,
	}
)

// Fulcrum configuration
type Config struct {
	Port           uint        `json:"port" env:"PORT"`
	Authenticators []string    `json:"authenticators" env:"AUTHENTICATORS"`
	JobConfig      JobConfig   `json:"job"`
	AgentConfig    AgentConfig `json:"agent"`
	LogConfig      LogConfig   `json:"log"`
	DBConfig       DBConfig    `json:"db"`
	OAuthConfig    OAuthConfig `json:"oauth"`
}

// Fulcrum Agent configuration
type AgentConfig struct {
	HealthTimeout time.Duration `json:"healthTimeout" env:"AGENT_HEALTH_TIMEOUT"`
}

// Fulcrum Log configuration
type LogConfig struct {
	Format string `json:"format" env:"LOG_FORMAT"`
	Level  string `json:"level" env:"LOG_LEVEL"`
}

// GetLogLevel converts a string log level to slog.Level
func (c *LogConfig) GetLogLevel() slog.Level {
	return logLevel(c.Level)
}

// Fulcrum Job configuration
type JobConfig struct {
	Maintenance time.Duration `json:"maintenance" env:"JOB_MAINTENANCE_INTERVAL"`
	Retention   time.Duration `json:"retention" env:"JOB_RETENTION_INTERVAL"`
	Timeout     time.Duration `json:"timeout" env:"JOB_TIMEOUT_INTERVAL"`
}

// Fulcrum DB configuration
type DBConfig struct {
	Host      string `json:"host" env:"DB_HOST"`
	User      string `json:"user" env:"DB_USER"`
	Password  string `json:"password" env:"DB_PASSWORD"`
	Name      string `json:"name" env:"DB_NAME"`
	Port      int    `json:"port" env:"DB_PORT"`
	SSLMode   string `json:"sslMode" env:"DB_SSL_MODE"`
	LogLevel  string `json:"logLevel" env:"DB_LOG_LEVEL"`
	LogFormat string `json:"logFormat" env:"DB_LOG_FORMAT"`
}

// Fulcrum OAuth configuration
type OAuthConfig struct {
	KeycloakURL    string `json:"keycloakUrl" env:"OAUTH_KEYCLOAK_URL"`
	Realm          string `json:"realm" env:"OAUTH_REALM"`
	ClientID       string `json:"clientId" env:"OAUTH_CLIENT_ID"`
	ClientSecret   string `json:"clientSecret" env:"OAUTH_CLIENT_SECRET"`
	JWKSCacheTTL   int    `json:"jwksCacheTtl" env:"OAUTH_JWKS_CACHE_TTL"`
	ValidateIssuer bool   `json:"validateIssuer" env:"OAUTH_VALIDATE_ISSUER"`
}

// GetJWKSURL returns the JWKS endpoint URL for the Keycloak realm
func (c *OAuthConfig) GetJWKSURL() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid_connect/certs", c.KeycloakURL, c.Realm)
}

// GetIssuer returns the expected issuer for JWT tokens
func (c *OAuthConfig) GetIssuer() string {
	return fmt.Sprintf("%s/realms/%s", c.KeycloakURL, c.Realm)
}

// DSN returns the PostgreSQL connection string
func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		c.Host, c.User, c.Password, c.Name, c.Port, c.SSLMode,
	)
}

// GetLogLevel converts the string log level to gorm logger.LogLevel
func (c *DBConfig) GetLogLevel() slog.Level {
	return logLevel(c.LogLevel)
}

func logLevel(value string) slog.Level {
	switch value {
	case "silent":
		return slog.Level(99) // Higher than any standard level
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "info", "": // Default to info if empty
		return slog.LevelInfo
	default:
		return slog.LevelInfo // Default to info for unknown levels
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate main config
	if c.Port == 0 {
		return fmt.Errorf("port cannot be 0")
	}

	// Validate log config
	if c.LogConfig.Format != "" && !validLogFormats[c.LogConfig.Format] {
		return fmt.Errorf("invalid log format: %s, must be one of: text, json",
			c.LogConfig.Format)
	}
	if c.LogConfig.Level != "" && !validLogLevels[c.LogConfig.Level] {
		return fmt.Errorf("invalid log level: %s, must be one of: silent, error, warn, info",
			c.LogConfig.Level)
	}

	// Validate job config
	if c.JobConfig.Maintenance <= 0 {
		return fmt.Errorf("job maintenance duration must be positive")
	}
	if c.JobConfig.Retention <= 0 {
		return fmt.Errorf("job retention duration must be positive")
	}
	if c.JobConfig.Timeout <= 0 {
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

	// Validate authenticators
	for _, auth := range c.Authenticators {
		switch auth {
		case "token":
			// Token authenticator requires no specific config validation here
		case "oauth":
			// OAuth authenticator requires OAuthConfig validation
			if c.OAuthConfig.KeycloakURL == "" {
				return fmt.Errorf("oauth keycloak URL cannot be empty when oauth authenticator is enabled")
			}
			if c.OAuthConfig.Realm == "" {
				return fmt.Errorf("oauth realm cannot be empty when oauth authenticator is enabled")
			}
			if c.OAuthConfig.ClientID == "" {
				return fmt.Errorf("oauth client ID cannot be empty when oauth authenticator is enabled")
			}
			if c.OAuthConfig.JWKSCacheTTL <= 0 {
				return fmt.Errorf("oauth JWKS cache TTL must be positive when oauth authenticator is enabled")
			}
		default:
			return fmt.Errorf("unknown authenticator: %s", auth)
		}
	}

	return nil
}

// ConfigBuilder implements a builder pattern for creating Config instances
type ConfigBuilder struct {
	config *Config
	err    error
}

// Default returns a ConfigBuilder with default configuration
func Builder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &Config{
			Port:           3000,
			Authenticators: []string{"token"},
			LogConfig: LogConfig{
				Format: "text",
				Level:  "info",
			},
			JobConfig: JobConfig{
				Maintenance: 3 * time.Minute,
				Retention:   3 * 24 * time.Hour,
				Timeout:     5 * time.Minute,
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
			OAuthConfig: OAuthConfig{
				KeycloakURL:    "",
				Realm:          "",
				ClientID:       "",
				ClientSecret:   "",
				JWKSCacheTTL:   3600, // 1 hour in seconds
				ValidateIssuer: true,
			},
		},
	}
}

// LoadFile loads configuration from a JSON file
func (b *ConfigBuilder) LoadFile(filepath *string) *ConfigBuilder {
	if b.err != nil {
		return b
	}

	if filepath == nil || *filepath == "" {
		return b
	}

	data, err := os.ReadFile(*filepath)
	if err != nil {
		b.err = fmt.Errorf("failed to read config file: %w", err)
		return b
	}

	if err := json.Unmarshal(data, b.config); err != nil {
		b.err = fmt.Errorf("failed to parse config file: %w", err)
		return b
	}

	return b
}

// WithEnv overrides configuration from environment variables
func (b *ConfigBuilder) WithEnv() *ConfigBuilder {
	if b.err != nil {
		return b
	}

	err := loadEnvFromAncestors()
	if err != nil {
		b.err = fmt.Errorf("failed to load environment variables: %w", err)
		return b
	}

	if err := loadEnvToStruct(b.config, envPrefix, tag); err != nil {
		b.err = fmt.Errorf("failed to override configuration from environment: %w", err)
		return b
	}

	// Manually load and parse FULCRUM_AUTHENTICATORS
	authenticatorsStr := os.Getenv(envPrefix + "AUTHENTICATORS")
	if authenticatorsStr != "" {
		b.config.Authenticators = strings.Split(authenticatorsStr, ",")
	}

	return b
}

// Build validates and returns the final Config
func (b *ConfigBuilder) Build() (*Config, error) {
	if b.err != nil {
		return nil, b.err
	}

	if err := b.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return b.config, nil
}
