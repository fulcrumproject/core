// Collection validators for enum, minItems, and maxItems
package schema

import (
	"context"
	"fmt"
	"reflect"
)

// EnumValidator validates that value is in allowed enum values
type EnumValidator[C any] struct{}

func (v *EnumValidator[C]) Validate(ctx context.Context, schemaCtx C, operation Operation, propPath string, oldValue, newValue any, config map[string]any) error {
	values, ok := config["values"]
	if !ok {
		return fmt.Errorf("%s: enum validator requires 'values' config", propPath)
	}

	// Convert to slice
	valuesSlice, ok := values.([]any)
	if !ok {
		return fmt.Errorf("%s: enum 'values' config must be an array", propPath)
	}

	// Check if newValue is in the allowed values
	for _, enumValue := range valuesSlice {
		if reflect.DeepEqual(newValue, enumValue) {
			return nil
		}
	}

	return fmt.Errorf("%s: value not in allowed enum values", propPath)
}

func (v *EnumValidator[C]) ValidateConfig(propPath string, config map[string]any) error {
	values, ok := config["values"].([]any)
	if !ok {
		return fmt.Errorf("%s: enum validator requires 'values' config as array", propPath)
	}

	if len(values) == 0 {
		return fmt.Errorf("%s: enum must have at least one value", propPath)
	}

	return nil
}

// MinItemsValidator validates minimum array length
type MinItemsValidator[C any] struct{}

func (v *MinItemsValidator[C]) Validate(ctx context.Context, schemaCtx C, operation Operation, propPath string, oldValue, newValue any, config map[string]any) error {
	arr, ok := newValue.([]any)
	if !ok {
		return fmt.Errorf("%s: expected array for minItems validator", propPath)
	}

	minInt, err := getIntConfig(propPath, "minItems", "value", config)
	if err != nil {
		return err
	}

	if len(arr) < minInt {
		return fmt.Errorf("%s: array length %d is less than minimum %d", propPath, len(arr), minInt)
	}

	return nil
}

func (v *MinItemsValidator[C]) ValidateConfig(propPath string, config map[string]any) error {
	_, err := getNonNegativeIntConfig(propPath, "minItems", "value", config)
	return err
}

// MaxItemsValidator validates maximum array length
type MaxItemsValidator[C any] struct{}

func (v *MaxItemsValidator[C]) Validate(ctx context.Context, schemaCtx C, operation Operation, propPath string, oldValue, newValue any, config map[string]any) error {
	arr, ok := newValue.([]any)
	if !ok {
		return fmt.Errorf("%s: expected array for maxItems validator", propPath)
	}

	maxInt, err := getIntConfig(propPath, "maxItems", "value", config)
	if err != nil {
		return err
	}

	if len(arr) > maxInt {
		return fmt.Errorf("%s: array length %d exceeds maximum %d", propPath, len(arr), maxInt)
	}

	return nil
}

func (v *MaxItemsValidator[C]) ValidateConfig(propPath string, config map[string]any) error {
	_, err := getNonNegativeIntConfig(propPath, "maxItems", "value", config)
	return err
}
