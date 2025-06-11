package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertyDefinition_Validation(t *testing.T) {
	tests := []struct {
		name        string
		propDef     PropertyDefinition
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid string property",
			propDef: PropertyDefinition{
				Type:  TypeString,
				Label: "Name",
			},
			expectError: false,
		},
		{
			name: "valid integer property with validators",
			propDef: PropertyDefinition{
				Type: TypeInteger,
				Validators: []ValidatorDefinition{
					{Type: ValidatorMin, Value: 1},
					{Type: ValidatorMax, Value: 100},
				},
			},
			expectError: false,
		},
		{
			name: "invalid type",
			propDef: PropertyDefinition{
				Type: "invalid_type",
			},
			expectError: true,
			errorMsg:    "Type",
		},
		{
			name: "missing type",
			propDef: PropertyDefinition{
				Label: "Test",
			},
			expectError: true,
			errorMsg:    "Type",
		},
		{
			name: "invalid validator type",
			propDef: PropertyDefinition{
				Type: TypeString,
				Validators: []ValidatorDefinition{
					{Type: "invalid_validator", Value: "test"},
				},
			},
			expectError: true,
			errorMsg:    "Type",
		},
		{
			name: "validator missing value",
			propDef: PropertyDefinition{
				Type: TypeString,
				Validators: []ValidatorDefinition{
					{Type: ValidatorMinLength},
				},
			},
			expectError: true,
			errorMsg:    "Value",
		},
		{
			name: "valid object with nested properties",
			propDef: PropertyDefinition{
				Type: TypeObject,
				Properties: map[string]PropertyDefinition{
					"name": {Type: TypeString},
					"age":  {Type: TypeInteger},
				},
			},
			expectError: false,
		},
		{
			name: "invalid nested property",
			propDef: PropertyDefinition{
				Type: TypeObject,
				Properties: map[string]PropertyDefinition{
					"invalid": {Type: "bad_type"},
				},
			},
			expectError: true,
			errorMsg:    "Type",
		},
		{
			name: "valid array with items",
			propDef: PropertyDefinition{
				Type: TypeArray,
				Items: &PropertyDefinition{
					Type: TypeString,
				},
			},
			expectError: false,
		},
		{
			name: "invalid array items",
			propDef: PropertyDefinition{
				Type: TypeArray,
				Items: &PropertyDefinition{
					Type: "invalid_type",
				},
			},
			expectError: true,
			errorMsg:    "Type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := CustomSchema{
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
		validator   ValidatorDefinition
		expectError bool
	}{
		{
			name: "valid minLength validator",
			validator: ValidatorDefinition{
				Type:  ValidatorMinLength,
				Value: 5,
			},
			expectError: false,
		},
		{
			name: "valid enum validator",
			validator: ValidatorDefinition{
				Type:  ValidatorEnum,
				Value: []string{"option1", "option2"},
			},
			expectError: false,
		},
		{
			name: "invalid validator type",
			validator: ValidatorDefinition{
				Type:  "invalid",
				Value: 5,
			},
			expectError: true,
		},
		{
			name: "missing type",
			validator: ValidatorDefinition{
				Value: 5,
			},
			expectError: true,
		},
		{
			name: "missing value",
			validator: ValidatorDefinition{
				Type: ValidatorMinLength,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			propDef := PropertyDefinition{
				Type:       TypeString,
				Validators: []ValidatorDefinition{tt.validator},
			}

			schema := CustomSchema{
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
	schema := CustomSchema{
		"name": PropertyDefinition{
			Type:     TypeString,
			Label:    "Full Name",
			Required: true,
			Validators: []ValidatorDefinition{
				{Type: ValidatorMinLength, Value: 2},
				{Type: ValidatorMaxLength, Value: 50},
			},
		},
		"age": PropertyDefinition{
			Type:    TypeInteger,
			Default: 0,
			Validators: []ValidatorDefinition{
				{Type: ValidatorMin, Value: 0},
				{Type: ValidatorMax, Value: 120},
			},
		},
		"config": PropertyDefinition{
			Type: TypeObject,
			Properties: map[string]PropertyDefinition{
				"theme": {Type: TypeString},
				"notifications": {
					Type:    TypeBoolean,
					Default: true,
				},
			},
		},
		"tags": PropertyDefinition{
			Type: TypeArray,
			Items: &PropertyDefinition{
				Type: TypeString,
			},
		},
	}

	// Test marshaling
	jsonData, err := schema.MarshalJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test unmarshaling
	var unmarshaled CustomSchema
	err = unmarshaled.UnmarshalJSON(jsonData)
	require.NoError(t, err)

	// Verify the unmarshaled schema
	assert.Equal(t, TypeString, unmarshaled["name"].Type)
	assert.Equal(t, "Full Name", unmarshaled["name"].Label)
	assert.True(t, unmarshaled["name"].Required)
	assert.Len(t, unmarshaled["name"].Validators, 2)

	assert.Equal(t, TypeInteger, unmarshaled["age"].Type)
	assert.Equal(t, float64(0), unmarshaled["age"].Default) // JSON unmarshaling converts numbers to float64

	assert.Equal(t, TypeObject, unmarshaled["config"].Type)
	assert.Len(t, unmarshaled["config"].Properties, 2)
	assert.Equal(t, TypeString, unmarshaled["config"].Properties["theme"].Type)

	assert.Equal(t, TypeArray, unmarshaled["tags"].Type)
	assert.NotNil(t, unmarshaled["tags"].Items)
	assert.Equal(t, TypeString, unmarshaled["tags"].Items.Type)
}

func TestCustomSchema_DatabaseSerialization(t *testing.T) {
	schema := CustomSchema{
		"test": PropertyDefinition{
			Type:  TypeString,
			Label: "Test Field",
		},
	}

	// Test Value() method (for database storage)
	value, err := schema.Value()
	require.NoError(t, err)
	assert.NotNil(t, value)

	// Test Scan() method (for database retrieval)
	var scanned CustomSchema
	err = scanned.Scan(value)
	require.NoError(t, err)

	assert.Equal(t, TypeString, scanned["test"].Type)
	assert.Equal(t, "Test Field", scanned["test"].Label)
}

func TestCustomSchema_GormDataType(t *testing.T) {
	schema := CustomSchema{}
	dataType := schema.GormDataType()
	assert.Equal(t, "jsonb", dataType)
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      ValidationError
		expected string
	}{
		{
			name: "error with path",
			err: ValidationError{
				Path:    "user.name",
				Message: "field is required",
			},
			expected: "user.name: field is required",
		},
		{
			name: "error without path",
			err: ValidationError{
				Message: "validation failed",
			},
			expected: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
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
				"type": TypeObject,
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
				"type": TypeArray,
				"items": map[string]any{
					"type": 123, // Invalid type
				},
			},
			expectError: true,
		},
		{
			name: "valid complex structure",
			input: map[string]any{
				"type":     TypeObject,
				"label":    "Complex Object",
				"required": true,
				"default":  map[string]any{"key": "value"},
				"properties": map[string]any{
					"nested": map[string]any{
						"type": TypeString,
						"validators": []any{
							map[string]any{
								"type":  ValidatorMinLength,
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
