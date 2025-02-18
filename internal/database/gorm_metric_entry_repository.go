package database

import (
	"context"

	"gorm.io/gorm"

	"fulcrumproject.org/core/internal/domain"
)

type metricEntryRepository struct {
	db *gorm.DB
}

// NewMetricEntryRepository creates a new instance of MetricEntryRepository
func NewMetricEntryRepository(db *gorm.DB) domain.MetricEntryRepository {
	return &metricEntryRepository{db: db}
}

func (r *metricEntryRepository) Create(ctx context.Context, metricEntry *domain.MetricEntry) error {
	result := r.db.WithContext(ctx).Create(metricEntry)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

var metricEntryFilterConfigs = map[string]FilterConfig{
	"agentId":   {Query: "agent_id", Valuer: func(v string) (interface{}, error) { return domain.ParseUUID(v) }},
	"serviceId": {Query: "service_id", Valuer: func(v string) (interface{}, error) { return domain.ParseUUID(v) }},
	"typeId":    {Query: "type_id", Valuer: func(v string) (interface{}, error) { return domain.ParseUUID(v) }},
}

func (r *metricEntryRepository) List(ctx context.Context, filter *domain.SimpleFilter, sorting *domain.Sorting, pagination *domain.Pagination) (*domain.PaginatedResult[domain.MetricEntry], error) {
	var metricEntries []domain.MetricEntry
	var totalItems int64

	query := r.db.WithContext(ctx).Model(&domain.MetricEntry{}).
		Preload("Agent").
		Preload("Service").
		Preload("Type")

	query, totalItems, err := applyFindAndCount(query, filter, metricEntryFilterConfigs, sorting, pagination)
	if err != nil {
		return nil, err
	}
	if err := query.Find(&metricEntries).Error; err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(metricEntries, totalItems, pagination), nil
}

func (r *metricEntryRepository) Count(ctx context.Context, filter *domain.SimpleFilter) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.MetricEntry{})
	_, count, err := applyFilterAndCount(query, filter, metricEntryFilterConfigs)
	if err != nil {
		return 0, err
	}
	return count, nil
}
