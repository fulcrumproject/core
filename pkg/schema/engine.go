// Engine orchestrates schema processing
package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
)

// Vault interface for secret storage
type Vault interface {
	Save(ctx context.Context, reference string, value any, metadata map[string]any) error
	Get(ctx context.Context, reference string) (any, error)
	Delete(ctx context.Context, reference string) error
}

// Engine orchestrates schema processing
// C is the context type specific to the domain (e.g., ServicePropertyContext)
type Engine[C any] struct {
	validators       map[string]PropertyValidator[C]
	schemaValidators map[string]SchemaValidator[C]
	generators       map[string]Generator[C]
	vault            Vault // For secret processing
}

// NewEngine creates a new engine with validators, generators, and vault
func NewEngine[C any](
	validators map[string]PropertyValidator[C],
	schemaValidators map[string]SchemaValidator[C],
	generators map[string]Generator[C],
	vault Vault,
) *Engine[C] {
	return &Engine[C]{
		validators:       validators,
		schemaValidators: schemaValidators,
		generators:       generators,
		vault:            vault,
	}
}

// ApplyCreate processes properties for creation according to schema
// Note: Schema must be validated by caller using ValidateSchema before first use
func (e *Engine[C]) ApplyCreate(
	ctx context.Context,
	schemaCtx C,
	schema Schema,
	properties map[string]any,
) (map[string]any, error) {
	return e.apply(ctx, schemaCtx, OperationCreate, schema, nil, properties)
}

// ApplyUpdate processes properties for update according to schema
// Note: Schema must be validated by caller using ValidateSchema before first use
func (e *Engine[C]) ApplyUpdate(
	ctx context.Context,
	schemaCtx C,
	schema Schema,
	oldProperties map[string]any,
	newProperties map[string]any,
) (map[string]any, error) {
	return e.apply(ctx, schemaCtx, OperationUpdate, schema, oldProperties, newProperties)
}

// apply is the internal implementation that processes properties according to schema
func (e *Engine[C]) apply(
	ctx context.Context,
	schemaCtx C,
	operation Operation,
	schema Schema,
	oldProperties map[string]any,
	newProperties map[string]any,
) (map[string]any, error) {
	result := make(map[string]any)
	var validationErrors []ValidationErrorDetail

	// Process each property, collecting all validation errors
	for propName, propDef := range schema.Properties {
		oldValue := oldProperties[propName]
		newValue := newProperties[propName]

		finalValue, err := e.processProperty(ctx, schemaCtx, operation, propName, propDef, oldValue, newValue)
		if err != nil {
			validationErrors = append(validationErrors, ValidationErrorDetail{
				Path:    propName,
				Message: err.Error(),
			})
			continue
		}

		// Store result if not nil
		if finalValue != nil {
			result[propName] = finalValue
		} else if propDef.Required && oldValue == nil {
			validationErrors = append(validationErrors, ValidationErrorDetail{
				Path:    propName,
				Message: "required property is missing",
			})
		}
	}

	// Run schema-level validators (cross-property validation)
	if err := e.validateSchema(ctx, schemaCtx, operation, schema.Validators, oldProperties, result); err != nil {
		validationErrors = append(validationErrors, ValidationErrorDetail{
			Path:    "",
			Message: err.Error(),
		})
	}

	// Return all validation errors at once
	if len(validationErrors) > 0 {
		return nil, NewValidationError(validationErrors)
	}

	return result, nil
}

// processProperty handles the complete processing of a single property
func (e *Engine[C]) processProperty(
	ctx context.Context,
	schemaCtx C,
	operation Operation,
	propName string,
	propDef PropertyDefinition,
	oldValue, newValue any,
) (any, error) {
	// 1. Check if generator property can be set manually
	if propDef.Generator != nil && newValue != nil {
		return nil, fmt.Errorf("%s: property is system-generated", propName)
	}

	// 2. Handle vault references early (skip validation for secret properties)
	if isVaultReference(newValue, propDef.Secret) {
		return newValue, nil
	}

	// 3. Check immutability
	if err := e.checkImmutability(operation, propName, propDef, oldValue, newValue); err != nil {
		return nil, err
	}

	// 4. Validate and process user-provided value
	if newValue != nil {
		if err := e.validatePropertyValue(ctx, schemaCtx, operation, propName, propDef, oldValue, newValue); err != nil {
			return nil, err
		}
		return e.finalizePropertyValue(ctx, schemaCtx, operation, propName, propDef, oldValue, newValue)
	}

	// 5. Apply default or generate value
	finalValue, err := e.applyDefaultOrGenerate(ctx, schemaCtx, propName, propDef, oldValue)
	if err != nil {
		return nil, err
	}

	// 6. If we have a final value, finalize it (recursive validation, secrets, etc.)
	if finalValue != nil {
		return e.finalizePropertyValue(ctx, schemaCtx, operation, propName, propDef, oldValue, finalValue)
	}

	return oldValue, nil
}

