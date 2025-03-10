package database

import (
	"fmt"
	"testing"

	"fulcrumproject.org/core/internal/config"
	"gorm.io/gorm"
)

// TestDB contains the database connection and utility functions for tests
type TestDB struct {
	DB     *gorm.DB
	DBName string
}

// NewTestDB creates a new instance of TestDB
func NewTestDB(t *testing.T) *TestDB {
	// Generate a unique database name using UUID without hyphens
	// uuidStr := strings.Replace(uuid.New().String(), "-", "", -1)
	dbName := fmt.Sprintf("fulcrum_test_%s", "db") // uuidStr)
	appConfig := config.DefaultConfig()
	appConfig.LoadFromEnv()

	// Connect to postgres database to create the test database
	adminConfig := getAdminDBConfig(&appConfig.DBConfig)
	adminDB, err := NewConnection(&adminConfig)
	if err != nil {
		t.Fatalf("Failed to connect to postgres database: %v", err)
	}

	// Create the test database
	sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Fatalf("Failed to drop test database: %v", err)
	}

	sql = fmt.Sprintf("CREATE DATABASE %s", dbName)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connect to the test database
	appConfig.DBConfig.Name = dbName
	db, err := NewConnection(&appConfig.DBConfig)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return &TestDB{
		DB:     db,
		DBName: dbName,
	}
}

// Cleanup removes the test database
func (tdb *TestDB) Cleanup(t *testing.T) {
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		t.Errorf("Failed to get underlying *sql.DB: %v", err)
		return
	}

	// Close all database connections
	if err := sqlDB.Close(); err != nil {
		t.Errorf("Failed to close database connection: %v", err)
		return
	}

	// Connect to postgres database to delete the test database
	adminConfig := getAdminDBConfig(nil)
	adminDB, err := NewConnection(&adminConfig)
	if err != nil {
		t.Errorf("Failed to connect to postgres database: %v", err)
		return
	}

	// Force close all connections to the test database
	sql := fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()`,
		tdb.DBName,
	)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Errorf("Failed to terminate database connections: %v", err)
	}

	// Delete the test database
	sql = fmt.Sprintf("DROP DATABASE IF EXISTS %s", tdb.DBName)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Errorf("Failed to drop test database: %v", err)
	}
}

// getAdminDBConfig returns a database configuration for connecting to the postgres admin database
// If baseConfig is nil, it will use defaults from the config package
func getAdminDBConfig(baseConfig *config.DBConfig) config.DBConfig {
	var adminConfig config.DBConfig

	if baseConfig != nil {
		// Copy the base configuration
		adminConfig = *baseConfig
	} else {
		// Use defaults from config package
		defaultConfig := config.DefaultConfig()
		adminConfig = defaultConfig.DBConfig
	}

	// Override the database name to connect to postgres admin database
	adminConfig.Name = "postgres"

	return adminConfig
}
