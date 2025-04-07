package domain

import (
	"context"
	"errors"
	"fmt"
)

// MetricEntityType represents the possible types of entities that can be measured
type MetricEntityType string

const (
	MetricEntityTypeAgent    MetricEntityType = "Agent"
	MetricEntityTypeService  MetricEntityType = "Service"
	MetricEntityTypeResource MetricEntityType = "Resource"
)

// Validate ensures the MetricEntityType is one of the allowed values
func (t MetricEntityType) Validate() error {
	switch t {
	case MetricEntityTypeAgent,
		MetricEntityTypeService,
		MetricEntityTypeResource:
		return nil
	default:
		return fmt.Errorf("invalid %v metric entity type", t)
	}
}

// MetricType represents a type of metric that can be collected
type MetricType struct {
	BaseEntity
	Name       string           `json:"name" gorm:"not null;unique"`
	EntityType MetricEntityType `json:"entityType" gorm:"not null"`
}

// NewMetricType creates a new metric type without validation
func NewMetricType(name string, entityType MetricEntityType) *MetricType {
	return &MetricType{
		Name:       name,
		EntityType: entityType,
	}
}

// TableName returns the table name for the metric type
func (MetricType) TableName() string {
	return "metric_types"
}

// Validate ensures all MetricType fields are valid
func (m *MetricType) Validate() error {
	if err := m.EntityType.Validate(); err != nil {
		return fmt.Errorf("invalid entity type: %w", err)
	}
	if m.Name == "" {
		return fmt.Errorf("metric type name cannot be empty")
	}
	return nil
}

// Update updates the metric type
func (m *MetricType) Update(name *string) {
	if name != nil {
		m.Name = *name
	}
}

// MetricTypeCommander defines the interface for metric type command operations
type MetricTypeCommander interface {
	// Create creates a new metric-type
	Create(ctx context.Context, name string, kind MetricEntityType) (*MetricType, error)

	// Update updates a metric-type
	Update(ctx context.Context, id UUID, name *string) (*MetricType, error)

	// Delete removes a metric-type by ID after checking for dependencies
	Delete(ctx context.Context, id UUID) error
}

// metricTypeCommander is the concrete implementation of MetricTypeCommander
type metricTypeCommander struct {
	store          Store
	auditCommander AuditEntryCommander
}

// NewMetricTypeCommander creates a new MetricTypeService
func NewMetricTypeCommander(
	store Store,
	auditCommander AuditEntryCommander,
) *metricTypeCommander {
	return &metricTypeCommander{
		store:          store,
		auditCommander: auditCommander,
	}
}

// Create creates a new metric-type
func (s *metricTypeCommander) Create(
	ctx context.Context,
	name string,
	kind MetricEntityType,
) (*MetricType, error) {
	// Create and validate
	var metricType *MetricType
	err := s.store.Atomic(ctx, func(store Store) error {
		metricType = NewMetricType(name, kind)
		if err := metricType.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.MetricTypeRepo().Create(ctx, metricType); err != nil {
			return err
		}

		_, err := s.auditCommander.CreateCtx(
			ctx,
			EventTypeMetricTypeCreated,
			JSON{"state": metricType},
			&metricType.ID, nil, nil, nil)
		return err
	})

	if err != nil {
		return nil, err
	}
	return metricType, nil
}

// Update updates a metric-type
func (s *metricTypeCommander) Update(ctx context.Context,
	id UUID,
	name *string,
) (*MetricType, error) {
	// Find it
	metricType, err := s.store.MetricTypeRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store a copy of the metricType before modifications for audit diff
	beforeMetricType := *metricType

	// Update and validate
	metricType.Update(name)
	if err := metricType.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and audit
	err = s.store.Atomic(ctx, func(store Store) error {
		err := store.MetricTypeRepo().Save(ctx, metricType)
		if err != nil {
			return err
		}

		_, err = s.auditCommander.CreateCtxWithDiff(
			ctx,
			EventTypeMetricTypeUpdated,
			&id, nil, nil, nil,
			&beforeMetricType, metricType)
		return err
	})
	if err != nil {
		return nil, err
	}
	return metricType, nil
}

// Delete removes a metric-type by ID after checking for dependencies
func (s *metricTypeCommander) Delete(ctx context.Context, id UUID) error {
	// Find it
	metricType, err := s.store.MetricTypeRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}
	// Check dependencies and delete
	return s.store.Atomic(ctx, func(store Store) error {
		numOfEntries, err := store.MetricEntryRepo().CountByMetricType(ctx, id)
		if err != nil {
			return err
		}
		if numOfEntries > 0 {
			return InvalidInputError{Err: errors.New("cannot delete metric-type with associated entries")}
		}

		if err := store.MetricTypeRepo().Delete(ctx, id); err != nil {
			return err
		}

		_, err = s.auditCommander.CreateCtx(
			ctx, EventTypeMetricTypeDeleted,
			JSON{"state": metricType}, &id, nil, nil, nil)
		return err
	})
}

type MetricTypeRepository interface {
	MetricTypeQuerier

	// Create creates a new entity
	Create(ctx context.Context, entity *MetricType) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *MetricType) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error
}

type MetricTypeQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*MetricType, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id UUID) (bool, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authScope *AuthScope, req *PageRequest) (*PageResponse[MetricType], error)

	// FindByName retrieves a metric type by name
	FindByName(ctx context.Context, name string) (*MetricType, error)

	// Count returns the number of entities
	Count(ctx context.Context) (int64, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthScope, error)
}
