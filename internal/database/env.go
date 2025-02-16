package database

import (
	"os"

	"gorm.io/gorm/logger"
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
