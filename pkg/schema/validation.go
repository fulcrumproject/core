package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
)

// Validator type constants
const (
	ValidatorMinLength   = "minLength"
	ValidatorMaxLength   = "maxLength"
	ValidatorPattern     = "pattern"
	ValidatorEnum        = "enum"
	ValidatorMin         = "min"
	ValidatorMax         = "max"
	ValidatorMinItems    = "minItems"
	ValidatorMaxItems    = "maxItems"
	ValidatorUniqueItems = "uniqueItems"
)

// Schema type constants
const (
	TypeString  = "string"
	TypeInteger = "integer"
	TypeNumber  = "number"
	TypeBoolean = "boolean"
	TypeObject  = "object"
	TypeArray   = "array"
)

// Standard target types for each schema type
// These are the canonical types we convert to for consistent comparisons
type (
	StandardString  = string
	StandardInteger = int64
	StandardNumber  = float64
	StandardBoolean = bool
	StandardObject  = map[string]any
	StandardArray   = []any
)

// Error message templates
const (
	ErrUnknownProperty           = "unknown property"
	ErrRequiredFieldMissing      = "required field is missing"
	ErrUnknownValidatorType      = "unknown validator type: %s"
	ErrUnknownSchemaType         = "unknown type: %s"
	ErrExpectedType              = "expected %s, got %T"
	ErrExpectedIntegerGotFloat   = "expected integer, got float with decimal part"
	ErrValidatorOnlyForType      = "%s validator can only be applied to %s"
	ErrValidatorValueMustBeType  = "%s validator value must be %s"
	ErrStringLengthLessThanMin   = "string length %d is less than minimum %d"
	ErrStringLengthExceedsMax    = "string length %d exceeds maximum %d"
	ErrInvalidRegexPattern       = "invalid regex pattern: %s"
	ErrStringDoesNotMatchPattern = "string does not match pattern %s"
	ErrValueNotInEnum            = "value is not in allowed enum values: %s"
	ErrValueLessThanMin          = "value %v is less than minimum %v"
	ErrValueExceedsMax           = "value %v exceeds maximum %v"
	ErrArrayLengthLessThanMin    = "array length %d is less than minimum %d"
	ErrArrayLengthExceedsMax     = "array length %d exceeds maximum %d"
	ErrArrayContainsDuplicates   = "array contains duplicate items"
	ErrFloatHasDecimalPart       = "float value has decimal part"
	ErrCannotConvertType         = "cannot convert %T to %T"
)

// ValidateWithDefaults validates data against the schema and applies defaults
func ValidateWithDefaults(data map[string]any, schema CustomSchema) (map[string]any, []ValidationError) {
	// Apply defaults first
	dataWithDefaults := ApplyDefaults(data, schema)

	// Then validate
	errors := Validate(dataWithDefaults, schema)

	return dataWithDefaults, errors
}

// ApplyDefaults applies default values to data based on the schema
func ApplyDefaults(data map[string]any, schema CustomSchema) map[string]any {
	result := make(map[string]any)

	// Copy existing data
	for key, value := range data {
		result[key] = value
	}

	// Apply defaults for missing properties
	for propName, propDef := range schema {
		if _, exists := result[propName]; !exists && propDef.Default != nil {
			result[propName] = propDef.Default
		} else if _, exists := result[propName]; exists {
			// Apply nested defaults for objects
			if propDef.Type == TypeObject && propDef.Properties != nil {
				if objValue, ok := result[propName].(map[string]any); ok {
					result[propName] = ApplyDefaults(objValue, propDef.Properties)
				}
			}
		}
	}

	return result
}

// Validate validates data against the provided schema
func Validate(data map[string]any, schema CustomSchema) []ValidationError {
	var errors []ValidationError

	// Check required fields
	for propName, propDef := range schema {
		if propDef.Required {
			if _, exists := data[propName]; !exists {
				errors = append(errors, ValidationError{
					Path:    propName,
					Message: ErrRequiredFieldMissing,
				})
			} else if data[propName] == nil {
				errors = append(errors, ValidationError{
					Path:    propName,
					Message: ErrRequiredFieldMissing,
				})
			}
		}
	}

	// Validate each property in data
	for propName, value := range data {
		propDef, exists := schema[propName]
		if !exists {
			errors = append(errors, ValidationError{
				Path:    propName,
				Message: ErrUnknownProperty,
			})
			continue
		}

		propErrors := validateProperty(propName, value, propDef)
		errors = append(errors, propErrors...)
	}

	return errors
}

