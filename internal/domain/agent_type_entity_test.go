package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentType_TableName(t *testing.T) {
	agentType := AgentType{}
	assert.Equal(t, "agent_types", agentType.TableName())
}
