package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type gormMetricEntryRepository struct {
	*GormRepository[domain.MetricEntry]
}

var applyMetricEntryFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"agentId":   parserInFilterFieldApplier("agent_id", domain.ParseUUID),
	"serviceId": parserInFilterFieldApplier("service_id", domain.ParseUUID),
	"typeId":    parserInFilterFieldApplier("type_id", domain.ParseUUID),
})

var applyMetricEntrySort = mapSortApplier(map[string]string{
	"createdAt": "created_at",
	"value":     "value",
})

// NewMetricEntryRepository creates a new instance of MetricEntryRepository
func NewMetricEntryRepository(db *gorm.DB) domain.MetricEntryRepository {
	repo := &gormMetricEntryRepository{
		GormRepository: NewGormRepository[domain.MetricEntry](
			db,
			applyMetricEntryFilter,
			applyMetricEntrySort,
			[]string{"Agent", "Service", "Type"}, // Find preload paths
			[]string{"Agent", "Service", "Type"}, // List preload paths
		),
	}
	return repo
}

// Count returns the number of metric entries matching the given filter
func (r *gormMetricEntryRepository) Count(ctx context.Context, conditions ...interface{}) (int64, error) {
	return r.GormRepository.Count(ctx, conditions...)
}
