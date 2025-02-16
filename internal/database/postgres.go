package database

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"fulcrumproject.org/core/internal/domain"
)

// Config contains the database connection configuration
type Config struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     string
	SSLMode  string
}

// NewConfigFromEnv creates a new database configuration from environment variables
func NewConfigFromEnv() *Config {
	return &Config{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		User:     getEnvOrDefault("DB_USER", "fulcrum"),
		Password: getEnvOrDefault("DB_PASSWORD", "fulcrum_password"),
		DBName:   getEnvOrDefault("DB_NAME", "fulcrum_db"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		SSLMode:  getEnvOrDefault("DB_SSL_MODE", "disable"),
	}
}

// DSN returns the PostgreSQL connection string
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host, c.User, c.Password, c.DBName, c.Port, c.SSLMode,
	)
}

// getLogLevelFromEnv gets the log level from environment variable or returns the default
func getLogLevelFromEnv(key string, defaultValue logger.LogLevel) logger.LogLevel {
	value := os.Getenv(key)
	switch value {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return defaultValue
	}
}

// NewConnection creates a new database connection
func NewConnection(config *Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: config.DSN(),
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // logger.Default.LogMode(getLogLevelFromEnv("DB_LOG_LEVEL", logger.Warn)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Enable foreign key constraint
	db = db.Set("gorm:auto_preload", true)

	// Run migrations
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Seed default types
	// if err := seedDefaultTypes(db); err != nil {
	// 	return nil, fmt.Errorf("failed to seed default types: %w", err)
	// }

	return db, nil
}

// autoMigrate performs automatic database migrations
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.Provider{},
		&domain.Agent{},
		&domain.AgentType{},
		&domain.ServiceType{},
	)
}

// seedDefaultTypes creates default agent and service types if they don't exist
func seedDefaultTypes(db *gorm.DB) error {
	// Fixed UUIDs for default types
	dummyAgentTypeID := domain.UUID(uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	vmServiceTypeID := domain.UUID(uuid.MustParse("22222222-2222-2222-2222-222222222222"))

	// Create vm service type if it doesn't exist
	vmServiceType := domain.ServiceType{
		BaseEntity: domain.BaseEntity{
			ID: vmServiceTypeID,
		},
		Name: "vm",
	}
	if err := db.FirstOrCreate(&vmServiceType, domain.ServiceType{Name: "vm"}).Error; err != nil {
		return fmt.Errorf("failed to create vm service type: %w", err)
	}

	// Create dummy agent type if it doesn't exist
	dummyAgentType := domain.AgentType{
		BaseEntity: domain.BaseEntity{
			ID: dummyAgentTypeID,
		},
		Name:         "dummy",
		ServiceTypes: []domain.ServiceType{vmServiceType},
	}
	if err := db.FirstOrCreate(&dummyAgentType, domain.AgentType{Name: "dummy"}).Error; err != nil {
		return fmt.Errorf("failed to create dummy agent type: %w", err)
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
