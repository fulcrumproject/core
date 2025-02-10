package database

import (
	"fmt"
	"os"

	"fulcrumproject.org/core/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     string
}

func NewConnection(config *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.Host,
		config.User,
		config.Password,
		config.DBName,
		config.Port,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&domain.Item{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func NewConfigFromEnv() *Config {
	return &Config{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		User:     getEnvOrDefault("DB_USER", "fulcrum"),
		Password: getEnvOrDefault("DB_PASSWORD", "fulcrum_password"),
		DBName:   getEnvOrDefault("DB_NAME", "fulcrum_db"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
