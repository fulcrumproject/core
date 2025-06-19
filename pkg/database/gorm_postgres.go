package database

import (
	"fmt"

	"github.com/fulcrumproject/commons/config"
	"github.com/fulcrumproject/commons/logging"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"fulcrumproject.org/core/pkg/domain"
)

// NewConnection creates a new database connection
func NewConnection(config *config.DB) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: config.DSN,
	}), &gorm.Config{
		Logger:                                   logging.NewGormLogger(config),
		DisableForeignKeyConstraintWhenMigrating: true,
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
		&domain.Token{},
		&domain.Participant{},
		&domain.Agent{},
		&domain.AgentType{},
		&domain.ServiceType{},
		&domain.ServiceGroup{},
		&domain.Service{},
		&domain.Job{},
		&domain.MetricType{},
		&domain.MetricEntry{},
		&domain.Event{},
		&domain.EventSubscription{},
	)
}
