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
type SchemaPoolGenerator struct {
	store Store
}

// Compile-time check that SchemaPoolGenerator implements schema.Generator
var _ schema.Generator[ServicePropertyContext] = (*SchemaPoolGenerator)(nil)

// NewSchemaPoolGenerator creates a new pool generator
func NewSchemaPoolGenerator(store Store) *SchemaPoolGenerator {
	return &SchemaPoolGenerator{
		store: store,
	}
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
	// Get pool type from config
	poolTypeRaw, hasPoolType := config["poolType"]
	if !hasPoolType {
		return nil, false, fmt.Errorf("%s: pool generator config missing 'poolType'", propPath)
	}

	poolType, ok := poolTypeRaw.(string)
	if !ok {
		return nil, false, fmt.Errorf("%s: pool generator config 'poolType' must be a string", propPath)
	}

	// Service must exist and have a pool set
	if schemaCtx.Service == nil {
		return nil, false, fmt.Errorf("%s: pool generator requires service context", propPath)
	}

	// Get the agent to access pool set ID
	if schemaCtx.Service.Agent == nil {
		// Lazy load agent if not already loaded
		agent, err := g.store.AgentRepo().Get(ctx, schemaCtx.Service.AgentID)
		if err != nil {
			return nil, false, fmt.Errorf("%s: failed to load service agent: %w", propPath, err)
		}
		schemaCtx.Service.Agent = agent
	}

	poolSetID := schemaCtx.Service.Agent.ServicePoolSetID
	if poolSetID == nil || *poolSetID == uuid.Nil {
		return nil, false, fmt.Errorf("%s: agent does not have a pool set configured", propPath)
	}

	// Find the pool with matching type in the pool set
	pools, err := g.store.ServicePoolRepo().ListByPoolSet(ctx, *poolSetID)
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
	factory := NewDefaultGeneratorFactory(g.store.ServicePoolValueRepo())
	generator, err := factory.CreateGenerator(targetPool)
	if err != nil {
		return nil, false, fmt.Errorf("%s: failed to create generator for pool: %w", propPath, err)
	}

	// Allocate value from pool
	allocatedValue, err := generator.Allocate(ctx, targetPool.ID, schemaCtx.Service.ID, propPath)
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
