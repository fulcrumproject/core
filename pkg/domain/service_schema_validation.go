package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"

	"github.com/fulcrumproject/core/pkg/properties"
)

// Validator type constants
const (
	SchemaValidatorMinLength   = "minLength"
	SchemaValidatorMaxLength   = "maxLength"
	SchemaValidatorPattern     = "pattern"
	SchemaValidatorEnum        = "enum"
	SchemaValidatorMin         = "min"
	SchemaValidatorMax         = "max"
	SchemaValidatorMinItems    = "minItems"
	SchemaValidatorMaxItems    = "maxItems"
	SchemaValidatorUniqueItems = "uniqueItems"
	SchemaValidatorSameOrigin  = "sameOrigin"
	SchemaValidatorServiceType = "serviceType"
)

// Schema type constants
const (
	SchemaTypeString    = "string"
	SchemaTypeInteger   = "integer"
	SchemaTypeNumber    = "number"
	SchemaTypeBoolean   = "boolean"
	SchemaTypeObject    = "object"
	SchemaTypeArray     = "array"
	SchemaTypeReference = "reference"
)

// Standard target types for each schema type
// These are the canonical types we convert to for consistent comparisons
type (
	SchemaStandardString    = string
	SchemaStandardInteger   = int64
	SchemaStandardNumber    = float64
	SchemaStandardBoolean   = bool
	SchemaStandardObject    = map[string]any
	SchemaStandardArray     = []any
	SchemaStandardReference = properties.UUID
)

// Error message templates
const (
	ErrSchemaUnknownProperty           = "unknown property"
	ErrSchemaRequiredFieldMissing      = "required field is missing"
	ErrSchemaUnknownValidatorType      = "unknown validator type: %s"
	ErrSchemaUnknownSchemaType         = "unknown type: %s"
	ErrSchemaExpectedType              = "expected %s, got %T"
	ErrSchemaExpectedIntegerGotFloat   = "expected integer, got float with decimal part"
	ErrSchemaValidatorOnlyForType      = "%s validator can only be applied to %s"
	ErrSchemaValidatorValueMustBeType  = "%s validator value must be %s"
	ErrSchemaStringLengthLessThanMin   = "string length %d is less than minimum %d"
	ErrSchemaStringLengthExceedsMax    = "string length %d exceeds maximum %d"
	ErrSchemaInvalidRegexPattern       = "invalid regex pattern: %s"
	ErrSchemaStringDoesNotMatchPattern = "string does not match pattern %s"
	ErrSchemaValueNotInEnum            = "value is not in allowed enum values: %s"
	ErrSchemaValueLessThanMin          = "value %v is less than minimum %v"
	ErrSchemaValueExceedsMax           = "value %v exceeds maximum %v"
	ErrSchemaArrayLengthLessThanMin    = "array length %d is less than minimum %d"
	ErrSchemaArrayLengthExceedsMax     = "array length %d exceeds maximum %d"
	ErrSchemaArrayContainsDuplicates   = "array contains duplicate items"
	ErrSchemaFloatHasDecimalPart       = "float value has decimal part"
	ErrSchemaCannotConvertType         = "cannot convert %T to %T"

	// Service reference error messages
	ErrSchemaInvalidServiceID                  = "invalid service ID format"
	ErrSchemaServiceNotFound                   = "referenced service does not exist"
	ErrSchemaServiceNotSameConsumer            = "referenced service must belong to the same consumer"
	ErrSchemaServiceNotSameGroup               = "referenced service must belong to the same service group"
	ErrSchemaInvalidSameOriginValue            = "invalid sameOrigin value: must be 'consumer' or 'group'"
	ErrSchemaReferenceValidationMissingContext = "service reference validation requires validation context"
	ErrSchemaServiceWrongType                  = "referenced service is not of the allowed service type"
	ErrSchemaInvalidServiceTypeValidatorValue  = "serviceType validator value must be a string or array of strings"
)

// ServicePropertyValidationCtx provides the context for validating service properties
type ServicePropertyValidationCtx struct {
	Context    context.Context
	Store      Store
	Schema     ServiceSchema
	GroupID    properties.UUID
	Properties map[string]any
}

