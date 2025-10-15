// State transition logic for service lifecycle
package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
)

// LifecycleSchema defines the state machine for a service type
type LifecycleSchema struct {
	States         []LifecycleState  `json:"states"`
	Actions        []LifecycleAction `json:"actions"`
	InitialState   string            `json:"initialState"`
	TerminalStates []string          `json:"terminalStates"`
	RunningStates  []string          `json:"runningStates,omitempty"`
}

// Scan implements the sql.Scanner interface
func (ls *LifecycleSchema) Scan(value any) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal LifecycleSchema value: %v", value)
	}

	return json.Unmarshal(bytes, ls)
}

// Value implements the driver.Valuer interface
func (ls LifecycleSchema) Value() (driver.Value, error) {
	return json.Marshal(ls)
}

// LifecycleState represents a state in the service lifecycle
type LifecycleState struct {
	Name string `json:"name"`
}

// LifecycleAction represents an action that can be performed on a service
type LifecycleAction struct {
	Name              string                `json:"name"`
	RequestSchemaType string                `json:"requestSchemaType,omitempty"`
	Transitions       []LifecycleTransition `json:"transitions"`
}

// LifecycleTransition represents a state transition triggered by an action
type LifecycleTransition struct {
	From          string `json:"from"`
	To            string `json:"to"`
	OnError       bool   `json:"onError,omitempty"`
	OnErrorRegexp string `json:"onErrorRegexp,omitempty"`
}

// ResolveNextState determines the next state for a service based on the current state,
// action, and optional error code using the lifecycle schema.
func (ls *LifecycleSchema) ResolveNextState(currentState string, action string, errorCode *string) (string, error) {
	// Find the action in the lifecycle schema
	var matchedAction *LifecycleAction
	for i := range ls.Actions {
		if ls.Actions[i].Name == action {
			matchedAction = &ls.Actions[i]
			break
		}
	}

	if matchedAction == nil {
		return "", fmt.Errorf("action %q not found in lifecycle schema", action)
	}

	// If no error code, find first transition without OnError flag that matches current state
	if errorCode == nil {
		for _, transition := range matchedAction.Transitions {
			if transition.From == currentState && !transition.OnError {
				return transition.To, nil
			}
		}
		return "", fmt.Errorf("no valid transition found for action %q from state %q", action, currentState)
	}

	// Error code provided - find matching error transition
	for _, transition := range matchedAction.Transitions {
		if transition.From != currentState {
			continue
		}
		if !transition.OnError {
			continue
		}

		// If no regexp specified, matches any error
		if transition.OnErrorRegexp == "" {
			return transition.To, nil
		}

		// Check if error code matches the regexp
		matched, err := regexp.MatchString(transition.OnErrorRegexp, *errorCode)
		if err != nil {
			return "", fmt.Errorf("failed to match error regexp %q: %w", transition.OnErrorRegexp, err)
		}
		if matched {
			return transition.To, nil
		}
	}

	return "", fmt.Errorf("no valid error transition found for action %q from state %q with error code %q", action, currentState, *errorCode)
}

// ValidateActionAllowed checks if an action is allowed from the current state
func (ls *LifecycleSchema) ValidateActionAllowed(currentState string, action string) error {
	// Find the action in the lifecycle schema
	var matchedAction *LifecycleAction
	for i := range ls.Actions {
		if ls.Actions[i].Name == action {
			matchedAction = &ls.Actions[i]
			break
		}
	}

	if matchedAction == nil {
		return fmt.Errorf("action %q not found in lifecycle schema", action)
	}

	// Check if any transition exists from current state (either success or error)
	for _, transition := range matchedAction.Transitions {
		if transition.From == currentState {
			return nil // Action is allowed
		}
	}

	return fmt.Errorf("action %q is not allowed from state %q", action, currentState)
}

// IsTerminalState checks if a state is a terminal state in the lifecycle
func (ls *LifecycleSchema) IsTerminalState(state string) bool {
	return slices.Contains(ls.TerminalStates, state)
}

// IsRunningStatus checks if a given status is considered a "running" state for uptime calculation
func (ls *LifecycleSchema) IsRunningStatus(status string) bool {
	return slices.Contains(ls.RunningStates, status)
}
