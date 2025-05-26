package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormAuditEntryRepository struct {
	*GormRepository[domain.AuditEntry]
}

var applyAuditEntryFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"authorityType": stringInFilterFieldApplier("authority_type"),
	"authorityId":   parserInFilterFieldApplier("authority_id", domain.ParseUUID),
	"eventType":     stringInFilterFieldApplier("event_type"),
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
			providerConsumerAgentAuthzFilterApplier,
			[]string{}, // No preload paths needed
			[]string{}, // No preload paths needed
		),
	}
	return repo
}

func (r *GormAuditEntryRepository) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthTargetScope, error) {
	return r.getAuthScope(ctx, id, "provider_id", "consumer_id", "agent_id")
}
