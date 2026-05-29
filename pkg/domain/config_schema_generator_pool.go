// Pool generator shared by the agent and infrastructure configuration schemas.
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
)

// poolAllocContext is the slice of a config schema context the pool generator needs:
// the transactional Store, the consuming entity's type+id (stamped onto the allocated
// value), and the provider scoping pool lookup. Implemented by AgentConfigContext and
// InfrastructureConfigContext.
type poolAllocContext interface {
	poolStore() Store
	poolEntityType() ConfigPoolValueEntityType
	poolEntityID() *properties.UUID
	poolProviderID() properties.UUID
}

// SchemaConfigPoolGenerator adapts ConfigPool allocation into the schema package's generator interface.
// It dispatches concrete allocation through an ConfigPoolGeneratorFactory so new generator types
// (e.g. subnet) can be added without touching the schema engine wiring or the commander.
type SchemaConfigPoolGenerator[C poolAllocContext] struct{}

// NewSchemaConfigPoolGenerator creates a generator that constructs its factory from the
// transactional Store carried in the config context at Generate time.
func NewSchemaConfigPoolGenerator[C poolAllocContext]() *SchemaConfigPoolGenerator[C] {
	return &SchemaConfigPoolGenerator[C]{}
}

// Generate resolves the ConfigPool by Type and allocates an available ConfigPoolValue.
// Skips generation if a value is already present (idempotent on update).
func (g *SchemaConfigPoolGenerator[C]) Generate(
	ctx context.Context,
	schemaCtx C,
	propPath string,
	currentValue any,
	config map[string]any,
) (value any, generated bool, err error) {
	if currentValue != nil {
		return currentValue, false, nil
	}

	poolType, err := parsePoolTypeConfig(config)
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", propPath, err)
	}

	store := schemaCtx.poolStore()
	if store == nil {
		return nil, false, fmt.Errorf("%s: config context missing store", propPath)
	}
	entityID := schemaCtx.poolEntityID()
	if entityID == nil {
		return nil, false, fmt.Errorf("%s: entity ID required for pool allocation", propPath)
	}

	providerID := schemaCtx.poolProviderID()
	pool, err := store.ConfigPoolRepo().FindByTypeAndProvider(ctx, poolType, &providerID)
	if err != nil {
		return nil, false, fmt.Errorf("%s: failed to find config pool with type %q: %w", propPath, poolType, err)
	}

	factory := NewDefaultConfigPoolGeneratorFactory(store.ConfigPoolValueRepo())
	generator, err := factory.CreateGenerator(pool)
	if err != nil {
		return nil, false, fmt.Errorf("%s: failed to create generator for pool: %w", propPath, err)
	}

	allocatedValue, err := generator.Allocate(ctx, schemaCtx.poolEntityType(), *entityID, propPath)
	if err != nil {
		return nil, false, fmt.Errorf("%s: failed to allocate from pool: %w", propPath, err)
	}

	return allocatedValue, true, nil
}

// ValidateConfig checks that poolType is present, a non-empty string.
// Called by the schema engine at schema-validation time (see pkg/schema/engine.go) so
// malformed configs are rejected when the type is saved, before any entity exists.
func (g *SchemaConfigPoolGenerator[C]) ValidateConfig(propPath string, config map[string]any) error {
	_, err := parsePoolTypeConfig(config)
	return err
}

// parsePoolTypeConfig reads the "poolType" field from a generator config map and validates it.
// Shared by Generate (which wraps with propPath) and ValidateConfig.
func parsePoolTypeConfig(config map[string]any) (string, error) {
	poolTypeRaw, hasPoolType := config["poolType"]
	if !hasPoolType {
		return "", fmt.Errorf("pool generator config missing 'poolType'")
	}
	poolType, ok := poolTypeRaw.(string)
	if !ok {
		return "", fmt.Errorf("pool generator config 'poolType' must be a string")
	}
	if poolType == "" {
		return "", fmt.Errorf("pool generator config 'poolType' cannot be empty")
	}
	return poolType, nil
}

var (
	_ schema.Generator[AgentConfigContext]          = (*SchemaConfigPoolGenerator[AgentConfigContext])(nil)
	_ schema.Generator[InfrastructureConfigContext] = (*SchemaConfigPoolGenerator[InfrastructureConfigContext])(nil)
)
