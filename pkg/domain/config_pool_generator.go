// Agent pool generator interface and factory
package domain

import (
	"context"

	"github.com/fulcrumproject/core/pkg/properties"
)

// ConfigPoolGenerator defines the interface for config pool value allocation strategies
type ConfigPoolGenerator interface {
	// Allocate allocates a value from the pool to the given entity (agent or
	// infrastructure) for the given property.
	Allocate(ctx context.Context, entityType ConfigPoolValueEntityType, entityID properties.UUID, propertyName string) (any, error)

	// Release releases the given pre-fetched allocations that belong to this generator's pool.
	// Callers pass the agent's full allocation slice; implementations filter to their own pool.
	Release(ctx context.Context, values []*ConfigPoolValue) error
}

// ConfigPoolGeneratorFactory creates the appropriate generator for an config pool
type ConfigPoolGeneratorFactory interface {
	CreateGenerator(pool *ConfigPool) (ConfigPoolGenerator, error)
}

// DefaultConfigPoolGeneratorFactory is the default implementation of ConfigPoolGeneratorFactory
type DefaultConfigPoolGeneratorFactory struct {
	valueRepo ConfigPoolValueRepository
}

// NewDefaultConfigPoolGeneratorFactory creates a new DefaultConfigPoolGeneratorFactory
func NewDefaultConfigPoolGeneratorFactory(valueRepo ConfigPoolValueRepository) *DefaultConfigPoolGeneratorFactory {
	return &DefaultConfigPoolGeneratorFactory{valueRepo: valueRepo}
}

// CreateGenerator dispatches on pool.GeneratorType so new types (e.g. subnet) can drop in
// without touching the schema engine or commander.
func (f *DefaultConfigPoolGeneratorFactory) CreateGenerator(pool *ConfigPool) (ConfigPoolGenerator, error) {
	switch pool.GeneratorType {
	case PoolGeneratorList:
		return NewConfigPoolListGenerator(f.valueRepo, pool.ID), nil
	default:
		return nil, NewInvalidInputErrorf("unsupported config pool generator type: %s", pool.GeneratorType)
	}
}

var _ ConfigPoolGeneratorFactory = (*DefaultConfigPoolGeneratorFactory)(nil)
