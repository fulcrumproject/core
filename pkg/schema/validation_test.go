package schema

import (
	"reflect"
	"testing"
)

func TestValidate_RequiredFields(t *testing.T) {
	schema := CustomSchema{
		"name": PropertyDefinition{
			Type:     TypeString,
			Required: true,
		},
		"age": PropertyDefinition{
			Type:     TypeInteger,
			Required: false,
		},
	}

	// Test missing required field
	data := map[string]any{
		"age": 25,
	}

	errors := Validate(data, schema)
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}
	if errors[0].Path != "name" {
		t.Errorf("Expected error path 'name', got '%s'", errors[0].Path)
	}
	if errors[0].Message != ErrRequiredFieldMissing {
		t.Errorf("Expected error message '%s', got '%s'", ErrRequiredFieldMissing, errors[0].Message)
	}

	// Test with required field present
	data["name"] = "John"
	errors = Validate(data, schema)
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

func TestValidate_TypeValidation(t *testing.T) {
	schema := CustomSchema{
		"name":   PropertyDefinition{Type: TypeString},
		"age":    PropertyDefinition{Type: TypeInteger},
		"score":  PropertyDefinition{Type: TypeNumber},
		"active": PropertyDefinition{Type: TypeBoolean},
		"config": PropertyDefinition{Type: TypeObject},
		"tags":   PropertyDefinition{Type: TypeArray},
	}

	// Test valid types
	validData := map[string]any{
		"name":   "John",
		"age":    25,
		"score":  95.5,
		"active": true,
		"config": map[string]any{"key": "value"},
		"tags":   []any{"tag1", "tag2"},
	}

	errors := Validate(validData, schema)
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors for valid data, got %d: %v", len(errors), errors)
	}

	// Test invalid types
	invalidData := map[string]any{
		"name":   123,         // should be string
		"age":    "twenty",    // should be integer
		"score":  "high",      // should be number
		"active": "yes",       // should be boolean
		"config": "invalid",   // should be object
		"tags":   "tag1,tag2", // should be array
	}

	errors = Validate(invalidData, schema)
	if len(errors) != 6 {
		t.Errorf("Expected 6 errors for invalid data, got %d", len(errors))
	}
}

