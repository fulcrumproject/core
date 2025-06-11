package schema

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
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

	// Type validation
	if err := validateType(value, propDef.Type); err != nil {
		errors = append(errors, ValidationError{
			Path:    path,
			Message: err.Error(),
		})
		return errors // Don't continue if type is wrong
	}

	// Validator rules
	for _, validator := range propDef.Validators {
		if err := applyValidator(value, validator); err != nil {
			errors = append(errors, ValidationError{
				Path:    path,
				Message: err.Error(),
			})
		}
	}

	// Nested validation for objects
	if propDef.Type == TypeObject && propDef.Properties != nil {
		if objValue, ok := value.(map[string]any); ok {
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
		if arrValue, ok := value.([]any); ok {
			for i, item := range arrValue {
				itemPath := fmt.Sprintf("%s[%d]", path, i)
				itemErrors := validateProperty(itemPath, item, *propDef.Items)
				errors = append(errors, itemErrors...)
			}
		}
	}

	return errors
}

// validateType checks if the value matches the expected type
func validateType(value any, expectedType string) error {
	if value == nil {
		return nil // Allow null values for optional fields
	}

	switch expectedType {
	case TypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf(ErrExpectedType, TypeString, value)
		}
	case TypeInteger:
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// Valid integer types
		case float64:
			// JSON numbers are parsed as float64, check if it's a whole number
			if v != float64(int64(v)) {
				return errors.New(ErrExpectedIntegerGotFloat)
			}
		default:
			return fmt.Errorf(ErrExpectedType, TypeInteger, value)
		}
	case TypeNumber:
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			// Valid number types
		default:
			return fmt.Errorf(ErrExpectedType, TypeNumber, value)
		}
	case TypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf(ErrExpectedType, TypeBoolean, value)
		}
	case TypeObject:
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf(ErrExpectedType, TypeObject, value)
		}
	case TypeArray:
		if _, ok := value.([]any); !ok {
			return fmt.Errorf(ErrExpectedType, TypeArray, value)
		}
	default:
		return fmt.Errorf(ErrUnknownSchemaType, expectedType)
	}

	return nil
}

// applyValidator applies a specific validator to a value
func applyValidator(value any, validator ValidatorDefinition) error {
	switch validator.Type {
	case ValidatorMinLength:
		return validateMinLength(value, validator.Value)
	case ValidatorMaxLength:
		return validateMaxLength(value, validator.Value)
	case ValidatorPattern:
		return validatePattern(value, validator.Value)
	case ValidatorEnum:
		return validateEnum(value, validator.Value)
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

// Generic string length validator
func validateStringLength[T ~int](value any, validatorValue any, validatorName string, compare func(int, T) bool, errorMsg string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf(ErrValidatorOnlyForType, validatorName, "strings")
	}

	length, err := extractValidatorValue[T](validatorValue, validatorName)
	if err != nil {
		// Fallback for backward compatibility
		intLength, err := getIntValue(validatorValue)
		if err != nil {
			return fmt.Errorf(ErrValidatorValueMustBeType, validatorName, "an integer")
		}
		length = T(intLength)
	}

	if compare(len(str), length) {
		return fmt.Errorf(errorMsg, len(str), length)
	}

	return nil
}

// validateMinLength is a specific validator for minimum string length
func validateMinLength(value any, validatorValue any) error {
	return validateStringLength(value, validatorValue, ValidatorMinLength,
		func(actual int, min int) bool { return actual < min },
		ErrStringLengthLessThanMin)
}

// validateMaxLength is a specific validator for maximum string length
func validateMaxLength(value any, validatorValue any) error {
	return validateStringLength(value, validatorValue, ValidatorMaxLength,
		func(actual int, max int) bool { return actual > max },
		ErrStringLengthExceedsMax)
}

// validatePattern checks if a string matches a regex pattern
func validatePattern(value any, validatorValue any) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf(ErrValidatorOnlyForType, ValidatorPattern, "strings")
	}

	pattern, err := extractValidatorValue[string](validatorValue, ValidatorPattern)
	if err != nil {
		return err
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf(ErrInvalidRegexPattern, pattern)
	}

	if !regex.MatchString(str) {
		return fmt.Errorf(ErrStringDoesNotMatchPattern, pattern)
	}

	return nil
}

