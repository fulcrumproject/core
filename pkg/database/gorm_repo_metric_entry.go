package database

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormMetricEntryRepository struct {
	*GormRepository[domain.MetricEntry]
}

var applyMetricEntryFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"agentId":   ParserInFilterFieldApplier("metric_entries.agent_id", properties.ParseUUID),
	"serviceId": ParserInFilterFieldApplier("metric_entries.service_id", properties.ParseUUID),
	"typeId":    ParserInFilterFieldApplier("metric_entries.type_id", properties.ParseUUID),
})

var applyMetricEntrySort = MapSortApplier(map[string]string{
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
func (r *GormMetricEntryRepository) CountByMetricType(ctx context.Context, typeID properties.UUID) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&domain.MetricEntry{}).
		Where("type_id = ?", typeID).
		Count(&count)
	return count, result.Error
}

// Aggregate performs aggregation operations on metric entries for a specific metric type and service within a time range
func (r *GormMetricEntryRepository) Aggregate(ctx context.Context, aggregateType domain.AggregateType, serviceID properties.UUID, typeID properties.UUID, start time.Time, end time.Time) (float64, error) {
	var result float64
	var err error

	baseQuery := r.db.WithContext(ctx).
		Model(&domain.MetricEntry{}).
		Where("service_id = ? AND type_id = ? AND created_at >= ? AND created_at <= ?", serviceID, typeID, start, end)

	switch aggregateType {
	case domain.AggregateMax:
		err = baseQuery.Select("COALESCE(MAX(value), 0)").Scan(&result).Error
	case domain.AggregateSum:
		err = baseQuery.Select("COALESCE(SUM(value), 0)").Scan(&result).Error
	case domain.AggregateDiffMaxMin:
		err = baseQuery.Select("COALESCE(MAX(value) - MIN(value), 0)").Scan(&result).Error
	case domain.AggregateAvg:
		err = baseQuery.Select("COALESCE(AVG(value), 0)").Scan(&result).Error
	default:
		return 0, fmt.Errorf("unsupported aggregate type: %s", aggregateType)
	}

	return result, err
}

func (r *GormMetricEntryRepository) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "null", "provider_id", "agent_id", "consumer_id")
}
