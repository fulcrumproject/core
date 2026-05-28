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
	if err := migrateConfigPoolScope(db); err != nil {
		return err
	}
	if err := migrateInstallTokens(db); err != nil {
		return err
	}

	err := db.AutoMigrate(
		&domain.Token{},
		&domain.Participant{},
		&domain.Agent{},
		&domain.InstallToken{},
		&domain.AgentType{},
		&domain.InfrastructureType{},
		&domain.Infrastructure{},
		&domain.ConfigPool{},
		&domain.ConfigPoolValue{},
		&domain.ServiceType{},
		&domain.ServiceGroup{},
		&domain.Service{},
		&domain.ServiceOptionType{},
		&domain.ServiceOption{},
		&domain.ServicePoolSet{},
		&domain.ServicePool{},
		&domain.ServicePoolValue{},
		&domain.Job{},
		&domain.MetricType{},
		&domain.Event{},
		&domain.EventSubscription{},
		&vaultSecret{},
	)
	if err != nil {
		return err
	}

	if err := backfillConfigPoolValueParticipant(db); err != nil {
		return err
	}

	return backfillServicePoolParticipant(db)
}

// backfillConfigPoolValueParticipant copies participant_id from the parent pool onto
// rows that predate the denormalization. Idempotent — the IS NULL guard keeps it safe
// to re-run on every boot. Must run after AutoMigrate since the column it writes to
// is introduced by AutoMigrate on first upgrade.
func backfillConfigPoolValueParticipant(db *gorm.DB) error {
	res := db.Exec(`
		UPDATE config_pool_values v
		SET participant_id = p.participant_id
		FROM config_pools p
		WHERE v.config_pool_id = p.id
		  AND v.participant_id IS NULL
		  AND p.participant_id IS NOT NULL
	`)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		db.Logger.Info(db.Statement.Context, "backfilled participant_id on %d config_pool_values rows", res.RowsAffected)
	}
	return nil
}

// backfillServicePoolParticipant copies provider_id from the parent pool set onto
// service_pools, then propagates to service_pool_values. Idempotent via IS NULL guards.
// Must run after AutoMigrate since the columns it writes to are introduced there.
func backfillServicePoolParticipant(db *gorm.DB) error {
	res := db.Exec(`
		UPDATE service_pools sp
		SET participant_id = s.provider_id
		FROM service_pool_sets s
		WHERE sp.service_pool_set_id = s.id
		  AND sp.participant_id IS NULL
	`)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		db.Logger.Info(db.Statement.Context, "backfilled participant_id on %d service_pools rows", res.RowsAffected)
	}

	res = db.Exec(`
		UPDATE service_pool_values v
		SET participant_id = p.participant_id
		FROM service_pools p
		WHERE v.service_pool_id = p.id
		  AND v.participant_id IS NULL
		  AND p.participant_id IS NOT NULL
	`)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		db.Logger.Info(db.Statement.Context, "backfilled participant_id on %d service_pool_values rows", res.RowsAffected)
	}
	return nil
}

