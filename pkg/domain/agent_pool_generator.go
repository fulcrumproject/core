// Agent pool generator interface and factory
package domain

import (
	"context"

	"github.com/fulcrumproject/core/pkg/properties"
)

// AgentPoolGenerator defines the interface for agent pool value allocation strategies
type AgentPoolGenerator interface {
	// Allocate allocates a value from the pool for the given agent and property
	Allocate(ctx context.Context, agentID properties.UUID, propertyName string) (any, error)

	// Release releases the given pre-fetched allocations that belong to this generator's pool.
	// Callers pass the agent's full allocation slice; implementations filter to their own pool.
	Release(ctx context.Context, values []*AgentPoolValue) error
}

// AgentPoolGeneratorFactory creates the appropriate generator for an agent pool
type AgentPoolGeneratorFactory interface {
	CreateGenerator(pool *AgentPool) (AgentPoolGenerator, error)
}

// DefaultAgentPoolGeneratorFactory is the default implementation of AgentPoolGeneratorFactory
type DefaultAgentPoolGeneratorFactory struct {
	valueRepo AgentPoolValueRepository
}

// NewDefaultAgentPoolGeneratorFactory creates a new DefaultAgentPoolGeneratorFactory
func NewDefaultAgentPoolGeneratorFactory(valueRepo AgentPoolValueRepository) *DefaultAgentPoolGeneratorFactory {
	return &DefaultAgentPoolGeneratorFactory{valueRepo: valueRepo}
}

// CreateGenerator dispatches on pool.GeneratorType so new types (e.g. subnet) can drop in
// without touching the schema engine or commander.
func (f *DefaultAgentPoolGeneratorFactory) CreateGenerator(pool *AgentPool) (AgentPoolGenerator, error) {
	switch pool.GeneratorType {
	case PoolGeneratorList:
		return NewAgentPoolListGenerator(f.valueRepo, pool.ID), nil
	default:
		return nil, NewInvalidInputErrorf("unsupported agent pool generator type: %s", pool.GeneratorType)
	}
}

var _ AgentPoolGeneratorFactory = (*DefaultAgentPoolGeneratorFactory)(nil)