// validateProperty validates a single property value against its definition
func validateProperty(path string, value any, propDef PropertyDefinition) []ValidationError {
	var errors []ValidationError

	// Type validation - get the converted standard value
	standardValue, err := convertToSchemaStandardType(value, propDef.Type)
	if err != nil {
		errors = append(errors, ValidationError{
			Path:    path,
			Message: err.Error(),
		})
		return errors // Don't continue if type is wrong
	}

	// Validator rules - use the already converted standard value
	for _, validator := range propDef.Validators {
		if err := applyValidator(standardValue, validator, propDef.Type); err != nil {
			errors = append(errors, ValidationError{
				Path:    path,
				Message: err.Error(),
			})
		}
	}

	// Nested validation for objects
	if propDef.Type == TypeObject && propDef.Properties != nil {
		if objValue, ok := standardValue.(map[string]any); ok {
			nestedErrors := Validate(objValue, propDef.Properties)
			for _, nestedErr := range nestedErrors {
				errors = append(errors, ValidationError{
					Path:    path + "." + nestedErr.Path,
					Message: nestedErr.Message,
				})
			}
		}
	}

	// Array item validation
	if propDef.Type == TypeArray && propDef.Items != nil {
		if arrValue, ok := standardValue.([]any); ok {
			for i, item := range arrValue {
				itemPath := fmt.Sprintf("%s[%d]", path, i)
				itemErrors := validateProperty(itemPath, item, *propDef.Items)
				errors = append(errors, itemErrors...)
			}
		}
	}

	return errors
}

// applyValidator applies a specific validator to a value
func applyValidator(value any, validator ValidatorDefinition, propertyType string) error {
	switch validator.Type {
	case ValidatorMinLength:
		return validateMinLength(value, validator.Value)
	case ValidatorMaxLength:
		return validateMaxLength(value, validator.Value)
	case ValidatorPattern:
		return validatePattern(value, validator.Value)
	case ValidatorEnum:
		return validateEnum(value, validator.Value, propertyType)
	case ValidatorMin:
		return validateMin(value, validator.Value)
	case ValidatorMax:
		return validateMax(value, validator.Value)
	case ValidatorMinItems:
		return validateMinItems(value, validator.Value)
	case ValidatorMaxItems:
		return validateMaxItems(value, validator.Value)
	case ValidatorUniqueItems:
		return validateUniqueItems(value, validator.Value)
	default:
		return fmt.Errorf(ErrUnknownValidatorType, validator.Type)
	}
}

// validateMinLength is a specific validator for minimum string length
func validateMinLength(standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardString by validateType
	standardStr, ok := standardValue.(StandardString)
	if !ok {
		return fmt.Errorf(ErrValidatorOnlyForType, ValidatorMinLength, "a string")
	}

	// Convert validator value to standard integer
	standardLimit, err := convertToStandardInteger(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrValidatorValueMustBeType, ValidatorMinLength, "an integer")
	}

	if int64(len(standardStr)) < standardLimit {
		return fmt.Errorf(ErrStringLengthLessThanMin, len(standardStr), standardLimit)
	}

	return nil
}

// validateMaxLength is a specific validator for maximum string length
func validateMaxLength(standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardString by validateType
	standardStr, ok := standardValue.(StandardString)
	if !ok {
		return fmt.Errorf(ErrValidatorOnlyForType, ValidatorMinLength, "a string")
	}

	// Convert validator value to standard integer
	standardLimit, err := convertToStandardInteger(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrValidatorValueMustBeType, ValidatorMaxLength, "an integer")
	}

	if int64(len(standardStr)) > standardLimit {
		return fmt.Errorf(ErrStringLengthExceedsMax, len(standardStr), standardLimit)
	}

	return nil
}

