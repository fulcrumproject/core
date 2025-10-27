// Domain-specific validators for service property schema
package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fulcrumproject/core/pkg/schema"
)

// ServiceOptionValidator validates that a property value matches one of the enabled service options
// for a specific service option type and provider.
type ServiceOptionValidator struct{}

// NewServiceOptionValidator creates a new service option validator
func NewServiceOptionValidator() *ServiceOptionValidator {
	return &ServiceOptionValidator{}
}

// Validate checks if the property value matches an enabled service option.
// The config should contain "value" which is the serviceOptionType string (e.g., "os", "machine_type").
func (v *ServiceOptionValidator) Validate(
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

	// Get serviceOptionType from config
	serviceOptionTypeRaw, hasValue := config["value"]
	if !hasValue {
		return fmt.Errorf("%s: serviceOption validator config missing 'value'", propPath)
	}

	serviceOptionType, ok := serviceOptionTypeRaw.(string)
	if !ok {
		return fmt.Errorf("%s: serviceOption validator config 'value' must be a string", propPath)
	}

	// Use provider ID from context
	providerID := schemaCtx.ProviderID

	// Get the service option type to find its ID
	optionType, err := schemaCtx.Store.ServiceOptionTypeRepo().FindByType(ctx, serviceOptionType)
	if err != nil {
		return fmt.Errorf("%s: failed to find service option type '%s': %w", propPath, serviceOptionType, err)
	}

	// Get all enabled service options for this provider and option type
	options, err := schemaCtx.Store.ServiceOptionRepo().ListByProviderAndType(ctx, providerID, optionType.ID)
	if err != nil {
		return fmt.Errorf("%s: failed to retrieve service options: %w", propPath, err)
	}

	// Filter to only enabled options
	var enabledOptions []*ServiceOption
	for _, opt := range options {
		if opt.Enabled {
			enabledOptions = append(enabledOptions, opt)
		}
	}

	if len(enabledOptions) == 0 {
		return fmt.Errorf("%s: no enabled service options available for type '%s'", propPath, serviceOptionType)
	}

	// Check if the new value matches any enabled option
	// Service option values are stored as JSON, so we need to compare properly
	for _, opt := range enabledOptions {
		if opt.Value == nil {
			continue
		}

		// Compare values - handle both JSON and simple types
		if valuesEqual(newValue, opt.Value) {
			return nil // Valid - matches an enabled option
		}
	}

	// Value doesn't match any enabled option
	return fmt.Errorf("%s: value must match one of the enabled service options for type '%s'", propPath, serviceOptionType)
}

// ValidateConfig validates the serviceOption validator configuration
func (v *ServiceOptionValidator) ValidateConfig(propPath string, config map[string]any) error {
	if len(config) == 0 {
		return fmt.Errorf("serviceOption validator config missing 'value'")
	}

	valueRaw, hasValue := config["value"]
	if !hasValue {
		return fmt.Errorf("serviceOption validator config missing 'value'")
	}

	value, ok := valueRaw.(string)
	if !ok {
		return fmt.Errorf("serviceOption validator config 'value' must be a string (serviceOptionType)")
	}

	if value == "" {
		return fmt.Errorf("serviceOption validator config 'value' cannot be empty")
	}

	return nil
}

// valuesEqual compares two values for equality, handling JSON marshaling
func valuesEqual(a, b any) bool {
	// Quick check for nil
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Marshal both values to JSON for comparison
	// This handles all types consistently without panic on non-comparable types
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}

	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}

	return string(aJSON) == string(bJSON)
}
