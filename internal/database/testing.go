package database

import (
	"context"
	"fmt"
	"os"
	"testing"

	"gorm.io/gorm"
)

// TestDB contiene la connessione al database e le funzioni di utility per i test
type TestDB struct {
	DB     *gorm.DB
	DBName string
}

// NewTestDB crea una nuova istanza di TestDB
func NewTestDB(t *testing.T) *TestDB {
	// Usa un database di test separato
	dbName := fmt.Sprintf("fulcrum_test_%d", os.Getpid()) // Database univoco per ogni processo di test
	config := Config{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		User:     getEnvOrDefault("TEST_DB_USER", "fulcrum"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", "fulcrum_password"),
		DBName:   dbName,
		Port:     getEnvOrDefault("TEST_DB_PORT", "5432"),
	}

	// Connettiti al database postgres per creare il database di test
	adminConfig := config
	adminConfig.DBName = "postgres"
	adminDB, err := NewConnection(&adminConfig)
	if err != nil {
		t.Fatalf("Failed to connect to postgres database: %v", err)
	}

	// Crea il database di test
	sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Fatalf("Failed to drop test database: %v", err)
	}

	sql = fmt.Sprintf("CREATE DATABASE %s", dbName)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Connettiti al database di test
	db, err := NewConnection(&config)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return &TestDB{
		DB:     db,
		DBName: dbName,
	}
}

// Cleanup elimina il database di test
func (tdb *TestDB) Cleanup(t *testing.T) {
	sqlDB, err := tdb.DB.DB()
	if err != nil {
		t.Errorf("Failed to get underlying *sql.DB: %v", err)
		return
	}

	// Chiudi tutte le connessioni al database
	if err := sqlDB.Close(); err != nil {
		t.Errorf("Failed to close database connection: %v", err)
		return
	}

	// Connettiti al database postgres per eliminare il database di test
	config := Config{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		User:     getEnvOrDefault("TEST_DB_USER", "fulcrum"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", "fulcrum_password"),
		DBName:   "postgres",
		Port:     getEnvOrDefault("TEST_DB_PORT", "5432"),
	}

	adminDB, err := NewConnection(&config)
	if err != nil {
		t.Errorf("Failed to connect to postgres database: %v", err)
		return
	}

	// Forza la chiusura di tutte le connessioni al database di test
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

	// Elimina il database di test
	sql = fmt.Sprintf("DROP DATABASE IF EXISTS %s", tdb.DBName)
	if err := adminDB.Exec(sql).Error; err != nil {
		t.Errorf("Failed to drop test database: %v", err)
	}
}

// TruncateTables elimina tutti i dati dalle tabelle
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

// RunWithinTransaction esegue una funzione all'interno di una transazione
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
