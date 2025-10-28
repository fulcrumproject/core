// Domain-specific validators for service property schema
package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
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

// ServiceReferenceValidator validates service references
type ServiceReferenceValidator struct{}

// NewServiceReferenceValidator creates a new service reference validator
func NewServiceReferenceValidator() *ServiceReferenceValidator {
	return &ServiceReferenceValidator{}
}

// Validate checks:
// - Service exists
// - Service type matches (if types specified in config)
// - Origin constraint (same consumer or group, if origin specified in config)
func (v *ServiceReferenceValidator) Validate(
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

	// Parse the UUID
	serviceIDStr, ok := newValue.(string)
	if !ok {
		return fmt.Errorf("%s: expected string uuid, got %T", propPath, newValue)
	}

	serviceID, err := parseUUID(serviceIDStr)
	if err != nil {
		return fmt.Errorf("%s: invalid service uuid: %w", propPath, err)
	}

	// Load the referenced service
	referencedService, err := schemaCtx.Store.ServiceRepo().Get(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("%s: referenced service not found: %w", propPath, err)
	}

	// Validate service type if specified
	if typesRaw, ok := config["types"]; ok {
		if err := v.validateServiceType(ctx, schemaCtx, propPath, referencedService, typesRaw); err != nil {
			return err
		}
	}

	// Validate origin constraint if specified
	if originRaw, ok := config["origin"]; ok {
		if err := v.validateOrigin(ctx, propPath, schemaCtx, referencedService, originRaw); err != nil {
			return err
		}
	}

	return nil
}

// validateServiceType checks if the referenced service is of an allowed type
func (v *ServiceReferenceValidator) validateServiceType(
	ctx context.Context,
	schemaCtx ServicePropertyContext,
	propPath string,
	referencedService *Service,
	typesConfig any,
) error {
	// Load the referenced service's type
	serviceType, err := schemaCtx.Store.ServiceTypeRepo().Get(ctx, referencedService.ServiceTypeID)
	if err != nil {
		return fmt.Errorf("%s: failed to load service type: %w", propPath, err)
	}

	// types must be an array
	typesArray, ok := typesConfig.([]any)
	if !ok {
		return fmt.Errorf("%s: types config must be an array of strings", propPath)
	}

	// Convert to string slice and check for match
	found := false
	allowedTypes := make([]string, 0, len(typesArray))
	for _, t := range typesArray {
		typeStr, ok := t.(string)
		if !ok {
			return fmt.Errorf("%s: types array must contain only strings", propPath)
		}
		allowedTypes = append(allowedTypes, typeStr)
		if serviceType.Name == typeStr {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("%s: service must be one of types %v, got '%s'", propPath, allowedTypes, serviceType.Name)
	}

	return nil
}

// validateOrigin checks if the referenced service has the same consumer or group
func (v *ServiceReferenceValidator) validateOrigin(
	_ context.Context,
	propPath string,
	schemaCtx ServicePropertyContext,
	referencedService *Service,
	originConfig any,
) error {
	origin, ok := originConfig.(string)
	if !ok {
		return fmt.Errorf("%s: origin config must be a string", propPath)
	}

	switch origin {
	case "consumer":
		if schemaCtx.ConsumerID != referencedService.ConsumerID {
			return fmt.Errorf("%s: referenced service must belong to the same consumer", propPath)
		}
	case "group":
		if schemaCtx.GroupID != referencedService.GroupID {
			return fmt.Errorf("%s: referenced service must belong to the same service group", propPath)
		}
	default:
		return fmt.Errorf("%s: unknown origin type '%s', must be 'consumer' or 'group'", propPath, origin)
	}

	return nil
}

// ValidateConfig validates the serviceReference validator configuration
func (v *ServiceReferenceValidator) ValidateConfig(propPath string, config map[string]any) error {
	// Config is optional, but if provided must be valid
	if len(config) == 0 {
		return nil // No constraints is valid
	}

	// Validate types if provided
	if typesRaw, ok := config["types"]; ok {
		typesArray, ok := typesRaw.([]any)
		if !ok {
			return fmt.Errorf("serviceReference validator 'types' must be an array")
		}
		if len(typesArray) == 0 {
			return fmt.Errorf("serviceReference validator 'types' array cannot be empty")
		}
		for _, t := range typesArray {
			if _, ok := t.(string); !ok {
				return fmt.Errorf("serviceReference validator 'types' array must contain only strings")
			}
		}
	}

	// Validate origin if provided
	if originRaw, ok := config["origin"]; ok {
		origin, ok := originRaw.(string)
		if !ok {
			return fmt.Errorf("serviceReference validator 'origin' must be a string")
		}
		if origin != "consumer" && origin != "group" {
			return fmt.Errorf("serviceReference validator 'origin' must be 'consumer' or 'group'")
		}
	}

	return nil
}

// parseUUID is a helper to parse UUID strings
func parseUUID(s string) (properties.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return properties.UUID{}, err
	}
	return properties.UUID(id), nil
}
