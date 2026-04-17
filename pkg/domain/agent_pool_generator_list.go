// List-based agent pool generator implementation
package domain

import "github.com/fulcrumproject/core/pkg/properties"

type AgentPoolListGenerator = PoolListGenerator[*AgentPoolValue]

func NewAgentPoolListGenerator(valueRepo AgentPoolValueRepository, poolID properties.UUID) *AgentPoolListGenerator {
	return NewPoolListGenerator(valueRepo, poolID)
}

var _ AgentPoolGenerator = (*AgentPoolListGenerator)(nil)
