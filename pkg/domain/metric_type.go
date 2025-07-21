package domain

import (
	"context"
	"errors"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeMetricTypeUpdated EventType = "metric_type.updated"
	EventTypeMetricTypeCreated EventType = "metric_type.created"
	EventTypeMetricTypeDeleted EventType = "metric_type.deleted"
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
func NewMetricType(params CreateMetricTypeParams) *MetricType {
	return &MetricType{
		Name:       params.Name,
		EntityType: params.EntityType,
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
	Create(ctx context.Context, params CreateMetricTypeParams) (*MetricType, error)

	// Update updates a metric-type
	Update(ctx context.Context, params UpdateMetricTypeParams) (*MetricType, error)

	// Delete removes a metric-type by ID after checking for dependencies
	Delete(ctx context.Context, id properties.UUID) error
}

type CreateMetricTypeParams struct {
	Name       string           `json:"name"`
	EntityType MetricEntityType `json:"entityType"`
}

type UpdateMetricTypeParams struct {
	ID   properties.UUID `json:"id"`
	Name *string         `json:"name"`
}

// metricTypeCommander is the concrete implementation of MetricTypeCommander
type metricTypeCommander struct {
	store           Store
	metricEntryRepo MetricEntryRepository
}

// NewMetricTypeCommander creates a new MetricTypeService
func NewMetricTypeCommander(
	store Store,
	metricEntryRepo MetricEntryRepository,
) *metricTypeCommander {
	return &metricTypeCommander{
		store:           store,
		metricEntryRepo: metricEntryRepo,
	}
}

// Create creates a new metric-type
func (s *metricTypeCommander) Create(
	ctx context.Context,
	params CreateMetricTypeParams,
) (*MetricType, error) {
	// Create and validate
	var metricType *MetricType
	err := s.store.Atomic(ctx, func(store Store) error {
		metricType = NewMetricType(params)
		if err := metricType.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.MetricTypeRepo().Create(ctx, metricType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeMetricTypeCreated, WithInitiatorCtx(ctx), WithMetricType(metricType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return err
	})

	if err != nil {
		return nil, err
	}
	return metricType, nil
}

// Update updates a metric-type
func (s *metricTypeCommander) Update(ctx context.Context,
	params UpdateMetricTypeParams,
) (*MetricType, error) {
	// Find it
	metricType, err := s.store.MetricTypeRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Store a copy of the metricType before modifications for event diff
	beforeMetricType := *metricType

	// Update and validate
	metricType.Update(params.Name)
	if err := metricType.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and event
	err = s.store.Atomic(ctx, func(store Store) error {
		err := store.MetricTypeRepo().Save(ctx, metricType)
		if err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeMetricTypeUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeMetricType, metricType), WithMetricType(metricType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return err
	})
	if err != nil {
		return nil, err
	}
	return metricType, nil
}

// Delete removes a metric-type by ID after checking for dependencies
func (s *metricTypeCommander) Delete(ctx context.Context, id properties.UUID) error {
	// Find it
	metricType, err := s.store.MetricTypeRepo().Get(ctx, id)
	if err != nil {
		return err
	}

	// check if the metric type is used in the metric store
	numOfEntries, err := s.metricEntryRepo.CountByMetricType(ctx, id)
	if err != nil {
		return err
	}
	if numOfEntries > 0 {
		return InvalidInputError{Err: errors.New("cannot delete metric-type with associated entries")}
	}

	// Check dependencies and delete
	return s.store.Atomic(ctx, func(store Store) error {
		if err := store.MetricTypeRepo().Delete(ctx, id); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeMetricTypeDeleted, WithInitiatorCtx(ctx), WithMetricType(metricType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return err
	})
}

type MetricTypeRepository interface {
	MetricTypeQuerier
	BaseEntityRepository[MetricType]
}

type MetricTypeQuerier interface {
	BaseEntityQuerier[MetricType]

	// FindByName retrieves a metric type by name
	FindByName(ctx context.Context, name string) (*MetricType, error)
}
