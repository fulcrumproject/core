package domain

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// MetricEntry represents a metric measurement for a specific resource
type MetricEntry struct {
	BaseEntity

	ResourceID string  `gorm:"not null"`
	Value      float64 `gorm:"not null"`

	// Relationships
	TypeID     UUID        `gorm:"not null"`
	Type       *MetricType `gorm:"foreignKey:TypeID"`
	AgentID    UUID        `gorm:"not null"`
	Agent      *Agent      `gorm:"foreignKey:AgentID"`
	ServiceID  UUID        `gorm:"not null"`
	Service    *Service    `gorm:"foreignKey:ServiceID"`
	ProviderID UUID        `gorm:"not null"`
	Provider   *Provider   `gorm:"foreignKey:ProviderID"`
	BrokerID   UUID        `gorm:"not null"`
	Broker     *Broker     `gorm:"foreignKey:BrokerID"`
}

// TableName returns the table name for the metric entry
func (MetricEntry) TableName() string {
	return "metric_entries"
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
	Create(ctx context.Context, typeName string, agentID UUID, serviceID UUID, resourceID string, value float64) (*MetricEntry, error)

	// CreateWithExternalID creates a new metric entry using service's external ID
	CreateWithExternalID(ctx context.Context, typeName string, agentID UUID, externalID string, resourceID string, value float64) (*MetricEntry, error)
}

// metricEntryCommander is the concrete implementation of MetricEntryCommander
type metricEntryCommander struct {
	store Store
}

// NewMetricEntryCommander creates a new MetricEntryService
func NewMetricEntryCommander(
	store Store,
) *metricEntryCommander {
	return &metricEntryCommander{
		store: store,
	}
}

func (s *metricEntryCommander) CreateWithExternalID(
	ctx context.Context,
	typeName string,
	agentID UUID,
	externalID string,
	resourceID string,
	value float64,
) (*MetricEntry, error) {
	svc, err := s.store.ServiceRepo().FindByExternalID(ctx, agentID, externalID)
	if err != nil {
		return nil, err
	}
	return s.Create(ctx, typeName, agentID, svc.ID, resourceID, value)
}

func (s *metricEntryCommander) Create(
	ctx context.Context,
	typeName string,
	agentID UUID,
	serviceID UUID,
	resourceID string,
	value float64,
) (*MetricEntry, error) {
	ok, err := s.store.AgentRepo().Exists(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, NewInvalidInputErrorf("invalid agent ID %s", agentID)
	}

	svc, err := s.store.ServiceRepo().FindByID(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	// Look up the metric type by name
	metricType, err := s.store.MetricTypeRepo().FindByName(ctx, typeName)
	if err != nil {
		return nil, err
	}

	// Validate metric type exists
	metricTypeExists, err := s.store.MetricTypeRepo().Exists(ctx, metricType.ID)
	if err != nil {
		return nil, err
	}
	if !metricTypeExists {
		return nil, InvalidInputError{Err: fmt.Errorf("metric type with ID %s does not exist", metricType.ID)}
	}

	metricEntry := &MetricEntry{
		BrokerID:   svc.BrokerID,
		ProviderID: svc.ProviderID,
		AgentID:    agentID,
		ServiceID:  svc.ID,
		ResourceID: resourceID,
		Value:      value,
		TypeID:     metricType.ID,
	}
	if err := metricEntry.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	if err := s.store.MetricEntryRepo().Create(ctx, metricEntry); err != nil {
		return nil, err
	}
	return metricEntry, nil
}

type MetricEntryRepository interface {
	MetricEntryQuerier

	// Create creates a new metric entry
	Create(ctx context.Context, entity *MetricEntry) error
}

type MetricEntryQuerier interface {
	// List retrieves a list of metric entries based on the provided filters
	List(ctx context.Context, authScope *AuthScope, req *PageRequest) (*PageResponse[MetricEntry], error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id UUID) (bool, error)

	// CountByMetricType counts the number of entries for a specific metric type
	CountByMetricType(ctx context.Context, typeID UUID) (int64, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthScope, error)
}