// migrateInstallTokens promotes the legacy agent-only install-token table to
// the generic install_tokens table that serves both Agent and Infrastructure.
// Guarded by HasTable / HasColumn so fresh DBs skip cleanly and re-runs are
// idempotent: AutoMigrate creates the new shape on first boot of a fresh DB,
// while upgrades from a Phase 2 schema rename the table + column and add
// entity_type defaulting to "agent" so existing rows stay valid.
func migrateInstallTokens(db *gorm.DB) error {
	m := db.Migrator()

	// 1. Rename the table if the legacy name exists and the new one doesn't.
	if m.HasTable("agent_install_tokens") && !m.HasTable("install_tokens") {
		if err := db.Exec("ALTER TABLE agent_install_tokens RENAME TO install_tokens").Error; err != nil {
			return fmt.Errorf("rename agent_install_tokens: %w", err)
		}
	}
	if !m.HasTable("install_tokens") {
		return nil // fresh DB; AutoMigrate will create install_tokens correctly
	}

	// 2. Rename agent_id -> entity_id.
	if m.HasColumn("install_tokens", "agent_id") && !m.HasColumn("install_tokens", "entity_id") {
		if err := db.Exec("ALTER TABLE install_tokens RENAME COLUMN agent_id TO entity_id").Error; err != nil {
			return fmt.Errorf("rename agent_id column: %w", err)
		}
	}

	// 3. Add entity_type with default 'agent', backfilling pre-existing rows
	//    in one shot. AutoMigrate would add it later but without the default,
	//    so legacy rows would fail the NOT NULL.
	if !m.HasColumn("install_tokens", "entity_type") {
		if err := db.Exec("ALTER TABLE install_tokens ADD COLUMN entity_type text NOT NULL DEFAULT 'agent'").Error; err != nil {
			return fmt.Errorf("add entity_type column: %w", err)
		}
	}

	// 4. Drop the legacy single-column unique on agent_id (whichever name it
	//    carries — gorm names vary across versions). AutoMigrate recreates
	//    the composite unique on (entity_type, entity_id) from the struct tag.
	for _, c := range []string{
		"uni_agent_install_tokens_agent_id",
		"agent_install_tokens_agent_id_key",
		"uni_install_tokens_entity_id",
		"install_tokens_entity_id_key",
	} {
		if err := db.Exec(fmt.Sprintf("ALTER TABLE install_tokens DROP CONSTRAINT IF EXISTS %s", c)).Error; err != nil {
			return fmt.Errorf("drop %s: %w", c, err)
		}
	}
	return nil
}

func migrateConfigPoolScope(db *gorm.DB) error {
	m := db.Migrator()

	// Step 1: drop the legacy global UNIQUE on Type. It came from `gorm:"unique"` and is
	// implemented as a CONSTRAINT (which owns its backing index), so DROP INDEX fails with
	// "constraint requires it" — we have to DROP CONSTRAINT, which cascades to the index.
	// The constraint's table-of-record may still be agent_pools (rename not done yet) or
	// already config_pools (Postgres preserves constraint names across RENAME), so try both.
	cleanups := []struct{ table, constraint string }{
		{"agent_pools", "agent_pools_type_key"},
		{"agent_pools", "uni_agent_pools_type"},
		{"config_pools", "agent_pools_type_key"},
		{"config_pools", "uni_agent_pools_type"},
		{"config_pools", "config_pools_type_key"},
		{"config_pools", "uni_config_pools_type"},
	}
	for _, c := range cleanups {
		if !m.HasTable(c.table) {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s", c.table, c.constraint)
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}

	// Step 2: rename tables. Idempotent — guarded so a fresh DB and a re-run both skip cleanly.
	if m.HasTable("agent_pools") && !m.HasTable("config_pools") {
		if err := db.Exec("ALTER TABLE agent_pools RENAME TO config_pools").Error; err != nil {
			return fmt.Errorf("rename agent_pools: %w", err)
		}
	}
	if m.HasTable("agent_pool_values") && !m.HasTable("config_pool_values") {
		if err := db.Exec("ALTER TABLE agent_pool_values RENAME TO config_pool_values").Error; err != nil {
			return fmt.Errorf("rename agent_pool_values: %w", err)
		}
	}

	// Step 3: rename the FK column on the (possibly already renamed) values table.
	if m.HasTable("config_pool_values") &&
		m.HasColumn("config_pool_values", "agent_pool_id") &&
		!m.HasColumn("config_pool_values", "config_pool_id") {
		if err := db.Exec("ALTER TABLE config_pool_values RENAME COLUMN agent_pool_id TO config_pool_id").Error; err != nil {
			return fmt.Errorf("rename agent_pool_id column: %w", err)
		}
	}

	return nil
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
