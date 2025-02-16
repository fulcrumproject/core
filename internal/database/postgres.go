package database

import (
	"fmt"

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

// NewConnection creates a new database connection
func NewConnection(config *Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: config.DSN(),
	}), &gorm.Config{
		Logger: logger.Default.LogMode(getLogLevelFromEnv("DB_LOG_LEVEL", logger.Info)),
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