// generateSecretReference creates a unique reference for a secret
func generateSecretReference() string {
	return uuid.New().String()
}

// isVaultReference checks if a value is a vault reference for a secret property
func isVaultReference(value any, secretConfig *SecretConfig) bool {
	if value == nil || secretConfig == nil {
		return false
	}
	strVal, ok := value.(string)
	return ok && strings.HasPrefix(strVal, "vault://")
}

// checkImmutability verifies immutability constraints
func (e *Engine[C]) checkImmutability(
	operation Operation,
	propName string,
	propDef PropertyDefinition,
	oldValue, newValue any,
) error {
	// Only check immutability if:
	// 1. Property is marked as immutable
	// 2. New value is being provided
	// 3. Old value exists (property was previously set)
	// 4. Operation is an update (not create)
	if propDef.Immutable && newValue != nil && oldValue != nil && operation == OperationUpdate {
		// Check if the value is actually changing
		if !reflect.DeepEqual(oldValue, newValue) {
			return fmt.Errorf("%s: property is immutable and cannot be changed", propName)
		}
		// If values are equal, allow it (no-op update)
	}
	return nil
}

// validatePropertyValue performs type validation and runs all validators
func (e *Engine[C]) validatePropertyValue(
	ctx context.Context,
	schemaCtx C,
	operation Operation,
	propName string,
	propDef PropertyDefinition,
	oldValue, newValue any,
) error {
	// Type validation first
	if err := e.validateType(propName, newValue, propDef.Type); err != nil {
		return err
	}

	// Run property validators in order (existence guaranteed by schema validation)
	for _, validatorCfg := range propDef.Validators {
		validator := e.validators[validatorCfg.Type]
		if err := validator.Validate(ctx, schemaCtx, operation, propName, oldValue, newValue, validatorCfg.Config); err != nil {
			return err
		}
	}

	return nil
}

// applyDefaultOrGenerate handles default values and generators
func (e *Engine[C]) applyDefaultOrGenerate(
	ctx context.Context,
	schemaCtx C,
	propName string,
	propDef PropertyDefinition,
	oldValue any,
) (any, error) {
	// Apply default value (only on create when no value exists)
	if propDef.Default != nil && oldValue == nil {
		if err := e.validatePropertyValue(ctx, schemaCtx, OperationCreate, propName, propDef, oldValue, propDef.Default); err != nil {
			return nil, fmt.Errorf("%s: default value failed validation: %w", propName, err)
		}
		return propDef.Default, nil
	}

	// Generate value if generator is configured
	if propDef.Generator != nil {
		generator := e.generators[propDef.Generator.Type]

		generatedValue, generated, err := generator.Generate(ctx, schemaCtx, propName, oldValue, propDef.Generator.Config)
		if err != nil {
			return nil, err
		}

		if generated {
			if err := e.validatePropertyValue(ctx, schemaCtx, OperationCreate, propName, propDef, oldValue, generatedValue); err != nil {
				return nil, fmt.Errorf("%s: generated value failed validation: %w", propName, err)
			}
			return generatedValue, nil
		}
	}

	return nil, nil
}

// finalizePropertyValue handles recursive validation and secret processing
func (e *Engine[C]) finalizePropertyValue(
	ctx context.Context,
	schemaCtx C,
	operation Operation,
	propName string,
	propDef PropertyDefinition,
	oldValue, value any,
) (any, error) {
	// 1. Recursive validation for nested structures
	processedValue, err := e.processNestedStructure(ctx, schemaCtx, operation, propName, propDef, oldValue, value)
	if err != nil {
		return nil, err
	}

	// 2. Secret processing (if property is secret)
	if propDef.Secret != nil {
		return e.processSecret(ctx, propName, propDef.Secret, oldValue, processedValue)
	}

	return processedValue, nil
}

// processNestedStructure handles recursive validation for objects and arrays
func (e *Engine[C]) processNestedStructure(
	ctx context.Context,
	schemaCtx C,
	operation Operation,
	propName string,
	propDef PropertyDefinition,
	oldValue, value any,
) (any, error) {
	switch propDef.Type {
	case "object":
		if len(propDef.Properties) > 0 {
			return e.processNestedObject(ctx, schemaCtx, operation, propName, propDef, oldValue, value)
		}

	case "array":
		if propDef.Items != nil {
			return e.processNestedArray(ctx, schemaCtx, operation, propName, propDef, oldValue, value)
		}
	}

	return value, nil
}

