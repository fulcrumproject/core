package database

import (
	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormAuditEntryRepository struct {
	*GormRepository[domain.AuditEntry]
}

var applyAuditEntryFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"authorityType": stringInFilterFieldApplier("authority_type"),
	"authorityId":   parserInFilterFieldApplier("authority_id", domain.ParseUUID),
	"type":          stringInFilterFieldApplier("type"),
})

var applyAuditEntrySort = mapSortApplier(map[string]string{
	"createdAt": "created_at",
})

// NewAuditEntryRepository creates a new instance of AuditEntryRepository
func NewAuditEntryRepository(db *gorm.DB) domain.AuditEntryRepository {
	repo := &gormAuditEntryRepository{
		GormRepository: NewGormRepository[domain.AuditEntry](
			db,
			applyAuditEntryFilter,
			applyAuditEntrySort,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}
