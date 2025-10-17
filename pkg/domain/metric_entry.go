package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AggregateType defines the type of aggregation to perform on metric entries
type AggregateType string

const (
	// AggregateMax returns the maximum value
	AggregateMax AggregateType = "max"
	// AggregateSum returns the sum of values
	AggregateSum AggregateType = "sum"
	// AggregateDiffMaxMin returns the difference between maximum and minimum values (for always increasing metrics)
	AggregateDiffMaxMin AggregateType = "diff"
	// AggregateAvg returns the average value
	AggregateAvg AggregateType = "avg"
)

// MetricEntry represents a metric measurement for a specific resource
// Does not extend BaseEntity because it has a custom index on created_at
type MetricEntry struct {
	// Base entity fields
	ID        properties.UUID `json:"id" gorm:"type:uuid;primary_key"`
	CreatedAt time.Time       `json:"-" gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_metric_aggregate,priority:3"`
	UpdatedAt time.Time       `json:"-" gorm:"not null;default:CURRENT_TIMESTAMP"`

	ResourceID string  `gorm:"not null"`
	Value      float64 `gorm:"not null"`

	// Relationships
	TypeID     properties.UUID `gorm:"not null;index:idx_metric_aggregate,priority:2"`
	Type       *MetricType     `gorm:"foreignKey:TypeID"`
	AgentID    properties.UUID `gorm:"not null"`
	Agent      *Agent          `gorm:"foreignKey:AgentID"`
	ServiceID  properties.UUID `gorm:"not null;index:idx_metric_aggregate,priority:1"`
	Service    *Service        `gorm:"foreignKey:ServiceID"`
	ProviderID properties.UUID `gorm:"not null"`
	Provider   *Participant    `gorm:"foreignKey:ProviderID"`
	ConsumerID properties.UUID `gorm:"not null"`
	Consumer   *Participant    `gorm:"foreignKey:ConsumerID"`
}

// NewMetricEntry creates a new metric entry
func NewMetricEntry(
	consumerID properties.UUID,
	providerID properties.UUID,
	agentID properties.UUID,
	serviceID properties.UUID,
	resourceID string,
	typeID properties.UUID,
	value float64,
) *MetricEntry {
	return &MetricEntry{
		ConsumerID: consumerID,
		ProviderID: providerID,
		AgentID:    agentID,
		ServiceID:  serviceID,
		ResourceID: resourceID,
		TypeID:     typeID,
		Value:      value,
	}
}

// TableName returns the table name for the metric entry
func (MetricEntry) TableName() string {
	return "metric_entries"
}

// GetID returns the entity's ID (implements Entity interface)
func (m MetricEntry) GetID() properties.UUID {
	return m.ID
}

// BeforeCreate ensures properties.UUID is set before creating a record
func (m *MetricEntry) BeforeCreate(tx *gorm.DB) error {
	if uuid.UUID(m.ID) == uuid.Nil {
		m.ID = properties.NewUUID()
	}
	return nil
}

// Validate ensures all MetricEntry fields are valid
func (p *MetricEntry) Validate() error {
	if p.ResourceID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	if p.TypeID == uuid.Nil {
		return fmt.Errorf("metric type ID cannot be empty")
	}
	if p.AgentID == uuid.Nil {
		return fmt.Errorf("agent ID cannot be empty")
	}
	if p.ServiceID == uuid.Nil {
		return fmt.Errorf("service ID cannot be empty")
	}
	return nil
}

// MetricEntryCommander defines the interface for metric entry command operations
type MetricEntryCommander interface {
	// Create creates a new metric entry
	Create(ctx context.Context, params CreateMetricEntryParams) (*MetricEntry, error)

	// CreateWithAgentInstanceID creates a new metric entry using service's external ID
	CreateWithAgentInstanceID(ctx context.Context, params CreateMetricEntryWithAgentInstanceIDParams) (*MetricEntry, error)
}

type CreateMetricEntryParams struct {
	TypeName   string          `json:"typeName"`
	AgentID    properties.UUID `json:"agentId"`
	ServiceID  properties.UUID `json:"serviceId"`
	ResourceID string          `json:"resourceId"`
	Value      float64         `json:"value"`
}