// applyServicePropertiesDefaults applies default values to data based on the schema
func applyServicePropertiesDefaults(data map[string]any, schema ServiceSchema) map[string]any {
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
			if propDef.Type == SchemaTypeObject && propDef.Properties != nil {
				if objValue, ok := result[propName].(map[string]any); ok {
					result[propName] = applyServicePropertiesDefaults(objValue, propDef.Properties)
				}
			}
		}
	}

	return result
}

// validateServiceProperties validates data against the provided schema
func validateServiceProperties(ctx *ServicePropertyValidationCtx) ([]ValidationErrorDetail, error) {
	var errors []ValidationErrorDetail

	// Check required fields
	for propName, propDef := range ctx.Schema {
		if propDef.Required {
			if _, exists := ctx.Properties[propName]; !exists {
				errors = append(errors, ValidationErrorDetail{
					Path:    propName,
					Message: ErrSchemaRequiredFieldMissing,
				})
			} else if ctx.Properties[propName] == nil {
				errors = append(errors, ValidationErrorDetail{
					Path:    propName,
					Message: ErrSchemaRequiredFieldMissing,
				})
			}
		}
	}

	// Validate each property with context
	for propName, value := range ctx.Properties {
		propDef, exists := ctx.Schema[propName]
		if !exists {
			errors = append(errors, ValidationErrorDetail{
				Path:    propName,
				Message: ErrSchemaUnknownProperty,
			})
			continue
		}

		propErrors, err := validateServiceProperty(ctx, propName, value, propDef)
		if err != nil {
			return nil, err
		}
		errors = append(errors, propErrors...)
	}

	return errors, nil
}

// validateServiceProperty validates a single property value against its definition
func validateServiceProperty(ctx *ServicePropertyValidationCtx, path string, value any, propDef ServicePropertyDefinition) ([]ValidationErrorDetail, error) {
	var errors []ValidationErrorDetail

	// Type validation - get the converted standard value
	standardValue, err := convertToServicePropertyStandardType(ctx, value, propDef.Type)
	if err != nil {
		errors = append(errors, ValidationErrorDetail{
			Path:    path,
			Message: err.Error(),
		})
		return errors, nil // Don't continue if type is wrong
	}

	// Validator rules - use the already converted standard value
	for _, validator := range propDef.Validators {
		if err := applyServicePropertyValidator(ctx, standardValue, validator, propDef.Type); err != nil {
			errors = append(errors, ValidationErrorDetail{
				Path:    path,
				Message: err.Error(),
			})
		}
	}

	// Nested validation for objects
	if propDef.Type == SchemaTypeObject && propDef.Properties != nil {
		if objValue, ok := standardValue.(map[string]any); ok {
			// Create nested context
			nestedCtx := &ServicePropertyValidationCtx{
				Context:    ctx.Context,
				Store:      ctx.Store,
				Schema:     propDef.Properties,
				GroupID:    ctx.GroupID,
				Properties: objValue,
			}
			nestedErrors, err := validateServiceProperties(nestedCtx)
			if err != nil {
				return nil, err
			}
			for _, nestedErr := range nestedErrors {
				errors = append(errors, ValidationErrorDetail{
					Path:    path + "." + nestedErr.Path,
					Message: nestedErr.Message,
				})
			}
		}
	}

	// Array item validation
	if propDef.Type == SchemaTypeArray && propDef.Items != nil {
		if arrValue, ok := standardValue.([]any); ok {
			for i, item := range arrValue {
				itemPath := fmt.Sprintf("%s[%d]", path, i)
				// Validate each array item using the item definition
				itemErrors, err := validateServiceProperty(ctx, itemPath, item, *propDef.Items)
				if err != nil {
					return nil, err
				}
				errors = append(errors, itemErrors...)
			}
		}
	}

	return errors, nil
}

