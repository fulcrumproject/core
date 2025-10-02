package config

import (
	"log/slog"
	"time"

	"github.com/fulcrumproject/core/pkg/keycloak"
	"github.com/fulcrumproject/utils/gormpg"
	"github.com/fulcrumproject/utils/logging"
)

const (
	EnvPrefix = "FULCRUM_"
)

// Fulcrum configuration
type Config struct {
	Port                    uint                  `json:"port" env:"PORT" validate:"required,min=1,max=65535"`
	ShutdownTimeout         time.Duration         `json:"shutdownTimeout" env:"SHUTDOWN_TIMEOUT"`
	SchedulerLockerConfig   SchedulerLockerConfig `json:"schedulerLocker" validate:"required"`
	SchedulerLockerDBConfig gormpg.Conf           `json:"schedulerLockerDb" env:"SCHEDULER_LOCKER_DB" validate:"required"`
	HealthPort              uint                  `json:"healthPort" env:"HEALTH_PORT" validate:"required,min=1,max=65535"`
	Authenticators          []string              `json:"authenticators" env:"AUTHENTICATORS" validate:"omitempty,dive,oneof=oauth token"`
	JobConfig               JobConfig             `json:"job" validate:"required"`
	AgentConfig             AgentConfig           `json:"agent" validate:"required"`
	LogConfig               logging.Conf          `json:"log" validate:"required"`
	DBConfig                gormpg.Conf           `json:"db" env:"DB" validate:"required"`
	MetricDBConfig          gormpg.Conf           `json:"metricDb" env:"METRIC_DB" validate:"required"`
	OAuthConfig             keycloak.Config       `json:"oauth" validate:"required"`
	ApiServer               bool                  `json:"apiServer" env:"API_SERVER" validate:"boolean"`
	JobMaintenance          bool                  `json:"jobMaintenance" env:"JOB_MAINTENANCE" validate:"boolean"`
	AgentMaintenance        bool                  `json:"agentMaintenance" env:"AGENT_MAINTENANCE" validate:"boolean"`
}

// Fulcrum scheduler locker configuration
type SchedulerLockerConfig struct {
	Name          string        `json:"name" env:"SCHEDULER_LOCKER_NAME"`
	CleanInterval time.Duration `json:"cleanInterval" env:"SCHEDULER_LOCKER_CLEAN_INTERVAL"`
	TTL           time.Duration `json:"ttl" env:"SCHEDULER_LOCKER_TTL"`
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
	Port:            8080,
	ShutdownTimeout: 30 * time.Second,
	SchedulerLockerConfig: SchedulerLockerConfig{
		Name:          "fulcrum-scheduler",
		CleanInterval: 30 * time.Minute,
		TTL:           72 * time.Hour,
	},
	SchedulerLockerDBConfig: gormpg.Conf{
		DSN:       "host=localhost user=fulcrum password=fulcrum_password dbname=fulcrum_db port=5432 sslmode=disable",
		LogLevel:  slog.LevelWarn,
		LogFormat: "text",
	},
	HealthPort:     8081,
	Authenticators: []string{"token"},
	JobConfig: JobConfig{
		Maintenance: 24 * time.Hour,
		Retention:   30 * 24 * time.Hour,
		Timeout:     5 * time.Minute,
	},
	AgentConfig: AgentConfig{
		HealthTimeout: 30 * time.Second,
	},
	LogConfig: logging.Conf{
		Level:  slog.LevelInfo,
		Format: "json",
	},
	DBConfig: gormpg.Conf{
		DSN:       "host=localhost user=fulcrum password=fulcrum_password dbname=fulcrum_db port=5432 sslmode=disable",
		LogLevel:  slog.LevelWarn,
		LogFormat: "text",
	},
	MetricDBConfig: gormpg.Conf{
		DSN:       "host=localhost user=fulcrum password=fulcrum_password dbname=fulcrum_db port=5432 sslmode=disable",
		LogLevel:  slog.LevelWarn,
		LogFormat: "text",
	},
	ApiServer:        true,
	JobMaintenance:   false,
	AgentMaintenance: false,
}
