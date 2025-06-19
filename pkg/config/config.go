package config

import (
	"time"

	"github.com/fulcrumproject/commons/config"
	"github.com/fulcrumproject/commons/keycloak"
)

const (
	EnvPrefix = "FULCRUM_"
)

// Fulcrum configuration
type Config struct {
	Port           uint            `json:"port" env:"PORT" validate:"required,min=1,max=65535"`
	Authenticators []string        `json:"authenticators" env:"AUTHENTICATORS" validate:"omitempty,dive,oneof=oauth token"`
	JobConfig      JobConfig       `json:"job" validate:"required"`
	AgentConfig    AgentConfig     `json:"agent" validate:"required"`
	LogConfig      config.Log      `json:"log" validate:"required"`
	DBConfig       config.DB       `json:"db" validate:"required"`
	OAuthConfig    keycloak.Config `json:"oauth" validate:"required"`
}

// Fulcrum Agent configuration
type AgentConfig struct {
	HealthTimeout time.Duration `json:"healthTimeout" env:"AGENT_HEALTH_TIMEOUT"`
}

// Fulcrum Job configuration
type JobConfig struct {
	Maintenance time.Duration `json:"maintenance" env:"JOB_MAINTENANCE_INTERVAL"`
	Retention   time.Duration `json:"retention" env:"JOB_RETENTION_INTERVAL"`
	Timeout     time.Duration `json:"timeout" env:"JOB_TIMEOUT_INTERVAL"`
}

var Default = Config{
	Port:           8080,
	Authenticators: []string{"token"},
	JobConfig: JobConfig{
		Maintenance: 24 * time.Hour,
		Retention:   30 * 24 * time.Hour,
		Timeout:     5 * time.Minute,
	},
	AgentConfig: AgentConfig{
		HealthTimeout: 30 * time.Second,
	},
	LogConfig: config.Log{
		Level:  "info",
		Format: "json",
	},
	DBConfig: config.DB{
		DSN:       "host=localhost user=fulcrum password=password dbname=fulcrum_db port=5432 sslmode=disable",
		LogLevel:  "warn",
		LogFormat: "text",
	},
}