// applyServicePropertyValidator applies a specific validator to a value
func applyServicePropertyValidator(ctx *ServicePropertyValidationCtx, value any, validator ServicePropertyValidatorDefinition, propertyType string) error {
	switch validator.Type {
	case SchemaValidatorMinLength:
		return validateServicePropertyMinLength(ctx, value, validator.Value)
	case SchemaValidatorMaxLength:
		return validateServicePropertyMaxLength(ctx, value, validator.Value)
	case SchemaValidatorPattern:
		return validateServicePropertyPattern(ctx, value, validator.Value)
	case SchemaValidatorEnum:
		return validateServicePropertyEnum(ctx, value, validator.Value, propertyType)
	case SchemaValidatorMin:
		return validateServicePropertyMin(ctx, value, validator.Value)
	case SchemaValidatorMax:
		return validateServicePropertyMax(ctx, value, validator.Value)
	case SchemaValidatorMinItems:
		return validateServicePropertyMinItems(ctx, value, validator.Value)
	case SchemaValidatorMaxItems:
		return validateServicePropertyMaxItems(ctx, value, validator.Value)
	case SchemaValidatorUniqueItems:
		return validateServicePropertyUniqueItems(ctx, value, validator.Value)
	case SchemaValidatorSameOrigin:
		return validateServicePropertySameOrigin(ctx, value, validator.Value)
	case SchemaValidatorServiceType:
		return validateServicePropertyServiceType(ctx, value, validator.Value)
	default:
		return fmt.Errorf(ErrSchemaUnknownValidatorType, validator.Type)
	}
}

// validateServicePropertyMinLength is a specific validator for minimum string length
func validateServicePropertyMinLength(_ *ServicePropertyValidationCtx, standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardString by validateType
	standardStr, ok := standardValue.(SchemaStandardString)
	if !ok {
		return fmt.Errorf(ErrSchemaValidatorOnlyForType, SchemaValidatorMinLength, "a string")
	}

	// Convert validator value to standard integer
	standardLimit, err := convertToServicePropertyStandardInteger(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrSchemaValidatorValueMustBeType, SchemaValidatorMinLength, "an integer")
	}

	if int64(len(standardStr)) < standardLimit {
		return fmt.Errorf(ErrSchemaStringLengthLessThanMin, len(standardStr), standardLimit)
	}

	return nil
}

// validateServicePropertyMaxLength is a specific validator for maximum string length
func validateServicePropertyMaxLength(_ *ServicePropertyValidationCtx, standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardString by validateType
	standardStr, ok := standardValue.(SchemaStandardString)
	if !ok {
		return fmt.Errorf(ErrSchemaValidatorOnlyForType, SchemaValidatorMaxLength, "a string")
	}

	// Convert validator value to standard integer
	standardLimit, err := convertToServicePropertyStandardInteger(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrSchemaValidatorValueMustBeType, SchemaValidatorMaxLength, "an integer")
	}

	if int64(len(standardStr)) > standardLimit {
		return fmt.Errorf(ErrSchemaStringLengthExceedsMax, len(standardStr), standardLimit)
	}

	return nil
}

// validateServicePropertyPattern checks if a string matches a regex pattern
func validateServicePropertyPattern(_ *ServicePropertyValidationCtx, standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardString by validateType
	standardStr, ok := standardValue.(SchemaStandardString)
	if !ok {
		return fmt.Errorf(ErrSchemaValidatorOnlyForType, SchemaValidatorPattern, "a string")
	}

	// Convert validator value to standard string type
	pattern, err := convertToServicePropertyStandardString(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrSchemaValidatorValueMustBeType, SchemaValidatorPattern, "a string")
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf(ErrSchemaInvalidRegexPattern, pattern)
	}

	if !regex.MatchString(standardStr) {
		return fmt.Errorf(ErrSchemaStringDoesNotMatchPattern, pattern)
	}

	return nil
}

