package schema

import (
	"context"
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
