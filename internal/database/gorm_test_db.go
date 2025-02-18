package database

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"fulcrumproject.org/core/internal/env"
	"github.com/google/uuid"
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
	uuidStr := strings.Replace(uuid.New().String(), "-", "", -1)
	dbName := fmt.Sprintf("fulcrum_test_%s", uuidStr)
	config := Config{
		Host:     env.GetOrDefault("TEST_DB_HOST", "localhost"),
		User:     env.GetOrDefault("TEST_DB_USER", "fulcrum"),
		Password: env.GetOrDefault("TEST_DB_PASSWORD", "fulcrum_password"),
		DBName:   dbName,
		Port:     env.GetOrDefault("TEST_DB_PORT", "5432"),
	}

	// Connect to postgres database to create the test database
	adminConfig := config
	adminConfig.DBName = "postgres"
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
	db, err := NewConnection(&config)
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
	config := Config{
		Host:     env.GetOrDefault("TEST_DB_HOST", "localhost"),
		User:     env.GetOrDefault("TEST_DB_USER", "fulcrum"),
		Password: env.GetOrDefault("TEST_DB_PASSWORD", "fulcrum_password"),
		DBName:   "postgres",
		Port:     env.GetOrDefault("TEST_DB_PORT", "5432"),
	}

	adminDB, err := NewConnection(&config)
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

// TruncateTables removes all data from the tables
func (tdb *TestDB) TruncateTables(t *testing.T) {
	tables := []string{
		"agent_type_service_types",
		"agents",
		"service_types",
		"agent_types",
		"providers",
	}

	for _, table := range tables {
		if err := tdb.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
			t.Errorf("Failed to truncate table %s: %v", table, err)
		}
	}
}

// RunWithinTransaction executes a function within a transaction
func (tdb *TestDB) RunWithinTransaction(t *testing.T, fn func(context.Context, *gorm.DB) error) {
	tx := tdb.DB.Begin()
	if tx.Error != nil {
		t.Fatalf("Failed to begin transaction: %v", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			t.Fatalf("Panic in transaction: %v", r)
		}
	}()

	ctx := context.Background()
	if err := fn(ctx, tx); err != nil {
		tx.Rollback()
		t.Fatalf("Transaction failed: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}
