package database

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/properties"
	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

type GormMetricEntryRepository struct {
	*GormRepository[domain.MetricEntry]
}

var applyMetricEntryFilter = MapFilterApplier(map[string]FilterFieldApplier{
	"agentId":    ParserInFilterFieldApplier("metric_entries.agent_id", properties.ParseUUID),
	"serviceId":  ParserInFilterFieldApplier("metric_entries.service_id", properties.ParseUUID),
	"typeId":     ParserInFilterFieldApplier("metric_entries.type_id", properties.ParseUUID),
	"resourceId": StringInFilterFieldApplier("metric_entries.resource_id"),
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

// aggregateSQLExpr maps an AggregateType to its SQL expression.
func aggregateSQLExpr(aggType domain.AggregateType) string {
	switch aggType {
	case domain.AggregateMin:
		return "COALESCE(MIN(value), 0)"
	case domain.AggregateMax:
		return "COALESCE(MAX(value), 0)"
	case domain.AggregateSum:
		return "COALESCE(SUM(value), 0)"
	case domain.AggregateAvg:
		return "COALESCE(AVG(value), 0)"
	case domain.AggregateDiffMaxMin:
		return "COALESCE(MAX(value) - MIN(value), 0)"
	default:
		return "0"
	}
}

type BucketRow struct {
	BucketTime time.Time
	AggValue   float64
}

// Aggregate performs aggregation operations on metric entries for a specific metric type and service within a time range
func (r *GormMetricEntryRepository) Aggregate(ctx context.Context, query domain.AggregateQuery) (domain.AggregationResult, error) {
	if err := query.Aggregate.Validate(); err != nil {
		return domain.AggregationResult{}, err
	}
	if err := query.Bucket.Validate(); err != nil {
		return domain.AggregationResult{}, err
	}

	selectStr := fmt.Sprintf("DATE_TRUNC('%s', created_at) as bucket_time, %s as agg_value", query.Bucket, aggregateSQLExpr(query.Aggregate))

	baseQuery := r.db.WithContext(ctx).
		Model(&domain.MetricEntry{}).Select(selectStr).
		Where("service_id = ? AND type_id = ? AND resource_id = ? AND created_at >= ? AND created_at <= ?", query.ServiceID, query.TypeID, query.ResourceID, query.Start, query.End)

	if query.Scope != nil {
		baseQuery = providerConsumerAgentAuthzFilterApplier(query.Scope, baseQuery)
	}

	var rows []BucketRow
	if err := baseQuery.
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

// AggregateTotal performs a simple scalar aggregation returning a single float64
func (r *GormMetricEntryRepository) AggregateTotal(ctx context.Context, aggregateType domain.AggregateType, serviceID properties.UUID, typeID properties.UUID, start time.Time, end time.Time) (float64, error) {
	if err := aggregateType.Validate(); err != nil {
		return 0, err
	}

	var result float64
	err := r.db.WithContext(ctx).
		Model(&domain.MetricEntry{}).
		Select(aggregateSQLExpr(aggregateType)).
		Where("service_id = ? AND type_id = ? AND created_at >= ? AND created_at <= ?", serviceID, typeID, start, end).
		Scan(&result).Error

	return result, err
}

// ListResourceIDs returns the distinct resource IDs
func (r *GormMetricEntryRepository) ListResourceIDs(ctx context.Context, scope *auth.IdentityScope, page *domain.PageReq) (*domain.PageRes[string], error) {
	baseQuery := r.db.WithContext(ctx).Model(&domain.MetricEntry{})

	baseQuery, err := applyMetricEntryFilter(baseQuery, page)
	if err != nil {
		return nil, err
	}

	if scope != nil {
		baseQuery = providerConsumerAgentAuthzFilterApplier(scope, baseQuery)
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
