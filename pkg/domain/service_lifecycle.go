// State transition logic for service lifecycle
package domain

import (
	"fmt"
	"regexp"
)

// ResolveNextState determines the next state for a service based on the current state,
// action, and optional error code using the lifecycle schema.
func ResolveNextState(lifecycle *LifecycleSchema, currentState string, action string, errorCode *string) (string, error) {
	if lifecycle == nil {
		return "", fmt.Errorf("lifecycle schema is nil")
	}

	// Find the action in the lifecycle schema
	var matchedAction *LifecycleAction
	for i := range lifecycle.Actions {
		if lifecycle.Actions[i].Name == action {
			matchedAction = &lifecycle.Actions[i]
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
func ValidateActionAllowed(lifecycle *LifecycleSchema, currentState string, action string) error {
	if lifecycle == nil {
		return fmt.Errorf("lifecycle schema is nil")
	}

	// Find the action in the lifecycle schema
	var matchedAction *LifecycleAction
	for i := range lifecycle.Actions {
		if lifecycle.Actions[i].Name == action {
			matchedAction = &lifecycle.Actions[i]
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
func IsTerminalState(lifecycle *LifecycleSchema, state string) bool {
	if lifecycle == nil {
		return false
	}

	for _, terminalState := range lifecycle.TerminalStates {
		if terminalState == state {
			return true
		}
	}
	return false
}

