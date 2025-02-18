package domain

import (
	"errors"
)

var (
	// ErrInvalidAgentState indicates that the agent state is not valid
	ErrInvalidAgentState = errors.New("invalid agent state")
)

// AgentState represents the possible states of an Agent
type AgentState string

const (
	// AgentNew represents a newly created agent
	AgentNew AgentState = "New"
	// AgentConnected represents a connected agent
	AgentConnected AgentState = "Connected"
	// AgentDisconnected represents a disconnected agent
	AgentDisconnected AgentState = "Disconnected"
	// AgentError represents an agent in error state
	AgentError AgentState = "Error"
	// AgentDisabled represents a disabled agent
	AgentDisabled AgentState = "Disabled"
)

// IsValid checks if the agent state is valid
func (s AgentState) IsValid() bool {
	switch s {
	case AgentNew, AgentConnected, AgentDisconnected, AgentError, AgentDisabled:
		return true
	default:
		return false
	}
}

func ParseAgentState(value string) (AgentState, error) {
	state := AgentState(value)
	if !state.IsValid() {
		return state, ErrInvalidAgentState
	}
	return state, nil
}