// validateServicePropertyEnum checks if a value is in the allowed enum values
func validateServicePropertyEnum(ctx *ServicePropertyValidationCtx, standardValue any, validatorValue any, propertyType string) error {
	// Convert validator value to standard array type
	enumArray, err := convertToServicePropertyStandardArray(validatorValue)
	if err != nil {
		return fmt.Errorf("enum validator value must be an array")
	}

	// Convert each enum value to the standard type and compare with the already-converted standardValue
	for _, enumValue := range enumArray {
		standardEnumValue, err := convertToServicePropertyStandardType(ctx, enumValue, propertyType)
		if err != nil {
			continue // Skip invalid enum values
		}

		if reflect.DeepEqual(standardValue, standardEnumValue) {
			return nil
		}
	}

	return fmt.Errorf(ErrSchemaValueNotInEnum, formatEnumValues(enumArray))
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

// validateServicePropertyMin is a specific validator for minimum numeric value
func validateServicePropertyMin(_ *ServicePropertyValidationCtx, standardValue any, validatorValue any) error {
	// Convert standardValue to float64 for comparison (handles both int64 and float64)
	var standardNum float64
	switch v := standardValue.(type) {
	case SchemaStandardInteger:
		standardNum = float64(v)
	case SchemaStandardNumber:
		standardNum = v
	default:
		return fmt.Errorf(ErrSchemaValidatorOnlyForType, SchemaValidatorMin, "numbers")
	}

	// Convert validator value to standard number type
	standardLimit, err := convertToServicePropertyStandardNumber(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrSchemaValidatorValueMustBeType, SchemaValidatorMin, "a number")
	}

	if standardNum < standardLimit {
		return fmt.Errorf(ErrSchemaValueLessThanMin, standardNum, standardLimit)
	}

	return nil
}

// validateServicePropertyMax is a specific validator for maximum numeric value
func validateServicePropertyMax(_ *ServicePropertyValidationCtx, standardValue any, validatorValue any) error {
	// Convert standardValue to float64 for comparison (handles both int64 and float64)
	var standardNum float64
	switch v := standardValue.(type) {
	case SchemaStandardInteger:
		standardNum = float64(v)
	case SchemaStandardNumber:
		standardNum = v
	default:
		return fmt.Errorf(ErrSchemaValidatorOnlyForType, SchemaValidatorMax, "numbers")
	}

	// Convert validator value to standard number type
	standardLimit, err := convertToServicePropertyStandardNumber(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrSchemaValidatorValueMustBeType, SchemaValidatorMax, "a number")
	}

	if standardNum > standardLimit {
		return fmt.Errorf(ErrSchemaValueExceedsMax, standardNum, standardLimit)
	}

	return nil
}

// validateServicePropertyMinItems is a specific validator for minimum array length
func validateServicePropertyMinItems(_ *ServicePropertyValidationCtx, standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardArray by validateType
	standardArr, ok := standardValue.(SchemaStandardArray)
	if !ok {
		return fmt.Errorf(ErrSchemaValidatorOnlyForType, SchemaValidatorMinItems, "an array")
	}

	// Convert validator value to standard integer
	standardLimit, err := convertToServicePropertyStandardInteger(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrSchemaValidatorValueMustBeType, SchemaValidatorMinItems, "an integer")
	}

	if int64(len(standardArr)) < standardLimit {
		return fmt.Errorf(ErrSchemaArrayLengthLessThanMin, len(standardArr), standardLimit)
	}

	return nil
}

// validateServicePropertyMaxItems is a specific validator for maximum array length
func validateServicePropertyMaxItems(_ *ServicePropertyValidationCtx, standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardArray by validateType
	standardArr, ok := standardValue.(SchemaStandardArray)
	if !ok {
		return fmt.Errorf(ErrSchemaValidatorOnlyForType, SchemaValidatorMaxItems, "an array")
	}

	// Convert validator value to standard integer
	standardLimit, err := convertToServicePropertyStandardInteger(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrSchemaValidatorValueMustBeType, SchemaValidatorMaxItems, "an integer")
	}

	if int64(len(standardArr)) > standardLimit {
		return fmt.Errorf(ErrSchemaArrayLengthExceedsMax, len(standardArr), standardLimit)
	}

	return nil
}

