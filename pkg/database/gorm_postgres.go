package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/gormlock"
	"github.com/fulcrumproject/utils/gormpg"
)

type migrateFn func(*gorm.DB) error

// NewConnection creates a new database connection
func NewConnection(config *gormpg.Conf) (*gorm.DB, error) {
	return connection(config, autoMigrate)
}

func NewMetricConnection(config *gormpg.Conf) (*gorm.DB, error) {
	return connection(config, autoMigrateMetric)
}

func NewLockerConnection(config *gormpg.Conf) (*gorm.DB, error) {
	return connection(config, autoMigrateLocker)
}

func connection(config *gormpg.Conf, fn migrateFn) (*gorm.DB, error) {
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
	if err := fn(db); err != nil {
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
		&domain.Event{},
		&domain.EventSubscription{},
	)
}

func autoMigrateMetric(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.MetricEntry{},
	)
}

func autoMigrateLocker(db *gorm.DB) error {
	return db.AutoMigrate(
		&gormlock.CronJobLock{},
	)
}
