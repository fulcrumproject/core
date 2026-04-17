// List-based agent pool generator implementation
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

// AgentPoolListGenerator allocates values from a pre-configured list of AgentPoolValue rows.
type AgentPoolListGenerator struct {
	valueRepo AgentPoolValueRepository
	poolID    properties.UUID
}

// NewAgentPoolListGenerator creates a new list-based generator for the given pool.
func NewAgentPoolListGenerator(valueRepo AgentPoolValueRepository, poolID properties.UUID) *AgentPoolListGenerator {
	return &AgentPoolListGenerator{valueRepo: valueRepo, poolID: poolID}
}

// Allocate takes the first available value, stamps it with agentID/propertyName/AllocatedAt, and persists.
func (g *AgentPoolListGenerator) Allocate(ctx context.Context, poolID properties.UUID, agentID properties.UUID, propertyName string) (any, error) {
	availableValues, err := g.valueRepo.FindAvailable(ctx, g.poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query available values: %w", err)
	}

	if len(availableValues) == 0 {
		return nil, NewInvalidInputErrorf("no available values in pool")
	}

	value := availableValues[0]
	value.Allocate(agentID, propertyName)

	if err := g.valueRepo.Update(ctx, value); err != nil {
		return nil, fmt.Errorf("failed to allocate value: %w", err)
	}

	return value.Value, nil
}

// Release iterates the pre-fetched values, skips any not belonging to this pool, and nulls the allocation fields.
func (g *AgentPoolListGenerator) Release(ctx context.Context, values []*AgentPoolValue) error {
	for _, value := range values {
		if value.AgentPoolID != g.poolID {
			continue
		}
		value.Release()
		if err := g.valueRepo.Update(ctx, value); err != nil {
			return fmt.Errorf("failed to release value: %w", err)
		}
	}

	return nil
}

var _ AgentPoolGenerator = (*AgentPoolListGenerator)(nil)