type CreateMetricEntryWithAgentInstanceIDParams struct {
	TypeName        string          `json:"typeName"`
	AgentID         properties.UUID `json:"agentId"`
	AgentInstanceID string          `json:"agentInstanceId"`
	ResourceID      string          `json:"resourceId"`
	Value           float64         `json:"value"`
}

// metricEntryCommander is the concrete implementation of MetricEntryCommander
type metricEntryCommander struct {
	store           Store
	metricEntryRepo MetricEntryRepository
}

// NewMetricEntryCommander creates a new MetricEntryCommander
func NewMetricEntryCommander(
	store Store,
	metricEntryRepo MetricEntryRepository,
) *metricEntryCommander {
	return &metricEntryCommander{
		store:           store,
		metricEntryRepo: metricEntryRepo,
	}
}

func (s *metricEntryCommander) CreateWithAgentInstanceID(
	ctx context.Context,
	params CreateMetricEntryWithAgentInstanceIDParams,
) (*MetricEntry, error) {
	// 1. Validate agent exists
	ok, err := s.store.AgentRepo().Exists(ctx, params.AgentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, NewInvalidInputErrorf("invalid agent ID %s", params.AgentID)
	}

	// 2. Find service by agent instance ID
	svc, err := s.store.ServiceRepo().FindByAgentInstanceID(ctx, params.AgentID, params.AgentInstanceID)
	if err != nil {
		return nil, err
	}

	// 3. Validate type compatibility
	metricType, err := s.store.MetricTypeRepo().FindByName(ctx, params.TypeName)
	if err != nil {
		return nil, err
	}

	// 4. Check metric type exists
	metricTypeExists, err := s.store.MetricTypeRepo().Exists(ctx, metricType.ID)
	if err != nil {
		return nil, err
	}
	if !metricTypeExists {
		return nil, InvalidInputError{Err: fmt.Errorf("metric type with ID %s does not exist", metricType.ID)}
	}

	// 5. Create and validate
	metricEntry := NewMetricEntry(
		svc.ConsumerID,
		svc.ProviderID,
		params.AgentID,
		svc.ID,
		params.ResourceID,
		metricType.ID,
		params.Value,
	)

	if err := metricEntry.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// 6. Save
	if err := s.metricEntryRepo.Create(ctx, metricEntry); err != nil {
		return nil, err
	}

	return metricEntry, nil
}

func (s *metricEntryCommander) Create(
	ctx context.Context,
	params CreateMetricEntryParams,
) (*MetricEntry, error) {
	// 1. Validate agent exists
	ok, err := s.store.AgentRepo().Exists(ctx, params.AgentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, NewInvalidInputErrorf("invalid agent ID %s", params.AgentID)
	}

	// 2. Find service
	svc, err := s.store.ServiceRepo().Get(ctx, params.ServiceID)
	if err != nil {
		return nil, err
	}

	// 3. Validate type compatibility
	metricType, err := s.store.MetricTypeRepo().FindByName(ctx, params.TypeName)
	if err != nil {
		return nil, err
	}

	// 4. Check metric type exists
	metricTypeExists, err := s.store.MetricTypeRepo().Exists(ctx, metricType.ID)
	if err != nil {
		return nil, err
	}
	if !metricTypeExists {
		return nil, InvalidInputError{Err: fmt.Errorf("metric type with ID %s does not exist", metricType.ID)}
	}

	// 5. Create and validate
	metricEntry := NewMetricEntry(
		svc.ConsumerID,
		svc.ProviderID,
		params.AgentID,
		svc.ID,
		params.ResourceID,
		metricType.ID,
		params.Value,
	)

	if err := metricEntry.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// 6. Save
	if err := s.metricEntryRepo.Create(ctx, metricEntry); err != nil {
		return nil, err
	}

	return metricEntry, nil
}

type MetricEntryRepository interface {
	MetricEntryQuerier
	BaseEntityRepository[MetricEntry]
}

type MetricEntryQuerier interface {
	BaseEntityQuerier[MetricEntry]

	// CountByMetricType counts the number of entries for a specific metric type
	CountByMetricType(ctx context.Context, typeID properties.UUID) (int64, error)

	// Aggregate performs aggregation operations on metric entries for a specific metric type and service within a time range
	Aggregate(ctx context.Context, aggregateType AggregateType, serviceID properties.UUID, typeID properties.UUID, start time.Time, end time.Time) (float64, error)
}