func TestValidate_StringValidators(t *testing.T) {
	schema := CustomSchema{
		"username": PropertyDefinition{
			Type: TypeString,
			Validators: []ValidatorDefinition{
				{Type: ValidatorMinLength, Value: 3},
				{Type: ValidatorMaxLength, Value: 20},
				{Type: ValidatorPattern, Value: "^[a-zA-Z0-9_]+$"},
			},
		},
		"status": PropertyDefinition{
			Type: TypeString,
			Validators: []ValidatorDefinition{
				{Type: ValidatorEnum, Value: []any{"active", "inactive", "pending"}},
			},
		},
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectError bool
		errorCount  int
	}{
		{
			name: "valid string data",
			data: map[string]any{
				"username": "john_doe123",
				"status":   "active",
			},
			expectError: false,
		},
		{
			name: "username too short",
			data: map[string]any{
				"username": "jo",
				"status":   "active",
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "username too long",
			data: map[string]any{
				"username": "this_username_is_way_too_long_for_validation",
				"status":   "active",
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "username invalid pattern",
			data: map[string]any{
				"username": "john-doe!",
				"status":   "active",
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "invalid enum value",
			data: map[string]any{
				"username": "john_doe",
				"status":   "unknown",
			},
			expectError: true,
			errorCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := Validate(tt.data, schema)
			if tt.expectError {
				if len(errors) != tt.errorCount {
					t.Errorf("Expected %d errors, got %d: %v", tt.errorCount, len(errors), errors)
				}
			} else {
				if len(errors) != 0 {
					t.Errorf("Expected no errors, got %d: %v", len(errors), errors)
				}
			}
		})
	}
}

func TestValidate_NumericValidators(t *testing.T) {
	schema := CustomSchema{
		"cpu": PropertyDefinition{
			Type: TypeInteger,
			Validators: []ValidatorDefinition{
				{Type: ValidatorMin, Value: 1},
				{Type: ValidatorMax, Value: 32},
			},
		},
		"memory": PropertyDefinition{
			Type: TypeNumber,
			Validators: []ValidatorDefinition{
				{Type: ValidatorMin, Value: 0.5},
				{Type: ValidatorMax, Value: 64.0},
			},
		},
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectError bool
		errorCount  int
	}{
		{
			name: "valid numeric data",
			data: map[string]any{
				"cpu":    4,
				"memory": 8.5,
			},
			expectError: false,
		},
		{
			name: "cpu below minimum",
			data: map[string]any{
				"cpu":    0,
				"memory": 8.0,
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "cpu above maximum",
			data: map[string]any{
				"cpu":    64,
				"memory": 8.0,
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "memory below minimum",
			data: map[string]any{
				"cpu":    4,
				"memory": 0.1,
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "memory above maximum",
			data: map[string]any{
				"cpu":    4,
				"memory": 128.0,
			},
			expectError: true,
			errorCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := Validate(tt.data, schema)
			if tt.expectError {
				if len(errors) != tt.errorCount {
					t.Errorf("Expected %d errors, got %d: %v", tt.errorCount, len(errors), errors)
				}
			} else {
				if len(errors) != 0 {
					t.Errorf("Expected no errors, got %d: %v", len(errors), errors)
				}
			}
		})
	}
}

func TestValidate_ArrayValidators(t *testing.T) {
	schema := CustomSchema{
		"ports": PropertyDefinition{
			Type: TypeArray,
			Validators: []ValidatorDefinition{
				{Type: ValidatorMinItems, Value: 1},
				{Type: ValidatorMaxItems, Value: 5},
				{Type: ValidatorUniqueItems, Value: true},
			},
			Items: &PropertyDefinition{
				Type: TypeInteger,
				Validators: []ValidatorDefinition{
					{Type: ValidatorMin, Value: 1},
					{Type: ValidatorMax, Value: 65535},
				},
			},
		},
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectError bool
		errorCount  int
	}{
		{
			name: "valid array",
			data: map[string]any{
				"ports": []any{80, 443, 8080},
			},
			expectError: false,
		},
		{
			name: "empty array (below minimum)",
			data: map[string]any{
				"ports": []any{},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "too many items",
			data: map[string]any{
				"ports": []any{80, 443, 8080, 9000, 3000, 5000},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "duplicate items",
			data: map[string]any{
				"ports": []any{80, 443, 80},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "invalid item values",
			data: map[string]any{
				"ports": []any{0, 80000},
			},
			expectError: true,
			errorCount:  2, // Two invalid port numbers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := Validate(tt.data, schema)
			if tt.expectError {
				if len(errors) != tt.errorCount {
					t.Errorf("Expected %d errors, got %d: %v", tt.errorCount, len(errors), errors)
				}
			} else {
				if len(errors) != 0 {
					t.Errorf("Expected no errors, got %d: %v", len(errors), errors)
				}
			}
		})
	}
}

func TestValidate_NestedObjects(t *testing.T) {
	schema := CustomSchema{
		"metadata": PropertyDefinition{
			Type: TypeObject,
			Properties: map[string]PropertyDefinition{
				"owner": {
					Type:     TypeString,
					Required: true,
				},
				"version": {
					Type: TypeNumber,
					Validators: []ValidatorDefinition{
						{Type: ValidatorMin, Value: 1.0},
					},
				},
				"tags": {
					Type: TypeArray,
					Items: &PropertyDefinition{
						Type: TypeString,
						Validators: []ValidatorDefinition{
							{Type: ValidatorMinLength, Value: 1},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectError bool
		errorCount  int
	}{
		{
			name: "valid nested object",
			data: map[string]any{
				"metadata": map[string]any{
					"owner":   "john",
					"version": 2.1,
					"tags":    []any{"prod", "web"},
				},
			},
			expectError: false,
		},
		{
			name: "missing required nested field",
			data: map[string]any{
				"metadata": map[string]any{
					"version": 2.1,
				},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "invalid nested array item",
			data: map[string]any{
				"metadata": map[string]any{
					"owner": "john",
					"tags":  []any{"prod", ""},
				},
			},
			expectError: true,
			errorCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := Validate(tt.data, schema)
			if tt.expectError {
				if len(errors) != tt.errorCount {
					t.Errorf("Expected %d errors, got %d: %v", tt.errorCount, len(errors), errors)
				}
			} else {
				if len(errors) != 0 {
					t.Errorf("Expected no errors, got %d: %v", len(errors), errors)
				}
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	schema := CustomSchema{
		"name": PropertyDefinition{
			Type:     TypeString,
			Required: true,
		},
		"port": PropertyDefinition{
			Type:    TypeInteger,
			Default: 8080,
		},
		"enabled": PropertyDefinition{
			Type:    TypeBoolean,
			Default: true,
		},
		"config": PropertyDefinition{
			Type: TypeObject,
			Properties: map[string]PropertyDefinition{
				"timeout": {
					Type:    TypeInteger,
					Default: 30,
				},
				"retries": {
					Type:    TypeInteger,
					Default: 3,
				},
			},
		},
	}

	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "apply missing defaults",
			input: map[string]any{
				"name": "test-service",
			},
			expected: map[string]any{
				"name":    "test-service",
				"port":    8080,
				"enabled": true,
			},
		},
		{
			name: "preserve existing values",
			input: map[string]any{
				"name":    "test-service",
				"port":    9000,
				"enabled": false,
			},
			expected: map[string]any{
				"name":    "test-service",
				"port":    9000,
				"enabled": false,
			},
		},
		{
			name: "apply nested defaults",
			input: map[string]any{
				"name": "test-service",
				"config": map[string]any{
					"timeout": 60,
				},
			},
			expected: map[string]any{
				"name":    "test-service",
				"port":    8080,
				"enabled": true,
				"config": map[string]any{
					"timeout": 60,
					"retries": 3,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyDefaults(tt.input, schema)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestValidateWithDefaults(t *testing.T) {
	schema := CustomSchema{
		"name": PropertyDefinition{
			Type:     TypeString,
			Required: true,
		},
		"port": PropertyDefinition{
			Type:    TypeInteger,
			Default: 8080,
			Validators: []ValidatorDefinition{
				{Type: ValidatorMin, Value: 1},
				{Type: ValidatorMax, Value: 65535},
			},
		},
	}

	// Test with valid data and defaults
	input := map[string]any{
		"name": "test-service",
	}

	result, errors := ValidateWithDefaults(input, schema)
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errors), errors)
	}

	expected := map[string]any{
		"name": "test-service",
		"port": 8080,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}

	// Test with invalid default (this shouldn't happen in practice, but tests edge case)
	schemaWithInvalidDefault := CustomSchema{
		"name": PropertyDefinition{
			Type:     TypeString,
			Required: true,
		},
		"port": PropertyDefinition{
			Type:    TypeInteger,
			Default: 0, // Invalid according to validator
			Validators: []ValidatorDefinition{
				{Type: ValidatorMin, Value: 1},
			},
		},
	}

	_, errors = ValidateWithDefaults(input, schemaWithInvalidDefault)
	if len(errors) == 0 {
		t.Error("Expected validation error for invalid default value")
	}
}

func TestValidate_UnknownProperties(t *testing.T) {
	schema := CustomSchema{
		"name": PropertyDefinition{
			Type: TypeString,
		},
	}

	data := map[string]any{
		"name":         "test",
		"unknown_prop": "value",
	}

	errors := Validate(data, schema)
	if len(errors) != 1 {
		t.Errorf("Expected 1 error for unknown property, got %d", len(errors))
	}

	if errors[0].Path != "unknown_prop" {
		t.Errorf("Expected error path 'unknown_prop', got '%s'", errors[0].Path)
	}

	if errors[0].Message != ErrUnknownProperty {
		t.Errorf("Expected error message '%s', got '%s'", ErrUnknownProperty, errors[0].Message)
	}
}

func TestValidate_ComplexExample(t *testing.T) {
	// This test uses the example schema from the feature specification
	schema := CustomSchema{
		"cpu": PropertyDefinition{
			Type:     TypeInteger,
			Label:    "CPU Cores",
			Required: true,
			Validators: []ValidatorDefinition{
				{Type: ValidatorMin, Value: 1},
			},
		},
		"image_name": PropertyDefinition{
			Type:     TypeString,
			Label:    "Container Image",
			Required: true,
			Validators: []ValidatorDefinition{
				{Type: ValidatorMinLength, Value: 5},
				{Type: ValidatorPattern, Value: "^[a-z0-9-]+$"},
			},
		},
		"environment": PropertyDefinition{
			Type:  TypeString,
			Label: "Environment",
			Validators: []ValidatorDefinition{
				{Type: ValidatorEnum, Value: []any{"development", "staging", "production"}},
			},
		},
		"enable_feature_x": PropertyDefinition{
			Type:  TypeBoolean,
			Label: "Enable Feature X",
		},
		"metadata": PropertyDefinition{
			Type:  TypeObject,
			Label: "Service Metadata",
			Properties: map[string]PropertyDefinition{
				"owner": {
					Type:     TypeString,
					Label:    "Owner",
					Required: true,
				},
				"version": {
					Type:  TypeNumber,
					Label: "Version",
				},
			},
		},
		"ports": PropertyDefinition{
			Type:  TypeArray,
			Label: "Port Configuration",
			Items: &PropertyDefinition{
				Type: TypeInteger,
				Validators: []ValidatorDefinition{
					{Type: ValidatorMin, Value: 1},
					{Type: ValidatorMax, Value: 65535},
				},
			},
			Validators: []ValidatorDefinition{
				{Type: ValidatorMinItems, Value: 1},
			},
		},
	}

	// Valid data
	validData := map[string]any{
		"cpu":              4,
		"image_name":       "nginx-alpine",
		"environment":      "production",
		"enable_feature_x": true,
		"metadata": map[string]any{
			"owner":   "devops-team",
			"version": 1.2,
		},
		"ports": []any{80, 443},
	}

	errors := Validate(validData, schema)
	if len(errors) != 0 {
		t.Errorf("Expected no errors for valid complex data, got %d: %v", len(errors), errors)
	}

	// Invalid data with multiple errors
	invalidData := map[string]any{
		"cpu":              0,         // Below minimum
		"image_name":       "NGINX",   // Invalid pattern (uppercase)
		"environment":      "testing", // Not in enum
		"enable_feature_x": "yes",     // Wrong type
		"metadata": map[string]any{
			"version": 1.2,
			// Missing required "owner"
		},
		"ports": []any{}, // Below minimum items
	}

	errors = Validate(invalidData, schema)
	if len(errors) < 5 {
		t.Errorf("Expected at least 5 errors for invalid complex data, got %d: %v", len(errors), errors)
	}
}