// processNestedObject recursively processes nested object properties
func (e *Engine[C]) processNestedObject(
	ctx context.Context,
	schemaCtx C,
	operation Operation,
	propName string,
	propDef PropertyDefinition,
	oldValue, value any,
) (any, error) {
	objValue, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected object but got %T", propName, value)
	}

	var oldObjValue map[string]any
	if oldValue != nil {
		oldObjValue, _ = oldValue.(map[string]any)
	}

	// Create nested schema and recursively process
	nestedSchema := Schema{Properties: propDef.Properties}

	return e.apply(ctx, schemaCtx, operation, nestedSchema, oldObjValue, objValue)
}

// processNestedArray recursively processes array items
func (e *Engine[C]) processNestedArray(
	ctx context.Context,
	schemaCtx C,
	operation Operation,
	propName string,
	propDef PropertyDefinition,
	oldValue, value any,
) (any, error) {
	arrValue, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected array but got %T", propName, value)
	}

	processedArr := make([]any, len(arrValue))

	for i, item := range arrValue {
		var oldItem any
		if oldValue != nil {
			if oldArr, ok := oldValue.([]any); ok && i < len(oldArr) {
				oldItem = oldArr[i]
			}
		}

		itemPropName := fmt.Sprintf("%s[%d]", propName, i)

		// Validate the item
		if err := e.validatePropertyValue(ctx, schemaCtx, operation, itemPropName, *propDef.Items, oldItem, item); err != nil {
			return nil, err
		}

		// Process nested structures in array items
		processedItem, err := e.processNestedStructure(ctx, schemaCtx, operation, itemPropName, *propDef.Items, oldItem, item)
		if err != nil {
			return nil, err
		}

		processedArr[i] = processedItem
	}

	return processedArr, nil
}