// validateServicePropertyUniqueItems checks if an array contains unique items
func validateServicePropertyUniqueItems(_ *ServicePropertyValidationCtx, standardValue any, validatorValue any) error {
	// standardValue is already converted to StandardArray by validateType
	standardArr, ok := standardValue.(SchemaStandardArray)
	if !ok {
		return fmt.Errorf(ErrSchemaValidatorOnlyForType, SchemaValidatorUniqueItems, "an array")
	}

	// Convert validator value to standard boolean
	unique, err := convertToServicePropertyStandardBoolean(validatorValue)
	if err != nil {
		return fmt.Errorf(ErrSchemaValidatorValueMustBeType, SchemaValidatorUniqueItems, "a boolean")
	}

	if !unique {
		return nil // uniqueItems: false means no validation needed
	}

	seen := make(map[string]bool)
	for _, item := range standardArr {
		key := fmt.Sprintf("%v", item)
		if seen[key] {
			return errors.New(ErrSchemaArrayContainsDuplicates)
		}
		seen[key] = true
	}

	return nil
}

// convertToServicePropertyStandardType converts a value to the standard type for a given schema type
func convertToServicePropertyStandardType(ctx *ServicePropertyValidationCtx, value any, schemaType string) (any, error) {
	if value == nil {
		return nil, nil
	}

	switch schemaType {
	case SchemaTypeString:
		return convertToServicePropertyStandardString(value)
	case SchemaTypeInteger:
		return convertToServicePropertyStandardInteger(value)
	case SchemaTypeNumber:
		return convertToServicePropertyStandardNumber(value)
	case SchemaTypeBoolean:
		return convertToServicePropertyStandardBoolean(value)
	case SchemaTypeObject:
		return convertToServicePropertyStandardObject(value)
	case SchemaTypeArray:
		return convertToServicePropertyStandardArray(value)
	case SchemaTypeReference:
		return convertToServicePropertyStandardReference(ctx, value)
	default:
		return nil, fmt.Errorf(ErrSchemaUnknownSchemaType, schemaType)
	}
}

// convertToServicePropertyStandardString converts a value to the standard string type
func convertToServicePropertyStandardString(value any) (SchemaStandardString, error) {
	if str, ok := value.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf(ErrSchemaExpectedType, SchemaTypeString, value)
}

// convertToServicePropertyStandardInteger converts a value to the standard integer type
func convertToServicePropertyStandardInteger(value any) (SchemaStandardInteger, error) {
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
			return 0, errors.New(ErrSchemaExpectedIntegerGotFloat)
		}
		return int64(v), nil
	case float64:
		if float64(int64(v)) != v {
			return 0, errors.New(ErrSchemaExpectedIntegerGotFloat)
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
				return 0, errors.New(ErrSchemaExpectedIntegerGotFloat)
			}
			return int64(floatVal), nil
		}
		return intVal, nil
	default:
		return 0, fmt.Errorf(ErrSchemaExpectedType, SchemaTypeInteger, value)
	}
}

// convertToServicePropertyStandardNumber converts a value to the standard number type
func convertToServicePropertyStandardNumber(value any) (SchemaStandardNumber, error) {
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
		return 0, fmt.Errorf(ErrSchemaExpectedType, SchemaTypeNumber, value)
	}
}

// convertToServicePropertyStandardBoolean converts a value to the standard boolean type
func convertToServicePropertyStandardBoolean(value any) (SchemaStandardBoolean, error) {
	if b, ok := value.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf(ErrSchemaExpectedType, SchemaTypeBoolean, value)
}

// convertToServicePropertyStandardObject converts a value to the standard object type
func convertToServicePropertyStandardObject(value any) (SchemaStandardObject, error) {
	if obj, ok := value.(map[string]any); ok {
		return obj, nil
	}
	return nil, fmt.Errorf(ErrSchemaExpectedType, SchemaTypeObject, value)
}

// convertToServicePropertyStandardArray converts a value to the standard array type
func convertToServicePropertyStandardArray(value any) (SchemaStandardArray, error) {
	if arr, ok := value.([]any); ok {
		return arr, nil
	}
	return nil, fmt.Errorf(ErrSchemaExpectedType, SchemaTypeArray, value)
}

