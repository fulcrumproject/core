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
	Name       string           `gorm:"not null;unique"`
	EntityType MetricEntityType `gorm:"not null"`
}

// TableName returns the table name for the metric type
func (*MetricType) TableName() string {
	return "metric_types"
}

// Validate ensures all MetricType fields are valid
func (m *MetricType) Validate() error {
	if err := m.EntityType.Validate(); err != nil {
		return err
	}
	return nil
}

// MetricTypeCommander handles metric-type operations with validation
type MetricTypeCommander struct {
	repo      MetricTypeRepository
	entryRepo MetricEntryRepository
}

// NewMetricTypeCommander creates a new MetricTypeService
func NewMetricTypeCommander(
	repo MetricTypeRepository,
	entryRepo MetricEntryRepository,
) *MetricTypeCommander {
	return &MetricTypeCommander{
		repo:      repo,
		entryRepo: entryRepo,
	}
}

// Create creates a new metric-type with validation
func (s *MetricTypeCommander) Create(
	ctx context.Context,
	name string,
	kind MetricEntityType,
) (*MetricType, error) {
	metricType := &MetricType{
		Name:       name,
		EntityType: kind,
	}
	if err := metricType.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, metricType); err != nil {
		return nil, err
	}
	return metricType, nil
}

// Update updates a metric-type with validation
func (s *MetricTypeCommander) Update(ctx context.Context,
	id UUID,
	name *string,
) (*MetricType, error) {
	metricType, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		metricType.Name = *name
	}
	if err := metricType.Validate(); err != nil {
		return nil, err
	}
	err = s.repo.Save(ctx, metricType)
	if err != nil {
		return nil, err
	}
	return metricType, nil
}

// Delete removes a metric-type by ID after checking for dependencies
func (s *MetricTypeCommander) Delete(ctx context.Context, id UUID) error {
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	// Check if metric-type has entries
	numOfAgents, err := s.entryRepo.CountByMetricType(ctx, id)
	if err != nil {
		return err
	}
	if numOfAgents > 0 {
		return errors.New("cannot delete metric-type with associated agents")
	}
	return s.repo.Delete(ctx, id)
}

type MetricTypeRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *MetricType) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *MetricType) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*MetricType, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[MetricType], error)

	// FindByName retrieves a metric type by name
	FindByName(ctx context.Context, name string) (*MetricType, error)

	// Count returns the number of entities
	Count(ctx context.Context) (int64, error)
}

type MetricTypeQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*MetricType, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[MetricType], error)

	// FindByName retrieves a metric type by name
	FindByName(ctx context.Context, name string) (*MetricType, error)

	// Count returns the number of entities
	Count(ctx context.Context) (int64, error)
}
