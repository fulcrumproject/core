package domain

import (
	"context"
)

// MetricEntry represents a metric measurement for a specific resource
type MetricEntry struct {
	BaseEntity
	AgentID    UUID    `gorm:"not null"`
	ServiceID  UUID    `gorm:"not null"`
	ResourceID string  `gorm:"not null"`
	Value      float64 `gorm:"not null"`
	TypeID     UUID    `gorm:"not null"`

	// Relationships
	Agent   *Agent      `gorm:"foreignKey:AgentID"`
	Service *Service    `gorm:"foreignKey:ServiceID"`
	Type    *MetricType `gorm:"foreignKey:TypeID"`
}

// TableName returns the table name for the metric entry
func (MetricEntry) TableName() string {
	return "metric_entries"
}

// Validate ensures all MetricEntry fields are valid
func (p *MetricEntry) Validate() error {
	// TODO
	return nil
}

// MetricEntryCommander handles provider operations with validation
type MetricEntryCommander struct {
	store Store
}

// NewMetricEntryCommander creates a new MetricEntryService
func NewMetricEntryCommander(
	store Store,
) *MetricEntryCommander {
	return &MetricEntryCommander{
		store: store,
	}
}

// Create creates a new audit-entry with externalID to simplify agent storage need
func (s *MetricEntryCommander) CreateWithExternalID(
	ctx context.Context,
	typeName string,
	agentID UUID,
	externalID string,
	resourceID string,
	value float64,
) (*MetricEntry, error) {
	// Look up the service by external ID
	service, err := s.store.ServiceRepo().FindByExternalID(ctx, agentID, externalID)
	if err != nil {
		return nil, err
	}
	return s.Create(ctx, typeName, agentID, service.ID, resourceID, value)
}

// Create creates a new audit-entry
func (s *MetricEntryCommander) Create(
	ctx context.Context,
	typeName string,
	agentID UUID,
	serviceID UUID,
	resourceID string,
	value float64,
) (*MetricEntry, error) {
	// Look up the service by external ID
	service, err := s.store.ServiceRepo().FindByID(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	// Look up the service type by name
	metricType, err := s.store.MetricTypeRepo().FindByName(ctx, typeName)
	if err != nil {
		return nil, err
	}
	// TODO check id's exist with cache
	metricEntry := &MetricEntry{
		AgentID:    agentID,
		ServiceID:  service.ID,
		ResourceID: resourceID,
		Value:      value,
		TypeID:     metricType.ID,
	}
	if err := metricEntry.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.MetricEntryRepo().Create(ctx, metricEntry); err != nil {
		return nil, err
	}
	return metricEntry, nil
}

type MetricEntryRepository interface {
	// Create creates a new metric entry
	Create(ctx context.Context, entity *MetricEntry) error

	// List retrieves a list of metric entries based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[MetricEntry], error)

	// CountByMetricType counts the number of entries for a specific metric type
	CountByMetricType(ctx context.Context, typeID UUID) (int64, error)
}

type MetricEntryQuerier interface {
	// List retrieves a list of metric entries based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[MetricEntry], error)

	// CountByMetricType counts the number of entries for a specific metric type
	CountByMetricType(ctx context.Context, typeID UUID) (int64, error)
}