// validatePattern checks if a string matches a regex pattern
func validatePattern(standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardString by validateType
	standardStr, ok := standardValue.(StandardString)
	if !ok {
		return fmt.Errorf(ErrValidatorOnlyForType, ValidatorPattern, "a string")
	}

	// Convert validator value to standard string type
	pattern, err := convertToStandardString(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrValidatorValueMustBeType, ValidatorPattern, "a string")
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf(ErrInvalidRegexPattern, pattern)
	}

	if !regex.MatchString(standardStr) {
		return fmt.Errorf(ErrStringDoesNotMatchPattern, pattern)
	}

	return nil
}

// validateEnum checks if a value is in the allowed enum values
func validateEnum(standardValue any, validatorValue any, propertyType string) error {
	// Convert validator value to standard array type
	enumArray, err := convertToStandardArray(validatorValue)
	if err != nil {
		return fmt.Errorf("enum validator value must be an array")
	}

	// Convert each enum value to the standard type and compare with the already-converted standardValue
	for _, enumValue := range enumArray {
		standardEnumValue, err := convertToSchemaStandardType(enumValue, propertyType)
		if err != nil {
			continue // Skip invalid enum values
		}

		if reflect.DeepEqual(standardValue, standardEnumValue) {
			return nil
		}
	}

	return fmt.Errorf(ErrValueNotInEnum, formatEnumValues(enumArray))
}

// formatEnumValues formats a slice of enum values into a nice string representation
func formatEnumValues(enumValues []any) string {
	if len(enumValues) == 0 {
		return "[]"
	}

	var formattedValues []string
	for _, val := range enumValues {
		formattedValues = append(formattedValues, fmt.Sprintf("%v", val))
	}

	result := "[" + formattedValues[0]
	for i := 1; i < len(formattedValues); i++ {
		result += ", " + formattedValues[i]
	}
	result += "]"

	return result
}

// validateMin is a specific validator for minimum numeric value
func validateMin(standardValue any, validatorValue any) error {
	// Convert standardValue to float64 for comparison (handles both int64 and float64)
	var standardNum float64
	switch v := standardValue.(type) {
	case StandardInteger:
		standardNum = float64(v)
	case StandardNumber:
		standardNum = v
	default:
		return fmt.Errorf(ErrValidatorOnlyForType, ValidatorMin, "numbers")
	}

	// Convert validator value to standard number type
	standardLimit, err := convertToStandardNumber(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrValidatorValueMustBeType, ValidatorMin, "a number")
	}

	if standardNum < standardLimit {
		return fmt.Errorf(ErrValueLessThanMin, standardNum, standardLimit)
	}

	return nil
}

// validateMax is a specific validator for maximum numeric value
func validateMax(standardValue any, validatorValue any) error {
	// Convert standardValue to float64 for comparison (handles both int64 and float64)
	var standardNum float64
	switch v := standardValue.(type) {
	case StandardInteger:
		standardNum = float64(v)
	case StandardNumber:
		standardNum = v
	default:
		return fmt.Errorf(ErrValidatorOnlyForType, ValidatorMax, "numbers")
	}

	// Convert validator value to standard number type
	standardLimit, err := convertToStandardNumber(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrValidatorValueMustBeType, ValidatorMax, "a number")
	}

	if standardNum > standardLimit {
		return fmt.Errorf(ErrValueExceedsMax, standardNum, standardLimit)
	}

	return nil
}

// validateMinItems is a specific validator for minimum array length
func validateMinItems(standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardArray by validateType
	standardArr, ok := standardValue.(StandardArray)
	if !ok {
		return fmt.Errorf(ErrValidatorOnlyForType, ValidatorMinItems, "an array")
	}

	// Convert validator value to standard integer
	standardLimit, err := convertToStandardInteger(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrValidatorValueMustBeType, ValidatorMinItems, "an integer")
	}

	if int64(len(standardArr)) < standardLimit {
		return fmt.Errorf(ErrArrayLengthLessThanMin, len(standardArr), standardLimit)
	}

	return nil
}