// processSecret handles vault storage and secret rotation
func (e *Engine[C]) processSecret(
	ctx context.Context,
	propName string,
	secretConfig *SecretConfig,
	oldValue, value any,
) (any, error) {
	// Check vault availability
	if e.vault == nil {
		return nil, fmt.Errorf("%s: vault is required for secret properties but not configured", propName)
	}

	// Check if already a vault reference
	if strVal, ok := value.(string); ok && strings.HasPrefix(strVal, "vault://") {
		return strVal, nil
	}

	// Store new secret value in vault
	reference := generateSecretReference()
	err := e.vault.Save(ctx, reference, value, map[string]any{
		"secretType":   secretConfig.Type,
		"propertyPath": propName,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: failed to store secret: %w", propName, err)
	}

	// Clean up old vault reference if this is a rotation
	if oldStrVal, ok := oldValue.(string); ok && strings.HasPrefix(oldStrVal, "vault://") {
		oldRef := strings.TrimPrefix(oldStrVal, "vault://")
		_ = e.vault.Delete(ctx, oldRef) // Best-effort cleanup
	}

	return fmt.Sprintf("vault://%s", reference), nil
}

// validateSchema runs schema-level validators
func (e *Engine[C]) validateSchema(
	ctx context.Context,
	schemaCtx C,
	operation Operation,
	validators []SchemaValidatorConfig,
	oldProperties, newProperties map[string]any,
) error {
	for _, validatorCfg := range validators {
		validator := e.schemaValidators[validatorCfg.Type]
		if err := validator.Validate(ctx, schemaCtx, operation, oldProperties, newProperties, validatorCfg.Config); err != nil {
			return err
		}
	}
	return nil
}

// ValidateSchema validates that the schema definition is valid
func (e *Engine[C]) ValidateSchema(schema Schema) error {
	// Validate each property definition
	for propName, propDef := range schema.Properties {
		if propName == "" {
			return fmt.Errorf("property name cannot be empty")
		}
		if err := e.validatePropertyDefinition(propName, propDef); err != nil {
			return err
		}
	}

	// Validate schema-level validators exist and have valid config
	for _, validatorCfg := range schema.Validators {
		validator, ok := e.schemaValidators[validatorCfg.Type]
		if !ok {
			return fmt.Errorf("unknown schema validator: %s", validatorCfg.Type)
		}
		if err := validator.ValidateConfig(validatorCfg.Config); err != nil {
			return err
		}
	}

	return nil
}

// validatePropertyDefinition recursively validates a property definition
func (e *Engine[C]) validatePropertyDefinition(propPath string, propDef PropertyDefinition) error {
	// 1. Validate type is known
	validTypes := map[string]bool{
		"string": true, "integer": true, "number": true, "boolean": true,
		"object": true, "array": true, "json": true,
	}
	if !validTypes[propDef.Type] {
		return fmt.Errorf("%s: invalid type '%s'", propPath, propDef.Type)
	}

	// 2. Array items cannot be secret
	if propDef.Type == "array" && propDef.Items != nil && propDef.Items.Secret != nil {
		return fmt.Errorf("%s: array items cannot be secret", propPath)
	}

	// 3. Objects and arrays cannot be secret (only primitives)
	if propDef.Secret != nil && (propDef.Type == "object" || propDef.Type == "array") {
		return fmt.Errorf("%s: only primitive types can be secret", propPath)
	}

	// 4. Validate default value type matches property type
	if propDef.Default != nil {
		if err := e.validateType(propPath, propDef.Default, propDef.Type); err != nil {
			return fmt.Errorf("%s: default value type mismatch: %w", propPath, err)
		}
	}

	// 5. Check for conflicting attributes
	if propDef.Default != nil && propDef.Generator != nil {
		return fmt.Errorf("%s: cannot have both default and generator", propPath)
	}

	// 6. Validate nested object properties
	if propDef.Type == "object" && len(propDef.Properties) > 0 {
		for nestedName, nestedDef := range propDef.Properties {
			if nestedName == "" {
				return fmt.Errorf("%s: nested property name cannot be empty", propPath)
			}
			nestedPath := fmt.Sprintf("%s.%s", propPath, nestedName)
			if err := e.validatePropertyDefinition(nestedPath, nestedDef); err != nil {
				return err
			}
		}
	}

	// 7. Validate array items recursively
	if propDef.Type == "array" && propDef.Items != nil {
		itemPath := fmt.Sprintf("%s[]", propPath)
		if err := e.validatePropertyDefinition(itemPath, *propDef.Items); err != nil {
			return err
		}
	}

	// 8. Validate validators exist and have valid config
	for _, validatorCfg := range propDef.Validators {
		validator, ok := e.validators[validatorCfg.Type]
		if !ok {
			return fmt.Errorf("%s: unknown validator '%s'", propPath, validatorCfg.Type)
		}
		if err := validator.ValidateConfig(propPath, validatorCfg.Config); err != nil {
			return err
		}
	}

	// 9. Validate generator exists and has valid config
	if propDef.Generator != nil {
		generator, ok := e.generators[propDef.Generator.Type]
		if !ok {
			return fmt.Errorf("%s: unknown generator '%s'", propPath, propDef.Generator.Type)
		}
		if err := generator.ValidateConfig(propPath, propDef.Generator.Config); err != nil {
			return err
		}
	}

	// 10. Secret type must be valid
	if propDef.Secret != nil {
		if propDef.Secret.Type != "persistent" && propDef.Secret.Type != "ephemeral" {
			return fmt.Errorf("%s: secret type must be 'persistent' or 'ephemeral'", propPath)
		}
	}

	return nil
}

// validateType checks if value matches the declared type
func (e *Engine[C]) validateType(propName string, value any, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s: expected string, got %T", propName, value)
		}
	case "integer":
		// Accept various integer types
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return nil
		case float32:
			if float32(int64(v)) != v {
				return fmt.Errorf("%s: expected integer, got float with decimal part", propName)
			}
			return nil
		case float64:
			// JSON unmarshals numbers as float64
			if float64(int64(v)) != v {
				return fmt.Errorf("%s: expected integer, got float with decimal part", propName)
			}
			return nil
		case json.Number:
			// Handle json.Number (used when JSON decoder uses UseNumber())
			_, err := v.Int64()
			if err != nil {
				// Try as float to check if it's a whole number
				floatVal, floatErr := v.Float64()
				if floatErr != nil {
					return fmt.Errorf("%s: expected integer, got invalid number", propName)
				}
				if floatVal != float64(int64(floatVal)) {
					return fmt.Errorf("%s: expected integer, got float with decimal part", propName)
				}
				return nil
			}
			return nil
		default:
			return fmt.Errorf("%s: expected integer, got %T", propName, value)
		}
	case "number":
		// Accept all numeric types (integers and floats)
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return nil
		case json.Number:
			// Handle json.Number (used when JSON decoder uses UseNumber())
			_, err := v.Float64()
			if err != nil {
				return fmt.Errorf("%s: expected number, got invalid number", propName)
			}
			return nil
		default:
			return fmt.Errorf("%s: expected number, got %T", propName, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s: expected boolean, got %T", propName, value)
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("%s: expected object, got %T", propName, value)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("%s: expected array, got %T", propName, value)
		}
	case "json":
		// Any valid JSON value is acceptable
		return nil
	default:
		return fmt.Errorf("%s: unknown type: %s", propName, expectedType)
	}
	return nil
}