// validateEnum checks if a value is in the allowed enum values
func validateEnum(value any, validatorValue any) error {
	enumValues, err := extractValidatorValue[[]any](validatorValue, ValidatorEnum)
	if err != nil {
		return err
	}

	for _, enumValue := range enumValues {
		if reflect.DeepEqual(value, enumValue) {
			return nil
		}
	}

	return fmt.Errorf(ErrValueNotInEnum, formatEnumValues(enumValues))
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

// Generic range validator for numeric types
func validateNumericRange[T Numeric](value any, validatorValue any, validatorName string, compare func(T, T) bool, errorMsg string) error {
	numValue, err := convertToType[T](value)
	if err != nil {
		return fmt.Errorf(ErrValidatorOnlyForType, validatorName, "numbers")
	}

	limitValue, err := convertToType[T](validatorValue)
	if err != nil {
		return fmt.Errorf(ErrValidatorValueMustBeType, validatorName, "a number")
	}

	if compare(numValue, limitValue) {
		return fmt.Errorf(errorMsg, numValue, limitValue)
	}

	return nil
}

// validateMin is a specific validator for minimum numeric value
func validateMin(value any, validatorValue any) error {
	return validateNumericRange(value, validatorValue, ValidatorMin,
		func(actual, min float64) bool { return actual < min },
		ErrValueLessThanMin)
}

// validateMax is a specific validator for maximum numeric value
func validateMax(value any, validatorValue any) error {
	return validateNumericRange(value, validatorValue, ValidatorMax,
		func(actual, max float64) bool { return actual > max },
		ErrValueExceedsMax)
}

// Generic array length validator
func validateArrayLength[T ~int](value any, validatorValue any, validatorName string, compare func(int, T) bool, errorMsg string) error {
	arr, ok := value.([]any)
	if !ok {
		return fmt.Errorf(ErrValidatorOnlyForType, validatorName, "arrays")
	}

	limit, err := extractValidatorValue[T](validatorValue, validatorName)
	if err != nil {
		// Fallback for backward compatibility
		intLimit, err := getIntValue(validatorValue)
		if err != nil {
			return fmt.Errorf(ErrValidatorValueMustBeType, validatorName, "an integer")
		}
		limit = T(intLimit)
	}

	if compare(len(arr), limit) {
		return fmt.Errorf(errorMsg, len(arr), limit)
	}

	return nil
}

// validateMinItems is a specific validator for minimum array length
func validateMinItems(value any, validatorValue any) error {
	return validateArrayLength(value, validatorValue, ValidatorMinItems,
		func(actual int, min int) bool { return actual < min },
		ErrArrayLengthLessThanMin)
}

// validateMaxItems is a specific validator for maximum array length
func validateMaxItems(value any, validatorValue any) error {
	return validateArrayLength(value, validatorValue, ValidatorMaxItems,
		func(actual int, max int) bool { return actual > max },
		ErrArrayLengthExceedsMax)
}

// validateUniqueItems checks if an array contains unique items
func validateUniqueItems(value any, validatorValue any) error {
	arr, ok := value.([]any)
	if !ok {
		return fmt.Errorf(ErrValidatorOnlyForType, ValidatorUniqueItems, "arrays")
	}

	unique, err := extractValidatorValue[bool](validatorValue, ValidatorUniqueItems)
	if err != nil {
		return err
	}

	if !unique {
		return nil // uniqueItems: false means no validation needed
	}

	seen := make(map[string]bool)
	for _, item := range arr {
		key := fmt.Sprintf("%v", item)
		if seen[key] {
			return errors.New(ErrArrayContainsDuplicates)
		}
		seen[key] = true
	}

	return nil
}

// Helper functions

// convertToType attempts to convert a value to the specified numeric type
func convertToType[T Numeric](value any) (T, error) {
	var zero T

	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return T(reflect.ValueOf(v).Convert(reflect.TypeOf(zero)).Interface().(T)), nil
	case string:
		// Try to determine if T is an integer type by checking the zero value
		switch any(zero).(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// For integer types, parse as int
			parsed, err := strconv.Atoi(v)
			if err != nil {
				return zero, err
			}
			return T(parsed), nil
		default:
			// For float types, parse as float
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return zero, err
			}
			return T(parsed), nil
		}
	default:
		return zero, fmt.Errorf(ErrCannotConvertType, value, zero)
	}
}

// getIntValue converts a value to an integer, ensuring it has no decimal part
func getIntValue(value any) (int, error) {
	result, err := convertToType[int](value)
	if err != nil {
		return 0, err
	}

	// Special handling for float64 from JSON to ensure it's a whole number
	if v, ok := value.(float64); ok {
		if v != float64(int(v)) {
			return 0, errors.New(ErrFloatHasDecimalPart)
		}
	}

	return result, nil
}

// Generic validator value extractor
func extractValidatorValue[T any](validatorValue any, expectedType string) (T, error) {
	var zero T
	if result, ok := validatorValue.(T); ok {
		return result, nil
	}
	return zero, fmt.Errorf("%s validator value must be %T", expectedType, zero)
}
