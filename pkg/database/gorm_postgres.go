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

	// Enable UUID v7 function for generating timestamp-based UUIDs
	if err := enableUUIDv7Extension(db); err != nil {
		return nil, fmt.Errorf("failed to enable UUID v7 function: %w", err)
	}

	// Run migrations
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// enableUUIDv7Extension creates a UUID v7 generation function using pure SQL
func enableUUIDv7Extension(db *gorm.DB) error {
	// Create a pure SQL implementation of UUID v7 generation
	// This is based on the widely-used implementation from the community
	// and will be compatible with PostgreSQL 18's native uuidv7() function
	uuidv7Function := `
CREATE OR REPLACE FUNCTION uuid_generate_v7()
RETURNS uuid
AS $$
BEGIN
    -- use random v4 uuid as starting point (which has the same variant we need)
    -- then overlay timestamp
    -- then set version 7 by flipping the 2 and 1 bit in the version 4 string
    RETURN encode(
        set_bit(
            set_bit(
                overlay(uuid_send(gen_random_uuid())
                    placing substring(int8send(floor(extract(epoch from clock_timestamp()) * 1000)::bigint) from 3)
                    from 1 for 6
                ),
                52, 1
            ),
            53, 1
        ),
        'hex')::uuid;
END
$$
LANGUAGE plpgsql
VOLATILE;`

	if err := db.Exec(uuidv7Function).Error; err != nil {
		return fmt.Errorf("failed to create uuid_generate_v7 function: %w", err)
	}
	return nil
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
