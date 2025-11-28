package schema

import (
	"context"
	"encoding/json"
	"fmt"
)

// ExactlyOneValidator ensures exactly one property from a group must be provided
type ExactlyOneValidator[C any] struct{}

func (v *ExactlyOneValidator[C]) Validate(ctx context.Context, schemaCtx C, operation Operation, oldProperties, newProperties map[string]any, config map[string]any) error {
	propsRaw, ok := config["properties"].([]any)
	if !ok {
		return fmt.Errorf("exactlyOne validator requires 'properties' config")
	}

	// Convert to string slice
	props := make([]string, 0, len(propsRaw))
	for _, p := range propsRaw {
		if propStr, ok := p.(string); ok {
			props = append(props, propStr)
		}
	}

	if len(props) < 2 {
		return fmt.Errorf("exactlyOne validator requires at least 2 properties")
	}

	// Count how many properties are provided
	providedCount := 0
	var providedProps []string

	for _, prop := range props {
		if val, exists := newProperties[prop]; exists && val != nil {
			providedCount++
			providedProps = append(providedProps, prop)
		}
	}

	if providedCount == 0 {
		return fmt.Errorf("exactly one of %v must be provided", props)
	}

	if providedCount > 1 {
		return fmt.Errorf("only one of %v can be provided, got: %v", props, providedProps)
	}

	return nil
}

func (v *ExactlyOneValidator[C]) ValidateConfig(config map[string]any) error {
	propsRaw, ok := config["properties"].([]any)
	if !ok {
		return fmt.Errorf("exactlyOne validator requires 'properties' config as array")
	}

	props := make([]string, 0, len(propsRaw))
	for _, p := range propsRaw {
		propStr, ok := p.(string)
		if !ok {
			return fmt.Errorf("exactlyOne validator: all properties must be strings, got %T", p)
		}
		props = append(props, propStr)
	}

	if len(props) < 2 {
		return fmt.Errorf("exactlyOne validator requires at least 2 properties")
	}

	return nil
}

// UniqueValuesValidator ensures all specified properties have unique (different) values.
type UniqueValuesValidator[C any] struct{}

func (v *UniqueValuesValidator[C]) Validate(ctx context.Context, schemaCtx C, operation Operation, oldProperties, newProperties map[string]any, config map[string]any) error {
	// Extract property names from config
	propsRaw, ok := config["properties"].([]any)
	if !ok {
		return fmt.Errorf("uniqueValues validator requires 'properties' config")
	}

	// Container for seen values for uniqueness check
	// Key: JSON-serialized value, Value: property name
	seenValues := make(map[string]string)

	// Individual property validation
	for _, p := range propsRaw {
		// Type assertion to string
		propStr, ok := p.(string)

		// Skip non-string property names
		if !ok {
			continue
		}

		val, exists := newProperties[propStr]

		// Skip nil or missing properties - only validate provided values
		if !exists || val == nil {
			continue
		}

		// Try JSON serialization for consistent comparison between different types
		jsonBytes, err := json.Marshal(val)
		// Fallback for non-JSON-serializable values
		if err != nil {
			valueStr := fmt.Sprintf("%v", val)
			jsonBytes = []byte(valueStr)
		}

		valueKey := string(jsonBytes)

		// Check if this value was already seen
		if existingProp, found := seenValues[valueKey]; found {
			return fmt.Errorf(
				"properties %s and %s must have unique values, both have: %v",
				existingProp,
				propStr,
				val,
			)
		}

		// Record this value as seen
		seenValues[valueKey] = propStr
	}

	return nil
}

func (v *UniqueValuesValidator[C]) ValidateConfig(config map[string]any) error {
	propsRaw, ok := config["properties"].([]any)

	// Malformed config
	if !ok {
		return fmt.Errorf("uniqueValues validator requires 'properties' config as array")
	}

	// Convert to string slice
	props := make([]string, 0, len(propsRaw))

	// Validate all entries are strings
	for _, p := range propsRaw {
		propStr, ok := p.(string)
		if !ok {
			return fmt.Errorf("uniqueValues validator: all properties must be strings, got %T", p)
		}
		props = append(props, propStr)
	}

	// Need at least 2 properties to compare
	if len(props) < 2 {
		return fmt.Errorf("uniqueValues validator requires at least 2 properties")
	}

	return nil
}
