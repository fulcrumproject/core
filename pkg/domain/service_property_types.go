package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// ServicePropertySchema represents the root schema structure
type ServicePropertySchema map[string]ServicePropertyDefinition

// Scan implements the sql.Scanner interface
func (cs *ServicePropertySchema) Scan(value any) error {
	if value == nil {
		*cs = make(ServicePropertySchema)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal CustomSchema value: %v", value)
	}

	return cs.UnmarshalJSON(bytes)
}

// Value implements the driver.Valuer interface
func (cs ServicePropertySchema) Value() (driver.Value, error) {
	if cs == nil {
		return nil, nil
	}
	return json.Marshal(cs)
}

// GormDataType returns the GORM data type for CustomSchema
func (cs ServicePropertySchema) GormDataType() string {
	return "jsonb"
}

// MarshalJSON implements custom properties.JSON marshaling for CustomSchema
func (cs ServicePropertySchema) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]ServicePropertyDefinition(cs))
}

// UnmarshalJSON implements custom properties.JSON unmarshaling for CustomSchema
func (cs *ServicePropertySchema) UnmarshalJSON(data []byte) error {
	var rawSchema map[string]any
	if err := json.Unmarshal(data, &rawSchema); err != nil {
		return err
	}

	*cs = make(ServicePropertySchema)
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
func (cs ServicePropertySchema) Validate() error {
	validate := validator.New()

	for propName, propDef := range cs {
		if propName == "" {
			return fmt.Errorf("property names cannot be empty")
		}
		if err := validate.Struct(propDef); err != nil {
			return fmt.Errorf("property %s: %w", propName, err)
		}

		// Validate property-specific rules
		if err := validatePropertyDefinition(propName, propDef); err != nil {
			return err
		}
	}
	return nil
}

// validatePropertyDefinition validates property-specific rules including source and updatability
func validatePropertyDefinition(propName string, propDef ServicePropertyDefinition) error {
	// Validate source field
	if propDef.Source != "" {
		if propDef.Source != "input" && propDef.Source != "agent" {
			return fmt.Errorf("property %s: source must be 'input' or 'agent', got '%s'", propName, propDef.Source)
		}
	}

	// Validate updatable field
	if propDef.Updatable != "" {
		if propDef.Updatable != "always" && propDef.Updatable != "never" && propDef.Updatable != "statuses" {
			return fmt.Errorf("property %s: updatable must be 'always', 'never', or 'statuses', got '%s'", propName, propDef.Updatable)
		}

		// If updatable is "statuses", updatableIn must be provided and not empty
		if propDef.Updatable == "statuses" {
			if len(propDef.UpdatableIn) == 0 {
				return fmt.Errorf("property %s: updatableIn must be provided and not empty when updatable is 'statuses'", propName)
			}
		}
	}

	// Recursively validate nested properties
	for nestedPropName, nestedPropDef := range propDef.Properties {
		if err := validatePropertyDefinition(fmt.Sprintf("%s.%s", propName, nestedPropName), nestedPropDef); err != nil {
			return err
		}
	}

	// Validate items for arrays
	if propDef.Items != nil {
		if err := validatePropertyDefinition(fmt.Sprintf("%s[]", propName), *propDef.Items); err != nil {
			return err
		}
	}

	return nil
}

// ServicePropertyDefinition defines a single property in the schema
type ServicePropertyDefinition struct {
	Type       string                               `json:"type" validate:"required,oneof=string integer number boolean object array serviceReference json"`
	Label      string                               `json:"label,omitempty"`
	Required   bool                                 `json:"required,omitempty"`
	Default    any                                  `json:"default,omitempty"`
	Validators []ServicePropertyValidatorDefinition `json:"validators,omitempty" validate:"dive"`
	Properties map[string]ServicePropertyDefinition `json:"properties,omitempty" validate:"dive"`
	Items      *ServicePropertyDefinition           `json:"items,omitempty"`

	// Property source and updatability control
	Source      string   `json:"source,omitempty"`      // "input", "agent"
	Updatable   string   `json:"updatable,omitempty"`   // "always", "never", "statuses"
	UpdatableIn []string `json:"updatableIn,omitempty"` // For "statuses" mode
}

// ServicePropertyValidatorDefinition defines a validation rule
type ServicePropertyValidatorDefinition struct {
	Type  string `json:"type" validate:"required,oneof=minLength maxLength pattern enum min max minItems maxItems uniqueItems sameOrigin serviceOption servicePool"`
	Value any    `json:"value" validate:"required"`
}

// parsePropertyDefinition is a helper function to parse a property definition recursively
func parsePropertyDefinition(propDefMap map[string]any) (ServicePropertyDefinition, error) {
	propDef := ServicePropertyDefinition{}

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
					validator := ServicePropertyValidatorDefinition{}
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
			propDef.Properties = make(map[string]ServicePropertyDefinition)
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

	// Parse source
	if sourceVal, exists := propDefMap["source"]; exists {
		if sourceStr, ok := sourceVal.(string); ok {
			propDef.Source = sourceStr
		}
	}

	// Parse updatable
	if updatableVal, exists := propDefMap["updatable"]; exists {
		if updatableStr, ok := updatableVal.(string); ok {
			propDef.Updatable = updatableStr
		}
	}

	// Parse updatableIn
	if updatableInVal, exists := propDefMap["updatableIn"]; exists {
		if updatableInSlice, ok := updatableInVal.([]any); ok {
			for _, item := range updatableInSlice {
				if itemStr, ok := item.(string); ok {
					propDef.UpdatableIn = append(propDef.UpdatableIn, itemStr)
				}
			}
		}
	}

	return propDef, nil
}
