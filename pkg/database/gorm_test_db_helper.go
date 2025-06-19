package database

import (
	"fmt"
	"regexp"
	"testing"

	"fulcrumproject.org/core/pkg/config"
	cb "github.com/fulcrumproject/commons/config"
	"gorm.io/gorm"
)

// TestDB contains the database connection and utility functions for tests
type TestDB struct {
	DB     *gorm.DB
	DBName string
}

// NewTestDB creates a new instance of TestDB
func NewTestDB(t *testing.T) *TestDB {
	// Generate a unique database name using properties.UUID without hyphens
	// uuidStr := strings.Replace(uuid.New().String(), "-", "", -1)
	dbName := fmt.Sprintf("fulcrum_test_%s", "db") // uuidStr)

	defaultConfig := config.Default
	appConfig, err := cb.New(&defaultConfig,
		cb.WithEnvPrefix[*config.Config](config.EnvPrefix),
		cb.WithEnvFiles[*config.Config](".env")).
		WithEnv().
		Build()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	// Connect to default fulcrum database to create the test database
	adminDB, err := NewConnection(&appConfig.DBConfig)
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
	appConfig.DBConfig.DSN = replaceDatabaseInDSN(appConfig.DBConfig.DSN, dbName)
	db, err := NewConnection(&appConfig.DBConfig)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return &TestDB{
		DB:     db,
		DBName: dbName,
	}
}

// replaceDatabaseInDSN replaces the database name in a PostgreSQL DSN string
// Format: "host=localhost user=fulcrum password=password dbname=fulcrum_db port=5432 sslmode=disable"
func replaceDatabaseInDSN(dsn, newDBName string) string {
	re := regexp.MustCompile(`dbname=\S+`)
	return re.ReplaceAllString(dsn, "dbname="+newDBName)
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
	defaultConfig := config.Default
	appConfig, err := cb.New(&defaultConfig,
		cb.WithEnvPrefix[*config.Config](config.EnvPrefix),
		cb.WithEnvFiles[*config.Config](".env")).
		WithEnv().
		Build()
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}

	adminDB, err := NewConnection(&appConfig.DBConfig)
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
