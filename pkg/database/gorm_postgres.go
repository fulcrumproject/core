package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/utils/gormpg"
)

// NewConnection creates a new database connection
func NewConnection(config *gormpg.Conf) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: config.DSN,
	}), &gorm.Config{
		Logger:                                   gormpg.NewGormLogger(config),
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
