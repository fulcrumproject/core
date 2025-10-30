// String validators for minLength, maxLength, and pattern
package schema

import (
	"context"
	"fmt"
	"regexp"
	"sync"
)

// MinLengthValidator validates minimum string length
type MinLengthValidator[C any] struct{}

func (v *MinLengthValidator[C]) Validate(ctx context.Context, schemaCtx C, operation Operation, propPath string, oldValue, newValue any, config map[string]any) error {
	str, ok := newValue.(string)
	if !ok {
		return fmt.Errorf("%s: expected string for minLength validator", propPath)
	}

	minInt, err := getIntConfig(propPath, "minLength", "value", config)
	if err != nil {
		return err
	}

	if len(str) < minInt {
		return fmt.Errorf("%s: string length %d is less than minimum %d", propPath, len(str), minInt)
	}

	return nil
}

func (v *MinLengthValidator[C]) ValidateConfig(propPath string, config map[string]any) error {
	_, err := getNonNegativeIntConfig(propPath, "minLength", "value", config)
	return err
}

// MaxLengthValidator validates maximum string length
type MaxLengthValidator[C any] struct{}

func (v *MaxLengthValidator[C]) Validate(ctx context.Context, schemaCtx C, operation Operation, propPath string, oldValue, newValue any, config map[string]any) error {
	str, ok := newValue.(string)
	if !ok {
		return fmt.Errorf("%s: expected string for maxLength validator", propPath)
	}

	maxInt, err := getIntConfig(propPath, "maxLength", "value", config)
	if err != nil {
		return err
	}

	if len(str) > maxInt {
		return fmt.Errorf("%s: string length %d exceeds maximum %d", propPath, len(str), maxInt)
	}

	return nil
}

func (v *MaxLengthValidator[C]) ValidateConfig(propPath string, config map[string]any) error {
	_, err := getNonNegativeIntConfig(propPath, "maxLength", "value", config)
	return err
}

// PatternValidator validates regex pattern
type PatternValidator[C any] struct {
	cache sync.Map // map[string]*regexp.Regexp
}

func (v *PatternValidator[C]) Validate(ctx context.Context, schemaCtx C, operation Operation, propPath string, oldValue, newValue any, config map[string]any) error {
	str, ok := newValue.(string)
	if !ok {
		return fmt.Errorf("%s: expected string for pattern validator", propPath)
	}

	pattern, ok := config["pattern"].(string)
	if !ok {
		return fmt.Errorf("%s: pattern validator requires 'pattern' config", propPath)
	}

	// Try to load from cache
	var regex *regexp.Regexp
	if cached, ok := v.cache.Load(pattern); ok {
		regex = cached.(*regexp.Regexp)
	} else {
		// Compile and cache
		var err error
		regex, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("%s: invalid regex pattern: %s", propPath, pattern)
		}
		v.cache.Store(pattern, regex)
	}

	if !regex.MatchString(str) {
		return fmt.Errorf("%s: string does not match required pattern", propPath)
	}

	return nil
}

func (v *PatternValidator[C]) ValidateConfig(propPath string, config map[string]any) error {
	pattern, ok := config["pattern"].(string)
	if !ok {
		return fmt.Errorf("%s: pattern validator requires 'pattern' config", propPath)
	}

	// Validate that the pattern is a valid regex
	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("%s: invalid regex pattern: %w", propPath, err)
	}

	return nil
}
