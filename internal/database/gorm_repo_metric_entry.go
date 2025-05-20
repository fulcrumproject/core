package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type GormMetricEntryRepository struct {
	*GormRepository[domain.MetricEntry]
}

var applyMetricEntryFilter = mapFilterApplier(map[string]FilterFieldApplier{
	"agentId":   parserInFilterFieldApplier("metric_entries.agent_id", domain.ParseUUID),
	"serviceId": parserInFilterFieldApplier("metric_entries.service_id", domain.ParseUUID),
	"typeId":    parserInFilterFieldApplier("metric_entries.type_id", domain.ParseUUID),
})

var applyMetricEntrySort = mapSortApplier(map[string]string{
	"createdAt": "metric_entries.created_at",
	"value":     "metric_entries.value",
})

// NewMetricEntryRepository creates a new instance of MetricEntryRepository
func NewMetricEntryRepository(db *gorm.DB) *GormMetricEntryRepository {
	repo := &GormMetricEntryRepository{
		GormRepository: NewGormRepository[domain.MetricEntry](
			db,
			applyMetricEntryFilter,
			applyMetricEntrySort,
			providerConsumerAgentAuthzFilterApplier,
			[]string{"Agent", "Service", "Type"}, // Find preload paths
			[]string{"Agent", "Service", "Type"}, // List preload paths
		),
	}
	return repo
}

// CountByMetricType counts the number of entries for a specific metric type
func (r *GormMetricEntryRepository) CountByMetricType(ctx context.Context, typeID domain.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&domain.MetricEntry{}).
		Where("type_id = ?", typeID).
		Count(&count)
	return count, result.Error
}

func (r *GormMetricEntryRepository) AuthScope(ctx context.Context, id domain.UUID) (*domain.AuthScope, error) {
	return r.getAuthScope(ctx, id, "provider_id", "consumer_id", "agent_id")
}
