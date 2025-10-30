// Numeric validators for min and max
package schema

import (
	"context"
	"fmt"
)

// MinValidator validates minimum numeric value
type MinValidator[C any] struct{}

func (v *MinValidator[C]) Validate(ctx context.Context, schemaCtx C, operation Operation, propPath string, oldValue, newValue any, config map[string]any) error {
	// Convert value to float64 for comparison
	num, err := convertToFloat64(propPath, "min", newValue)
	if err != nil {
		return err
	}

	minFloat, err := getFloatConfig(propPath, "min", "value", config)
	if err != nil {
		return err
	}

	if num < minFloat {
		return fmt.Errorf("%s: value %v is less than minimum %v", propPath, num, minFloat)
	}

	return nil
}

func (v *MinValidator[C]) ValidateConfig(propPath string, config map[string]any) error {
	_, ok := config["value"]
	if !ok {
		return fmt.Errorf("%s: min validator requires 'value' config", propPath)
	}

	// Value can be any numeric type, no additional validation needed
	return nil
}

// MaxValidator validates maximum numeric value
type MaxValidator[C any] struct{}

func (v *MaxValidator[C]) Validate(ctx context.Context, schemaCtx C, operation Operation, propPath string, oldValue, newValue any, config map[string]any) error {
	// Convert value to float64 for comparison
	num, err := convertToFloat64(propPath, "max", newValue)
	if err != nil {
		return err
	}

	maxFloat, err := getFloatConfig(propPath, "max", "value", config)
	if err != nil {
		return err
	}

	if num > maxFloat {
		return fmt.Errorf("%s: value %v exceeds maximum %v", propPath, num, maxFloat)
	}

	return nil
}

func (v *MaxValidator[C]) ValidateConfig(propPath string, config map[string]any) error {
	_, ok := config["value"]
	if !ok {
		return fmt.Errorf("%s: max validator requires 'value' config", propPath)
	}

	// Value can be any numeric type, no additional validation needed
	return nil
}
