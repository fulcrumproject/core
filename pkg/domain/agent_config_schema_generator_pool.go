// Pool generator for agent configuration schema
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/schema"
)

// SchemaAgentPoolGenerator adapts AgentPool allocation into the schema package's generator interface.
// It dispatches concrete allocation through an AgentPoolGeneratorFactory so new generator types
// (e.g. subnet) can be added without touching the schema engine wiring or the commander.
type SchemaAgentPoolGenerator struct{}

// NewSchemaAgentPoolGenerator creates a generator that constructs its factory from the
// transactional Store carried in AgentConfigContext at Generate time.
func NewSchemaAgentPoolGenerator() *SchemaAgentPoolGenerator {
	return &SchemaAgentPoolGenerator{}
}

// Generate resolves the AgentPool by Type and allocates an available AgentPoolValue.
// Skips generation if a value is already present (idempotent on update).
func (g *SchemaAgentPoolGenerator) Generate(
	ctx context.Context,
	schemaCtx AgentConfigContext,
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

	if schemaCtx.Store == nil {
		return nil, false, fmt.Errorf("%s: agent config context missing store", propPath)
	}
	if schemaCtx.AgentID == nil {
		return nil, false, fmt.Errorf("%s: agent ID required for pool allocation", propPath)
	}

	pool, err := schemaCtx.Store.AgentPoolRepo().FindByType(ctx, poolType)
	if err != nil {
		return nil, false, fmt.Errorf("%s: failed to find agent pool with type %q: %w", propPath, poolType, err)
	}

	factory := NewDefaultAgentPoolGeneratorFactory(schemaCtx.Store.AgentPoolValueRepo())
	generator, err := factory.CreateGenerator(pool)
	if err != nil {
		return nil, false, fmt.Errorf("%s: failed to create generator for pool: %w", propPath, err)
	}

	allocatedValue, err := generator.Allocate(ctx, *schemaCtx.AgentID, propPath)
	if err != nil {
		return nil, false, fmt.Errorf("%s: failed to allocate from pool: %w", propPath, err)
	}

	return allocatedValue, true, nil
}

// ValidateConfig checks that poolType is present, a non-empty string.
// Called by the schema engine at schema-validation time (see pkg/schema/engine.go) so
// malformed configs are rejected when the AgentType is saved, before any agent exists.
func (g *SchemaAgentPoolGenerator) ValidateConfig(propPath string, config map[string]any) error {
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

var _ schema.Generator[AgentConfigContext] = (*SchemaAgentPoolGenerator)(nil)
