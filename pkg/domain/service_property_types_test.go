package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertyDefinition_Validation(t *testing.T) {
	tests := []struct {
		name        string
		propDef     ServicePropertyDefinition
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid string property",
			propDef: ServicePropertyDefinition{
				Type:  SchemaTypeString,
				Label: "Name",
			},
			expectError: false,
		},
		{
			name: "valid integer property with validators",
			propDef: ServicePropertyDefinition{
				Type: SchemaTypeInteger,
				Validators: []ServicePropertyValidatorDefinition{
					{Type: SchemaValidatorMin, Value: 1},
					{Type: SchemaValidatorMax, Value: 100},
				},
			},
			expectError: false,
		},
		{
			name: "invalid type",
			propDef: ServicePropertyDefinition{
				Type: "invalid_type",
			},
			expectError: true,
			errorMsg:    "Type",
		},
		{
			name: "missing type",
			propDef: ServicePropertyDefinition{
				Label: "Test",
			},
			expectError: true,
			errorMsg:    "Type",
		},
		{
			name: "invalid validator type",
			propDef: ServicePropertyDefinition{
				Type: SchemaTypeString,
				Validators: []ServicePropertyValidatorDefinition{
					{Type: "invalid_validator", Value: "test"},
				},
			},
			expectError: true,
			errorMsg:    "Type",
		},
		{
			name: "validator missing value",
			propDef: ServicePropertyDefinition{
				Type: SchemaTypeString,
				Validators: []ServicePropertyValidatorDefinition{
					{Type: SchemaValidatorMinLength},
				},
			},
			expectError: true,
			errorMsg:    "Value",
		},
		{
			name: "valid object with nested properties",
			propDef: ServicePropertyDefinition{
				Type: SchemaTypeObject,
				Properties: map[string]ServicePropertyDefinition{
					"name": {Type: SchemaTypeString},
					"age":  {Type: SchemaTypeInteger},
				},
			},
			expectError: false,
		},
		{
			name: "invalid nested property",
			propDef: ServicePropertyDefinition{
				Type: SchemaTypeObject,
				Properties: map[string]ServicePropertyDefinition{
					"invalid": {Type: "bad_type"},
				},
			},
			expectError: true,
			errorMsg:    "Type",
		},
		{
			name: "valid array with items",
			propDef: ServicePropertyDefinition{
				Type: SchemaTypeArray,
				Items: &ServicePropertyDefinition{
					Type: SchemaTypeString,
				},
			},
			expectError: false,
		},
		{
			name: "invalid array items",
			propDef: ServicePropertyDefinition{
				Type: SchemaTypeArray,
				Items: &ServicePropertyDefinition{
					Type: "invalid_type",
				},
			},
			expectError: true,
			errorMsg:    "Type",
		},
		{
			name: "valid json property",
			propDef: ServicePropertyDefinition{
				Type:  SchemaTypeJSON,
				Label: "Configuration",
			},
			expectError: false,
		},
		{
			name: "valid json property with serviceOption validator",
			propDef: ServicePropertyDefinition{
				Type: SchemaTypeJSON,
				Validators: []ServicePropertyValidatorDefinition{
					{Type: SchemaValidatorServiceOption, Value: "option-type-id"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := ServicePropertySchema{
				"test_prop": tt.propDef,
			}

			err := schema.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatorDefinition_Validation(t *testing.T) {
	tests := []struct {
		name        string
		validator   ServicePropertyValidatorDefinition
		expectError bool
	}{
		{
			name: "valid minLength validator",
			validator: ServicePropertyValidatorDefinition{
				Type:  SchemaValidatorMinLength,
				Value: 5,
			},
			expectError: false,
		},
		{
			name: "valid enum validator",
			validator: ServicePropertyValidatorDefinition{
				Type:  SchemaValidatorEnum,
				Value: []string{"option1", "option2"},
			},
			expectError: false,
		},
		{
			name: "invalid validator type",
			validator: ServicePropertyValidatorDefinition{
				Type:  "invalid",
				Value: 5,
			},
			expectError: true,
		},
		{
			name: "missing type",
			validator: ServicePropertyValidatorDefinition{
				Value: 5,
			},
			expectError: true,
		},
		{
			name: "missing value",
			validator: ServicePropertyValidatorDefinition{
				Type: SchemaValidatorMinLength,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propDef := ServicePropertyDefinition{
				Type:       SchemaTypeString,
				Validators: []ServicePropertyValidatorDefinition{tt.validator},
			}

			schema := ServicePropertySchema{
				"test_prop": propDef,
			}

			err := schema.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCustomSchema_JSONSerialization(t *testing.T) {
	schema := ServicePropertySchema{
		"name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Label:    "Full Name",
			Required: true,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMinLength, Value: 2},
				{Type: SchemaValidatorMaxLength, Value: 50},
			},
		},
		"age": ServicePropertyDefinition{
			Type:    SchemaTypeInteger,
			Default: 0,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 0},
				{Type: SchemaValidatorMax, Value: 120},
			},
		},
		"config": ServicePropertyDefinition{
			Type: SchemaTypeObject,
			Properties: map[string]ServicePropertyDefinition{
				"theme": {Type: SchemaTypeString},
				"notifications": {
					Type:    SchemaTypeBoolean,
					Default: true,
				},
			},
		},
		"tags": ServicePropertyDefinition{
			Type: SchemaTypeArray,
			Items: &ServicePropertyDefinition{
				Type: SchemaTypeString,
			},
		},
	}

	// Test marshaling
	jsonData, err := schema.MarshalJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test unmarshaling
	var unmarshaled ServicePropertySchema
	err = unmarshaled.UnmarshalJSON(jsonData)
	require.NoError(t, err)

	// Verify the unmarshaled schema
	assert.Equal(t, SchemaTypeString, unmarshaled["name"].Type)
	assert.Equal(t, "Full Name", unmarshaled["name"].Label)
	assert.True(t, unmarshaled["name"].Required)
	assert.Len(t, unmarshaled["name"].Validators, 2)

	assert.Equal(t, SchemaTypeInteger, unmarshaled["age"].Type)
	assert.Equal(t, float64(0), unmarshaled["age"].Default) // properties.JSON unmarshaling converts numbers to float64

	assert.Equal(t, SchemaTypeObject, unmarshaled["config"].Type)
	assert.Len(t, unmarshaled["config"].Properties, 2)
	assert.Equal(t, SchemaTypeString, unmarshaled["config"].Properties["theme"].Type)

	assert.Equal(t, SchemaTypeArray, unmarshaled["tags"].Type)
	assert.NotNil(t, unmarshaled["tags"].Items)
	assert.Equal(t, SchemaTypeString, unmarshaled["tags"].Items.Type)
}

func TestCustomSchema_DatabaseSerialization(t *testing.T) {
	schema := ServicePropertySchema{
		"test": ServicePropertyDefinition{
			Type:  SchemaTypeString,
			Label: "Test Field",
		},
	}

	// Test Value() method (for database storage)
	value, err := schema.Value()
	require.NoError(t, err)
	assert.NotNil(t, value)

	// Test Scan() method (for database retrieval)
	var scanned ServicePropertySchema
	err = scanned.Scan(value)
	require.NoError(t, err)

	assert.Equal(t, SchemaTypeString, scanned["test"].Type)
	assert.Equal(t, "Test Field", scanned["test"].Label)
}

func TestCustomSchema_GormDataType(t *testing.T) {
	schema := ServicePropertySchema{}
	dataType := schema.GormDataType()
	assert.Equal(t, "jsonb", dataType)
}

func TestValidationErrorDetail_Fields(t *testing.T) {
	tests := []struct {
		name            string
		err             ValidationErrorDetail
		expectedPath    string
		expectedMessage string
	}{
		{
			name: "error with path",
			err: ValidationErrorDetail{
				Path:    "user.name",
				Message: "field is required",
			},
			expectedPath:    "user.name",
			expectedMessage: "field is required",
		},
		{
			name: "error without path",
			err: ValidationErrorDetail{
				Message: "validation failed",
			},
			expectedPath:    "",
			expectedMessage: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedPath, tt.err.Path)
			assert.Equal(t, tt.expectedMessage, tt.err.Message)
		})
	}
}

func TestParsePropertyDefinition_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]any
		expectError bool
	}{
		{
			name: "type is not string",
			input: map[string]any{
				"type": 123,
			},
			expectError: true,
		},
		{
			name: "nested property parsing error",
			input: map[string]any{
				"type": SchemaTypeObject,
				"properties": map[string]any{
					"invalid": map[string]any{
						"type": 123, // Invalid type
					},
				},
			},
			expectError: true,
		},
		{
			name: "array items parsing error",
			input: map[string]any{
				"type": SchemaTypeArray,
				"items": map[string]any{
					"type": 123, // Invalid type
				},
			},
			expectError: true,
		},
		{
			name: "valid complex structure",
			input: map[string]any{
				"type":     SchemaTypeObject,
				"label":    "Complex Object",
				"required": true,
				"default":  map[string]any{"key": "value"},
				"properties": map[string]any{
					"nested": map[string]any{
						"type": SchemaTypeString,
						"validators": []any{
							map[string]any{
								"type":  SchemaValidatorMinLength,
								"value": 5,
							},
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePropertyDefinition(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