// validateMaxItems is a specific validator for maximum array length
func validateMaxItems(standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardArray by validateType
	standardArr, ok := standardValue.(StandardArray)
	if !ok {
		return fmt.Errorf(ErrValidatorOnlyForType, ValidatorMaxItems, "an array")
	}

	// Convert validator value to standard integer
	standardLimit, err := convertToStandardInteger(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrValidatorValueMustBeType, ValidatorMaxItems, "an integer")
	}

	if int64(len(standardArr)) > standardLimit {
		return fmt.Errorf(ErrArrayLengthExceedsMax, len(standardArr), standardLimit)
	}

	return nil
}

// validateUniqueItems checks if an array contains unique items
func validateUniqueItems(standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardArray by validateType
	standardArr, ok := standardValue.(StandardArray)
	if !ok {
		return fmt.Errorf(ErrValidatorOnlyForType, ValidatorUniqueItems, "an array")
	}

	// Convert validator value to standard boolean
	unique, err := convertToStandardBoolean(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrValidatorValueMustBeType, ValidatorUniqueItems, "a boolean")
	}

	if !unique {
		return nil // uniqueItems: false means no validation needed
	}

	seen := make(map[string]bool)
	for _, item := range standardArr {
		key := fmt.Sprintf("%v", item)
		if seen[key] {
			return errors.New(ErrArrayContainsDuplicates)
		}
		seen[key] = true
	}

	return nil
}

// convertToSchemaStandardType converts a value to the standard type for a given schema type
func convertToSchemaStandardType(value any, schemaType string) (any, error) {
	if value == nil {
		return nil, nil
	}

	switch schemaType {
	case TypeString:
		return convertToStandardString(value)
	case TypeInteger:
		return convertToStandardInteger(value)
	case TypeNumber:
		return convertToStandardNumber(value)
	case TypeBoolean:
		return convertToStandardBoolean(value)
	case TypeObject:
		return convertToStandardObject(value)
	case TypeArray:
		return convertToStandardArray(value)
	default:
		return nil, fmt.Errorf(ErrUnknownSchemaType, schemaType)
	}
}

// convertToStandardString converts a value to the standard string type
func convertToStandardString(value any) (StandardString, error) {
	if str, ok := value.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf(ErrExpectedType, TypeString, value)
}

// convertToStandardInteger converts a value to the standard integer type
func convertToStandardInteger(value any) (StandardInteger, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		if float32(int64(v)) != v {
			return 0, errors.New(ErrExpectedIntegerGotFloat)
		}
		return int64(v), nil
	case float64:
		if float64(int64(v)) != v {
			return 0, errors.New(ErrExpectedIntegerGotFloat)
		}
		return int64(v), nil
	case json.Number:
		intVal, err := v.Int64()
		if err != nil {
			// Try as float to check if it's a whole number
			floatVal, floatErr := v.Float64()
			if floatErr != nil {
				return 0, err
			}
			if floatVal != float64(int64(floatVal)) {
				return 0, errors.New(ErrExpectedIntegerGotFloat)
			}
			return int64(floatVal), nil
		}
		return intVal, nil
	default:
		return 0, fmt.Errorf(ErrExpectedType, TypeInteger, value)
	}
}

// convertToStandardNumber converts a value to the standard number type
func convertToStandardNumber(value any) (StandardNumber, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case json.Number:
		return v.Float64()
	default:
		return 0, fmt.Errorf(ErrExpectedType, TypeNumber, value)
	}
}

// convertToStandardBoolean converts a value to the standard boolean type
func convertToStandardBoolean(value any) (StandardBoolean, error) {
	if b, ok := value.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf(ErrExpectedType, TypeBoolean, value)
}

// convertToStandardObject converts a value to the standard object type
func convertToStandardObject(value any) (StandardObject, error) {
	if obj, ok := value.(map[string]any); ok {
		return obj, nil
	}
	return nil, fmt.Errorf(ErrExpectedType, TypeObject, value)
}

// convertToStandardArray converts a value to the standard array type
func convertToStandardArray(value any) (StandardArray, error) {
	if arr, ok := value.([]any); ok {
		return arr, nil
	}
	return nil, fmt.Errorf(ErrExpectedType, TypeArray, value)
}