// convertToServicePropertyStandardReference validates service reference and converts to UUID
func convertToServicePropertyStandardReference(ctx *ServicePropertyValidationCtx, value any) (SchemaStandardReference, error) {
	// 1. Convert to string and validate UUID format
	str, ok := value.(string)
	if !ok {
		return properties.UUID{}, fmt.Errorf(ErrSchemaExpectedType, SchemaTypeReference, value)
	}

	serviceUUID, err := properties.ParseUUID(str)
	if err != nil {
		return properties.UUID{}, errors.New(ErrSchemaInvalidServiceID)
	}

	// 2. Check service existence using standard Get method
	_, err = ctx.Store.ServiceRepo().Get(ctx.Context, serviceUUID)
	if err != nil {
		return properties.UUID{}, errors.New(ErrSchemaServiceNotFound)
	}

	// 3. Return UUID if valid
	return serviceUUID, nil
}

// validateServicePropertySameOrigin validates that referenced service belongs to same origin (consumer or group)
func validateServicePropertySameOrigin(ctx *ServicePropertyValidationCtx, standardValue any, validatorValue any) error {
	serviceUUID, ok := standardValue.(SchemaStandardReference)
	if !ok {
		return fmt.Errorf("expected service reference UUID")
	}

	// Get the origin type from validator value
	originType, err := convertToServicePropertyStandardString(validatorValue)
	if err != nil {
		return fmt.Errorf("sameOrigin validator value must be a string")
	}

	// Get the referenced service
	referencedService, err := ctx.Store.ServiceRepo().Get(ctx.Context, serviceUUID)
	if err != nil {
		return fmt.Errorf("failed to validate service reference: %w", err)
	}

	// Check constraint based on origin type
	switch originType {
	case "consumer":
		// Get the consumer from the current group
		currentGroup, err := ctx.Store.ServiceGroupRepo().Get(ctx.Context, ctx.GroupID)
		if err != nil {
			return fmt.Errorf("failed to get current service group: %w", err)
		}
		if referencedService.ConsumerID != currentGroup.ConsumerID {
			return errors.New(ErrSchemaServiceNotSameConsumer)
		}
	case "group":
		if referencedService.GroupID != ctx.GroupID {
			return errors.New(ErrSchemaServiceNotSameGroup)
		}
	default:
		return fmt.Errorf("invalid sameOrigin value: %s (must be 'consumer' or 'group')", originType)
	}

	return nil
}

// validateServicePropertyServiceType validates that referenced service is of allowed service type(s)
func validateServicePropertyServiceType(ctx *ServicePropertyValidationCtx, standardValue any, validatorValue any) error {
	serviceUUID, ok := standardValue.(SchemaStandardReference)
	if !ok {
		return fmt.Errorf("expected service reference UUID")
	}

	// Get the referenced service
	referencedService, err := ctx.Store.ServiceRepo().Get(ctx.Context, serviceUUID)
	if err != nil {
		return fmt.Errorf("failed to validate service reference: %w", err)
	}

	// Get the referenced service's service type
	serviceType, err := ctx.Store.ServiceTypeRepo().Get(ctx.Context, referencedService.ServiceTypeID)
	if err != nil {
		return fmt.Errorf("failed to get service type for referenced service: %w", err)
	}

	// Parse the validator value - can be a single string or array of strings
	var allowedTypes []string
	switch v := validatorValue.(type) {
	case string:
		allowedTypes = []string{v}
	case []any:
		for _, item := range v {
			if typeStr, ok := item.(string); ok {
				allowedTypes = append(allowedTypes, typeStr)
			} else {
				return errors.New(ErrSchemaInvalidServiceTypeValidatorValue)
			}
		}
	default:
		return errors.New(ErrSchemaInvalidServiceTypeValidatorValue)
	}

	// Check if the referenced service's type name is in the allowed types
	if !slices.Contains(allowedTypes, serviceType.Name) {
		return errors.New(ErrSchemaServiceWrongType)
	}

	return nil
}
