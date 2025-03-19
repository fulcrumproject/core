package database

import (
	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormAuditEntryRepository struct {
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
func NewAuditEntryRepository(db *gorm.DB) *GormAuditEntryRepository {
	repo := &GormAuditEntryRepository{
		GormRepository: NewGormRepository[domain.AuditEntry](
			db,
			applyAuditEntryFilter,
			applyAuditEntrySort,
			auditEntryAuthzFilterApplier,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

// auditEntryAuthzFilterApplier applies authorization scoping to audit entry queries
func auditEntryAuthzFilterApplier(s *domain.AuthScope, q *gorm.DB) *gorm.DB {
	// TODO authz filter
	return q
}
