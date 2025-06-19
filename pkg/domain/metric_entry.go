package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/commons/auth"
	"github.com/fulcrumproject/commons/properties"
	"github.com/google/uuid"
)

// MetricEntry represents a metric measurement for a specific resource
type MetricEntry struct {
	BaseEntity

	ResourceID string  `gorm:"not null"`
	Value      float64 `gorm:"not null"`

	// Relationships
	TypeID     properties.UUID `gorm:"not null"`
	Type       *MetricType     `gorm:"foreignKey:TypeID"`
	AgentID    properties.UUID `gorm:"not null"`
	Agent      *Agent          `gorm:"foreignKey:AgentID"`
	ServiceID  properties.UUID `gorm:"not null"`
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
	Create(ctx context.Context, typeName string, agentID properties.UUID, serviceID properties.UUID, resourceID string, value float64) (*MetricEntry, error)

	// CreateWithExternalID creates a new metric entry using service's external ID
	CreateWithExternalID(ctx context.Context, typeName string, agentID properties.UUID, externalID string, resourceID string, value float64) (*MetricEntry, error)
}

// metricEntryCommander is the concrete implementation of MetricEntryCommander
type metricEntryCommander struct {
	store Store
}

// NewMetricEntryCommander creates a new MetricEntryCommander
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
	agentID properties.UUID,
	externalID string,
	resourceID string,
	value float64,
) (*MetricEntry, error) {
	// 1. Validate agent exists
	ok, err := s.store.AgentRepo().Exists(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, NewInvalidInputErrorf("invalid agent ID %s", agentID)
	}

	// 2. Find service by external ID
	svc, err := s.store.ServiceRepo().FindByExternalID(ctx, agentID, externalID)
	if err != nil {
		return nil, err
	}

	// 3. Validate type compatibility
	metricType, err := s.store.MetricTypeRepo().FindByName(ctx, typeName)
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
		agentID,
		svc.ID,
		resourceID,
		metricType.ID,
		value,
	)

	if err := metricEntry.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// 6. Save
	if err := s.store.MetricEntryRepo().Create(ctx, metricEntry); err != nil {
		return nil, err
	}

	return metricEntry, nil
}

func (s *metricEntryCommander) Create(
	ctx context.Context,
	typeName string,
	agentID properties.UUID,
	serviceID properties.UUID,
	resourceID string,
	value float64,
) (*MetricEntry, error) {
	// 1. Validate agent exists
	ok, err := s.store.AgentRepo().Exists(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, NewInvalidInputErrorf("invalid agent ID %s", agentID)
	}

	// 2. Find service
	svc, err := s.store.ServiceRepo().Get(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	// 3. Validate type compatibility
	metricType, err := s.store.MetricTypeRepo().FindByName(ctx, typeName)
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
		agentID,
		svc.ID,
		resourceID,
		metricType.ID,
		value,
	)

	if err := metricEntry.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// 6. Save
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
	List(ctx context.Context, authIdentityScope *auth.IdentityScope, req *PageRequest) (*PageResponse[MetricEntry], error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id properties.UUID) (bool, error)

	// CountByMetricType counts the number of entries for a specific metric type
	CountByMetricType(ctx context.Context, typeID properties.UUID) (int64, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error)
}
