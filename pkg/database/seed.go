// Seed initializes administrative access once, or adopts existing Core data.
// A durable transactional guard prevents restart from undoing credential revocation.
package database

import (
	"context"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Presence of this row means bootstrap completed, even if every token is gone.
// It is committed atomically with initial token creation (or store adoption).
type bootstrapState struct {
	Name string `gorm:"primaryKey"`
}

func (bootstrapState) TableName() string { return "bootstrap_states" }

func Seed(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// PostgreSQL's unique-key conflict waits for a concurrent initializer to
		// commit or roll back. Only the transaction that inserts may bootstrap.
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&bootstrapState{Name: "administrative-bootstrap"})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		existing, err := hasCoreData(tx)
		if err != nil || existing {
			return err
		}

		// First-setup-only development credential: regenerate before normal use.
		const value = "change-me"
		return NewTokenRepository(tx).Create(context.Background(), &domain.Token{
			BaseEntity:  domain.BaseEntity{ID: uuid.New()},
			Name:        "Admin Token",
			PlainValue:  value,
			HashedValue: domain.HashTokenValue(value),
			Role:        auth.RoleAdmin,
			ExpireAt:    time.Now().AddDate(0, 0, 1),
		})
	})
}

func hasCoreData(db *gorm.DB) (bool, error) {
	for _, model := range coreModels() {
		var exists bool
		query := db.Model(model).Select("1").Limit(1)
		if err := db.Raw("SELECT EXISTS (?)", query).Scan(&exists).Error; err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
	}
	// Associations can outlive their entities; metrics may share the Core DB
	// but are migrated separately. Scheduler locks are not domain history.
	for _, table := range []string{"agent_type_service_types", "agent_type_infrastructure_types", "metric_entries"} {
		if !db.Migrator().HasTable(table) {
			continue
		}
		var exists bool
		query := db.Table(table).Select("1").Limit(1)
		if err := db.Raw("SELECT EXISTS (?)", query).Scan(&exists).Error; err != nil {
			return false, err
		}
		if exists {
			return true, nil
		}
	}
	return false, nil
}
