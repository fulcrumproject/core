// Pool generator for service property schema
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
)

// SchemaPoolGenerator implements schema.Generator for pool-based value allocation.
// It adapts the existing pool allocation infrastructure to the schema package's generator interface.
type SchemaPoolGenerator struct{}

// Compile-time check that SchemaPoolGenerator implements schema.Generator
var _ schema.Generator[ServicePropertyContext] = (*SchemaPoolGenerator)(nil)

// NewSchemaPoolGenerator creates a new pool generator
func NewSchemaPoolGenerator() *SchemaPoolGenerator {
	return &SchemaPoolGenerator{}
}

// Generate allocates a value from a service pool.
// The config must contain "poolType" which identifies which pool to allocate from.
// The service context must have a Service with a valid Agent.ServicePoolSetID.
func (g *SchemaPoolGenerator) Generate(
	ctx context.Context,
	schemaCtx ServicePropertyContext,
	propPath string,
	currentValue any,
	config map[string]any,
) (value any, generated bool, err error) {
	// Skip generation if value already exists (e.g., on update operations)
	if currentValue != nil {
		return currentValue, false, nil
	}

	// Get pool type from config
	poolTypeRaw, hasPoolType := config["poolType"]
	if !hasPoolType {
		return nil, false, fmt.Errorf("%s: pool generator config missing 'poolType'", propPath)
	}

	poolType, ok := poolTypeRaw.(string)
	if !ok {
		return nil, false, fmt.Errorf("%s: pool generator config 'poolType' must be a string", propPath)
	}

	// Pool set ID must be configured
	poolSetID := schemaCtx.ServicePoolSetID
	if poolSetID == nil || *poolSetID == uuid.Nil {
		return nil, false, fmt.Errorf("%s: agent does not have a pool set configured", propPath)
	}

	// Service ID must exist for pool allocation
	if schemaCtx.ServiceID == nil {
		return nil, false, fmt.Errorf("%s: service ID required for pool allocation", propPath)
	}

	// Find the pool with matching type in the pool set
	pools, err := schemaCtx.Store.ServicePoolRepo().ListByPoolSet(ctx, *poolSetID)
	if err != nil {
		return nil, false, fmt.Errorf("%s: failed to list pools for pool set: %w", propPath, err)
	}

	var targetPool *ServicePool
	for _, pool := range pools {
		if pool.Type == poolType {
			targetPool = pool
			break
		}
	}

	if targetPool == nil {
		return nil, false, fmt.Errorf("%s: no pool found with type '%s' in pool set", propPath, poolType)
	}

	// Create generator and allocate
	factory := NewDefaultGeneratorFactory(schemaCtx.Store.ServicePoolValueRepo())
	generator, err := factory.CreateGenerator(targetPool)
	if err != nil {
		return nil, false, fmt.Errorf("%s: failed to create generator for pool: %w", propPath, err)
	}

	// Allocate value from pool
	allocatedValue, err := generator.Allocate(ctx, targetPool.ID, *schemaCtx.ServiceID, propPath)
	if err != nil {
		return nil, false, fmt.Errorf("%s: failed to allocate from pool: %w", propPath, err)
	}

	return allocatedValue, true, nil
}

// ValidateConfig validates the pool generator configuration
func (g *SchemaPoolGenerator) ValidateConfig(propPath string, config map[string]any) error {
	if len(config) == 0 {
		return fmt.Errorf("pool generator config missing 'poolType'")
	}

	poolTypeRaw, hasPoolType := config["poolType"]
	if !hasPoolType {
		return fmt.Errorf("pool generator config missing 'poolType'")
	}

	poolType, ok := poolTypeRaw.(string)
	if !ok {
		return fmt.Errorf("pool generator config 'poolType' must be a string")
	}

	if poolType == "" {
		return fmt.Errorf("pool generator config 'poolType' cannot be empty")
	}

	return nil
}
