package database

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/authz"
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
func NewMetricEntryRepository(metricDb *gorm.DB) *GormMetricEntryRepository {
	repo := &GormMetricEntryRepository{
		GormRepository: NewGormRepository[domain.MetricEntry](
			metricDb,
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

type BucketRow struct {
	BucketTime time.Time
	AggValue   float64
}

// Aggregate performs aggregation operations on metric entries for a specific metric type and service within a time range
func (r *GormMetricEntryRepository) Aggregate(ctx context.Context, query domain.AggregateQuery) (domain.AggregationResult, error) {
	selectStr := fmt.Sprintf("DATE_TRUNC('%s', created_at) as bucket_time, COALESCE(%s(value), 0) as agg_value", query.Bucket, query.Aggregate)

	var rows []BucketRow
	if err := r.db.WithContext(ctx).
		Model(&domain.MetricEntry{}).Select(selectStr).
		Where("service_id = ? AND type_id = ? AND resource_id = ? AND created_at >= ? AND created_at <= ?", query.ServiceID, query.TypeID, query.ResourceID, query.Start, query.End).
		Group("bucket_time").Order("bucket_time").Scan(&rows).Error; err != nil {
		return domain.AggregationResult{}, err
	}

	data := make([]domain.AggregateData, len(rows))
	for i, row := range rows {
		data[i] = domain.AggregateData{row.BucketTime.Format(time.RFC3339), row.AggValue}
	}

	return domain.AggregationResult{
		Data:      data,
		Aggregate: query.Aggregate,
		Bucket:    query.Bucket,
		Start:     query.Start,
		End:       query.End,
	}, nil
}

// ListResourceIDs returns the distinct resource IDs
func (r *GormMetricEntryRepository) ListResourceIDs(ctx context.Context, page *domain.PageReq) (*domain.PageRes[string], error) {
	baseQuery := r.db.WithContext(ctx).Model(&domain.MetricEntry{})

	baseQuery, err := applyMetricEntryFilter(baseQuery, page)
	if err != nil {
		return nil, err
	}

	var count int64
	if err := baseQuery.Distinct("resource_id").Count(&count).Error; err != nil {
		return nil, err
	}

	var resourceIds []string
	offset := (page.Page - 1) * page.PageSize
	if err := baseQuery.Distinct("resource_id").Offset(offset).Limit(page.PageSize).Pluck("resource_id", &resourceIds).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(resourceIds, count, page), nil
}

func (r *GormMetricEntryRepository) AuthScope(ctx context.Context, id properties.UUID) (authz.ObjectScope, error) {
	return r.AuthScopeByFields(ctx, id, "null", "provider_id", "agent_id", "consumer_id")
}
