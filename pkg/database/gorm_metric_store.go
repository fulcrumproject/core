package database

import (
	"context"

	"gorm.io/gorm"

	"github.com/fulcrumproject/core/pkg/domain"
)

// GormMetricStore implements metric entry operations using GORM
// This store is specifically designed for metric entry operations and uses a separate database
type GormMetricStore struct {
	metricDb        *gorm.DB
	metricEntryRepo domain.MetricEntryRepository
}

// NewGormMetricStore creates a new GormMetricStore instance
func NewGormMetricStore(metricDb *gorm.DB) *GormMetricStore {
	return &GormMetricStore{
		metricDb: metricDb,
	}
}

// Atomic executes the given function within a transaction on the metric database
func (s *GormMetricStore) Atomic(ctx context.Context, fn func(domain.MetricStore) error) error {
	return s.metricDb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txMetricStore := NewGormMetricStore(tx)
		return fn(txMetricStore)
	})
}

// MetricEntryRepo returns the metric entry repository
func (s *GormMetricStore) MetricEntryRepo() domain.MetricEntryRepository {
	if s.metricEntryRepo == nil {
		s.metricEntryRepo = NewMetricEntryRepository(s.metricDb)
	}
	return s.metricEntryRepo
}

// GetMetricDb returns the underlying metric database connection
func (s *GormMetricStore) GetMetricDb() *gorm.DB {
	return s.metricDb
}

// Close closes the metric database connection
func (s *GormMetricStore) Close() error {
	sqlDB, err := s.metricDb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
