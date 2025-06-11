package schema

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// CustomSchema represents the root schema structure
type CustomSchema map[string]PropertyDefinition

// Scan implements the sql.Scanner interface
func (cs *CustomSchema) Scan(value interface{}) error {
	if value == nil {
		*cs = make(CustomSchema)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal CustomSchema value: %v", value)
	}

	return cs.UnmarshalJSON(bytes)
}

// Value implements the driver.Valuer interface
func (cs CustomSchema) Value() (driver.Value, error) {
	if cs == nil {
		return nil, nil
	}
	return json.Marshal(cs)
}

// GormDataType returns the GORM data type for CustomSchema
func (cs CustomSchema) GormDataType() string {
	return "jsonb"
}

// MarshalJSON implements custom JSON marshaling for CustomSchema
func (cs CustomSchema) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]PropertyDefinition(cs))
}

// UnmarshalJSON implements custom JSON unmarshaling for CustomSchema
func (cs *CustomSchema) UnmarshalJSON(data []byte) error {
	var rawSchema map[string]any
	if err := json.Unmarshal(data, &rawSchema); err != nil {
		return err
	}

	*cs = make(CustomSchema)
	for propName, propDefRaw := range rawSchema {
		if propDefMap, ok := propDefRaw.(map[string]any); ok {
			propDef, err := parsePropertyDefinition(propDefMap)
			if err != nil {
				return fmt.Errorf("error parsing property %s: %w", propName, err)
			}
			(*cs)[propName] = propDef
		}
	}

	return nil
}

// Validate checks if the CustomSchema is valid using go-playground/validator
func (cs CustomSchema) Validate() error {
	validate := validator.New()

	for propName, propDef := range cs {
		if propName == "" {
			return fmt.Errorf("property names cannot be empty")
		}
		if err := validate.Struct(propDef); err != nil {
			return fmt.Errorf("property %s: %w", propName, err)
		}
	}
	return nil
}

// PropertyDefinition defines a single property in the schema
type PropertyDefinition struct {
	Type       string                        `json:"type" validate:"required,oneof=string integer number boolean object array"`
	Label      string                        `json:"label,omitempty"`
	Required   bool                          `json:"required,omitempty"`
	Default    any                           `json:"default,omitempty"`
	Validators []ValidatorDefinition         `json:"validators,omitempty" validate:"dive"`
	Properties map[string]PropertyDefinition `json:"properties,omitempty" validate:"dive"`
	Items      *PropertyDefinition           `json:"items,omitempty"`
}

// ValidatorDefinition defines a validation rule
type ValidatorDefinition struct {
	Type  string `json:"type" validate:"required,oneof=minLength maxLength pattern enum min max minItems maxItems uniqueItems"`
	Value any    `json:"value" validate:"required"`
}

// ValidationError represents a validation error with path context
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.Path == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// Numeric represents types that can be converted to numbers
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// parsePropertyDefinition is a helper function to parse a property definition recursively
func parsePropertyDefinition(propDefMap map[string]any) (PropertyDefinition, error) {
	propDef := PropertyDefinition{}

	// Parse type
	if typeVal, exists := propDefMap["type"]; exists {
		if typeStr, ok := typeVal.(string); ok {
			propDef.Type = typeStr
		} else {
			return propDef, fmt.Errorf("type must be a string")
		}
	}

	// Parse label
	if labelVal, exists := propDefMap["label"]; exists {
		if labelStr, ok := labelVal.(string); ok {
			propDef.Label = labelStr
		}
	}

	// Parse required
	if requiredVal, exists := propDefMap["required"]; exists {
		if requiredBool, ok := requiredVal.(bool); ok {
			propDef.Required = requiredBool
		}
	}

	// Parse default
	if defaultVal, exists := propDefMap["default"]; exists {
		propDef.Default = defaultVal
	}

	// Parse validators
	if validatorsVal, exists := propDefMap["validators"]; exists {
		if validatorsSlice, ok := validatorsVal.([]any); ok {
			for _, validatorRaw := range validatorsSlice {
				if validatorMap, ok := validatorRaw.(map[string]any); ok {
					validator := ValidatorDefinition{}
					if typeVal, exists := validatorMap["type"]; exists {
						if typeStr, ok := typeVal.(string); ok {
							validator.Type = typeStr
						}
					}
					if valueVal, exists := validatorMap["value"]; exists {
						validator.Value = valueVal
					}
					propDef.Validators = append(propDef.Validators, validator)
				}
			}
		}
	}

	// Parse nested properties for objects (recursive)
	if propertiesVal, exists := propDefMap["properties"]; exists {
		if propertiesMap, ok := propertiesVal.(map[string]any); ok {
			propDef.Properties = make(map[string]PropertyDefinition)
			for nestedPropName, nestedPropDefRaw := range propertiesMap {
				if nestedPropDefMap, ok := nestedPropDefRaw.(map[string]any); ok {
					nestedPropDef, err := parsePropertyDefinition(nestedPropDefMap)
					if err != nil {
						return propDef, fmt.Errorf("nested property %s: %w", nestedPropName, err)
					}
					propDef.Properties[nestedPropName] = nestedPropDef
				}
			}
		}
	}

	// Parse items for arrays (recursive)
	if itemsVal, exists := propDefMap["items"]; exists {
		if itemsMap, ok := itemsVal.(map[string]any); ok {
			itemsDef, err := parsePropertyDefinition(itemsMap)
			if err != nil {
				return propDef, fmt.Errorf("array items: %w", err)
			}
			propDef.Items = &itemsDef
		}
	}

	return propDef, nil
}
