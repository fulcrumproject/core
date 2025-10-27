// Domain-specific authorizers for service property schema
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/schema"
)

// ActorAuthorizer validates that the actor is authorized to set/update a property.
// Config: {"actors": ["user", "agent", "system"]}
// Logic: Current actor must be in the actors array (OR logic within actors)
type ActorAuthorizer struct{}

// Authorize checks if the actor is in the allowed list
func (a *ActorAuthorizer) Authorize(
	ctx context.Context,
	schemaCtx ServicePropertyContext,
	operation schema.Operation,
	propPath string,
	hasNewValue bool,
	config map[string]any,
) error {
	// Only check authorization if a value is being set
	if !hasNewValue {
		return nil
	}

	// Get allowed actors from config
	allowedActorsRaw, hasConfig := config["actors"]
	if !hasConfig {
		return fmt.Errorf("%s: actor authorizer config missing 'actors'", propPath)
	}

	allowedActors, ok := allowedActorsRaw.([]any)
	if !ok {
		return fmt.Errorf("%s: actor authorizer config 'actors' must be an array", propPath)
	}

	// Check if current actor is in allowed list
	currentActor := string(schemaCtx.Actor)
	for _, actorRaw := range allowedActors {
		if actor, ok := actorRaw.(string); ok && actor == currentActor {
			return nil // Authorized
		}
	}

	// Build list of allowed actors for error message
	allowedNames := make([]string, 0, len(allowedActors))
	for _, actorRaw := range allowedActors {
		if actor, ok := actorRaw.(string); ok {
			allowedNames = append(allowedNames, actor)
		}
	}

	return fmt.Errorf("%s: property can only be set by: %v (current actor: %s)", propPath, allowedNames, currentActor)
}

// ValidateConfig validates the actor authorizer configuration
func (a *ActorAuthorizer) ValidateConfig(propPath string, config map[string]any) error {
	allowedActorsRaw, hasConfig := config["actors"]
	if !hasConfig {
		return fmt.Errorf("actor authorizer config missing 'actors'")
	}

	allowedActors, ok := allowedActorsRaw.([]any)
	if !ok {
		return fmt.Errorf("actor authorizer config 'actors' must be an array")
	}

	if len(allowedActors) == 0 {
		return fmt.Errorf("actor authorizer config 'actors' must not be empty")
	}

	// Validate each actor
	validActors := map[string]bool{"user": true, "agent": true, "system": true}
	for _, actorRaw := range allowedActors {
		actor, ok := actorRaw.(string)
		if !ok {
			return fmt.Errorf("actor authorizer config 'actors' must contain only strings")
		}
		if !validActors[actor] {
			return fmt.Errorf("actor authorizer config 'actors' contains invalid actor '%s' (must be: user, agent, system)", actor)
		}
	}

	return nil
}

// StateAuthorizer validates that a property can be updated in the service's current state.
// Config: {"allowedStates": ["New", "Stopped", "Started"]}
// Logic: Current service status must be in allowedStates array (OR logic within states)
// Only applies to update operations.
type StateAuthorizer struct{}

// Authorize checks if the property can be updated in the service's current state
func (a *StateAuthorizer) Authorize(
	ctx context.Context,
	schemaCtx ServicePropertyContext,
	operation schema.Operation,
	propPath string,
	hasNewValue bool,
	config map[string]any,
) error {
	// Only apply to update operations
	if operation != schema.OperationUpdate {
		return nil
	}

	// Only check if a value is being set
	if !hasNewValue {
		return nil
	}

	// Service status must exist for update operations
	if schemaCtx.ServiceStatus == "" {
		return fmt.Errorf("%s: state authorizer requires service status for update operations", propPath)
	}

	// Get allowed states
	allowedStatesRaw, hasConfig := config["allowedStates"]
	if !hasConfig {
		return fmt.Errorf("%s: state authorizer config missing 'allowedStates'", propPath)
	}

	allowedStates, ok := allowedStatesRaw.([]any)
	if !ok {
		return fmt.Errorf("%s: state authorizer config 'allowedStates' must be an array", propPath)
	}

	// Check if current status is in the allowed states
	currentStatus := schemaCtx.ServiceStatus
	for _, stateRaw := range allowedStates {
		if state, ok := stateRaw.(string); ok && state == currentStatus {
			return nil // Current state allows updates
		}
	}

	// Build list of allowed states for error message
	allowedNames := make([]string, 0, len(allowedStates))
	for _, stateRaw := range allowedStates {
		if state, ok := stateRaw.(string); ok {
			allowedNames = append(allowedNames, state)
		}
	}

	return fmt.Errorf("%s: property cannot be updated in state '%s' (allowed states: %v)", propPath, currentStatus, allowedNames)
}

// ValidateConfig validates the state authorizer configuration
func (a *StateAuthorizer) ValidateConfig(propPath string, config map[string]any) error {
	allowedStatesRaw, hasConfig := config["allowedStates"]
	if !hasConfig {
		return fmt.Errorf("state authorizer config missing 'allowedStates'")
	}

	allowedStates, ok := allowedStatesRaw.([]any)
	if !ok {
		return fmt.Errorf("state authorizer config 'allowedStates' must be an array")
	}

	if len(allowedStates) == 0 {
		return fmt.Errorf("state authorizer config 'allowedStates' must not be empty")
	}

	// Validate each state is a string
	for _, stateRaw := range allowedStates {
		if _, ok := stateRaw.(string); !ok {
			return fmt.Errorf("state authorizer config 'allowedStates' must contain only strings")
		}
	}

	return nil
}
