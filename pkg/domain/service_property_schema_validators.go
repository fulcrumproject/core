// Domain-specific validators for service property schema
package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/schema"
)

// SourceValidator validates that the actor is authorized to set/update a property
// based on the property's source configuration.
type SourceValidator struct{}

// Validate checks if the actor can set/update the property based on its source.
// Rules:
// - Properties with generators are system-generated and cannot be manually set
// - Properties with source="agent" can only be set by agents
// - All other properties can be set by users (default)
func (v *SourceValidator) Validate(
	ctx context.Context,
	schemaCtx ServicePropertyContext,
	operation schema.Operation,
	propPath string,
	oldValue, newValue any,
	config map[string]any,
) error {
	// Only validate if a value is being set
	if newValue == nil {
		return nil
	}

	// Check for explicit source configuration
	sourceRaw, hasSource := config["source"]
	if !hasSource {
		// No explicit source means user input (default)
		return nil
	}

	source, ok := sourceRaw.(string)
	if !ok {
		return fmt.Errorf("%s: source validator config 'source' must be a string", propPath)
	}

	// Validate based on source
	switch source {
	case "agent":
		if schemaCtx.Actor != ActorAgent {
			return fmt.Errorf("%s: property can only be set by agents", propPath)
		}
	case "system":
		// System properties should have generators and be caught earlier
		return fmt.Errorf("%s: property is system-generated and cannot be set manually", propPath)
	default:
		return fmt.Errorf("%s: source validator config has invalid source '%s'", propPath, source)
	}

	return nil
}

// ValidateConfig validates the source validator configuration
func (v *SourceValidator) ValidateConfig(propPath string, config map[string]any) error {
	if len(config) == 0 {
		return nil // No explicit source is valid (defaults to user input)
	}

	sourceRaw, hasSource := config["source"]
	if !hasSource {
		return nil // No source specified is valid
	}

	source, ok := sourceRaw.(string)
	if !ok {
		return fmt.Errorf("source validator config 'source' must be a string")
	}

	// Validate source value
	if source != "agent" && source != "system" {
		return fmt.Errorf("source validator config 'source' must be 'agent' or 'system', got '%s'", source)
	}

	return nil
}

// MutableValidator validates that a property can be updated based on the service's current state.
// This validator only applies to update operations.
type MutableValidator struct{}

// Validate checks if the property can be updated in the service's current state.
// The config should contain "updatableIn" with an array of allowed states.
func (v *MutableValidator) Validate(
	ctx context.Context,
	schemaCtx ServicePropertyContext,
	operation schema.Operation,
	propPath string,
	oldValue, newValue any,
	config map[string]any,
) error {
	// Only apply to update operations
	if operation != schema.OperationUpdate {
		return nil
	}

	// Only validate if value is being changed
	if newValue == nil {
		return nil
	}

	// Service must exist for update operations
	if schemaCtx.Service == nil {
		return fmt.Errorf("%s: mutable validator requires service context for update operations", propPath)
	}

	// Get updatableIn config
	updatableInRaw, hasConfig := config["updatableIn"]
	if !hasConfig {
		return fmt.Errorf("%s: mutable validator config missing 'updatableIn'", propPath)
	}

	updatableIn, ok := updatableInRaw.([]any)
	if !ok {
		return fmt.Errorf("%s: mutable validator config 'updatableIn' must be an array", propPath)
	}

	// Check if current status is in the allowed states
	currentStatus := schemaCtx.Service.Status
	for _, stateRaw := range updatableIn {
		state, ok := stateRaw.(string)
		if !ok {
			continue
		}
		if state == currentStatus {
			return nil // Current state allows updates
		}
	}

	// Current state not in allowed list
	return fmt.Errorf("%s: property cannot be updated in state '%s'", propPath, currentStatus)
}

// ValidateConfig validates the mutable validator configuration
func (v *MutableValidator) ValidateConfig(propPath string, config map[string]any) error {
	if len(config) == 0 {
		return fmt.Errorf("mutable validator config missing 'updatableIn'")
	}

	updatableInRaw, hasConfig := config["updatableIn"]
	if !hasConfig {
		return fmt.Errorf("mutable validator config missing 'updatableIn'")
	}

	updatableIn, ok := updatableInRaw.([]any)
	if !ok {
		return fmt.Errorf("mutable validator config 'updatableIn' must be an array")
	}

	if len(updatableIn) == 0 {
		return fmt.Errorf("mutable validator config 'updatableIn' cannot be empty")
	}

	// Validate each state is a string
	for i, stateRaw := range updatableIn {
		if _, ok := stateRaw.(string); !ok {
			return fmt.Errorf("mutable validator config 'updatableIn[%d]' must be a string", i)
		}
	}

	return nil
}
