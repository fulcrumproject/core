// Service pool generator interface and implementations
package domain

import (
	"context"

	"github.com/fulcrumproject/core/pkg/properties"
)

// PoolGenerator defines the interface for pool value allocation strategies
type PoolGenerator interface {
	// Allocate allocates a value from the pool for the given service and property
	// Returns the actual value to be copied to the service property
	Allocate(ctx context.Context, poolID properties.UUID, serviceID properties.UUID, propertyName string) (any, error)

	// Release releases all allocations for the given service
	Release(ctx context.Context, serviceID properties.UUID) error
}

// PoolGeneratorFactory creates the appropriate generator for a pool
type PoolGeneratorFactory interface {
	// CreateGenerator creates a generator for the given pool
	CreateGenerator(pool *ServicePool) (PoolGenerator, error)
}

// DefaultGeneratorFactory is the default implementation of PoolGeneratorFactory
type DefaultGeneratorFactory struct {
	valueRepo ServicePoolValueRepository
}

// NewDefaultGeneratorFactory creates a new DefaultGeneratorFactory
func NewDefaultGeneratorFactory(valueRepo ServicePoolValueRepository) *DefaultGeneratorFactory {
	return &DefaultGeneratorFactory{
		valueRepo: valueRepo,
	}
}

// CreateGenerator creates the appropriate generator based on pool type
func (f *DefaultGeneratorFactory) CreateGenerator(pool *ServicePool) (PoolGenerator, error) {
	switch pool.GeneratorType {
	case PoolGeneratorList:
		return NewListGenerator(f.valueRepo, pool.ID), nil
	case PoolGeneratorSubnet:
		if pool.GeneratorConfig == nil {
			return nil, NewInvalidInputErrorf("subnet pool missing generator config")
		}
		return NewSubnetGenerator(f.valueRepo, pool.ID, *pool.GeneratorConfig), nil
	default:
		return nil, NewInvalidInputErrorf("unsupported pool generator type: %s", pool.GeneratorType)
	}
}
