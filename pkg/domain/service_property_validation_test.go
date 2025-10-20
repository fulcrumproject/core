package domain

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func createTestStore(t *testing.T, serviceType *ServiceType) Store {
	mockStore := NewMockStore(t)
	mockServiceTypeRepo := NewMockServiceTypeRepository(t)

	// Setup ServiceTypeRepo mock
	mockStore.EXPECT().ServiceTypeRepo().Return(mockServiceTypeRepo).Maybe()
	mockServiceTypeRepo.EXPECT().Get(mock.Anything, serviceType.ID).Return(serviceType, nil).Maybe()

	// Setup Atomic to just execute the function with the same store
	mockStore.EXPECT().Atomic(mock.Anything, mock.Anything).RunAndReturn(
		func(ctx context.Context, fn func(Store) error) error {
			return fn(mockStore)
		},
	).Maybe()

	return mockStore
}

func createTestServiceType(schema ServicePropertySchema) *ServiceType {
	uuid := uuid.New()
	id := properties.UUID(uuid)
	return &ServiceType{
		BaseEntity: BaseEntity{
			ID:        id,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:           "Test Service Type",
		PropertySchema: &schema,
	}
}

func validateServicePropertiesHelper(t *testing.T, data map[string]any, schema ServicePropertySchema) (map[string]any, []ValidationErrorDetail) {
	ctx := context.Background()
	serviceType := createTestServiceType(schema)
	store := createTestStore(t, serviceType)
	uuid := uuid.New()
	groupID := properties.UUID(uuid)

	params := &ServicePropertyValidationParams{
		ServiceTypeID: serviceType.ID,
		GroupID:       groupID,
		Properties:    data,
	}

	result, err := ValidateServiceProperties(ctx, store, params)
	if err != nil {
		if validationError, ok := err.(ValidationError); ok {
			return nil, validationError.Errors
		}
		// For other errors, return them as a single validation error
		return nil, []ValidationErrorDetail{{Message: err.Error()}}
	}
	return result, nil
}

func TestValidate_RequiredFields(t *testing.T) {
	schema := ServicePropertySchema{
		"name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Required: true,
		},
		"age": ServicePropertyDefinition{
			Type:     SchemaTypeInteger,
			Required: false,
		},
	}

	// Test missing required field
	data := map[string]any{
		"age": 25,
	}

	_, errors := validateServicePropertiesHelper(t, data, schema)
	require.Len(t, errors, 1)
	assert.Equal(t, "name", errors[0].Path)
	assert.Equal(t, ErrSchemaRequiredFieldMissing, errors[0].Message)

	// Test with required field present
	data["name"] = "John"
	result, errors := validateServicePropertiesHelper(t, data, schema)
	require.Len(t, errors, 0)
	assert.Equal(t, "John", result["name"])
	assert.Equal(t, 25, result["age"])
}

func TestValidate_TypeValidation(t *testing.T) {
	schema := ServicePropertySchema{
		"name":   ServicePropertyDefinition{Type: SchemaTypeString},
		"age":    ServicePropertyDefinition{Type: SchemaTypeInteger},
		"score":  ServicePropertyDefinition{Type: SchemaTypeNumber},
		"active": ServicePropertyDefinition{Type: SchemaTypeBoolean},
		"config": ServicePropertyDefinition{Type: SchemaTypeObject},
		"tags":   ServicePropertyDefinition{Type: SchemaTypeArray},
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

	result, errors := validateServicePropertiesHelper(t, validData, schema)
	require.Len(t, errors, 0)
	assert.Equal(t, "John", result["name"])

	// Test invalid types
	invalidData := map[string]any{
		"name":   123,         // should be string
		"age":    "twenty",    // should be integer
		"score":  "high",      // should be number
		"active": "yes",       // should be boolean
		"config": "invalid",   // should be object
		"tags":   "tag1,tag2", // should be array
	}

	_, errors = validateServicePropertiesHelper(t, invalidData, schema)
	assert.Len(t, errors, 6)
}

func TestValidate_StringValidators(t *testing.T) {
	schema := ServicePropertySchema{
		"username": ServicePropertyDefinition{
			Type: SchemaTypeString,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMinLength, Value: 3},
				{Type: SchemaValidatorMaxLength, Value: 20},
				{Type: SchemaValidatorPattern, Value: "^[a-zA-Z0-9_]+$"},
			},
		},
		"status": ServicePropertyDefinition{
			Type: SchemaTypeString,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorEnum, Value: []any{"active", "inactive", "pending"}},
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
			_, errors := validateServicePropertiesHelper(t, tt.data, schema)
			if tt.expectError {
				assert.Len(t, errors, tt.errorCount)
			} else {
				assert.Len(t, errors, 0)
			}
		})
	}
}

func TestValidate_NumericValidators(t *testing.T) {
	schema := ServicePropertySchema{
		"cpu": ServicePropertyDefinition{
			Type: SchemaTypeInteger,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 1},
				{Type: SchemaValidatorMax, Value: 32},
			},
		},
		"memory": ServicePropertyDefinition{
			Type: SchemaTypeNumber,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 0.5},
				{Type: SchemaValidatorMax, Value: 64.0},
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
			_, errors := validateServicePropertiesHelper(t, tt.data, schema)
			if tt.expectError {
				assert.Len(t, errors, tt.errorCount)
			} else {
				assert.Len(t, errors, 0)
			}
		})
	}
}

func TestValidate_ArrayValidators(t *testing.T) {
	schema := ServicePropertySchema{
		"ports": ServicePropertyDefinition{
			Type: SchemaTypeArray,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMinItems, Value: 1},
				{Type: SchemaValidatorMaxItems, Value: 5},
				{Type: SchemaValidatorUniqueItems, Value: true},
			},
			Items: &ServicePropertyDefinition{
				Type: SchemaTypeInteger,
				Validators: []ServicePropertyValidatorDefinition{
					{Type: SchemaValidatorMin, Value: 1},
					{Type: SchemaValidatorMax, Value: 65535},
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
			_, errors := validateServicePropertiesHelper(t, tt.data, schema)
			if tt.expectError {
				assert.Len(t, errors, tt.errorCount)
			} else {
				assert.Len(t, errors, 0)
			}
		})
	}
}

func TestValidate_NestedObjects(t *testing.T) {
	schema := ServicePropertySchema{
		"metadata": ServicePropertyDefinition{
			Type: SchemaTypeObject,
			Properties: map[string]ServicePropertyDefinition{
				"owner": {
					Type:     SchemaTypeString,
					Required: true,
				},
				"version": {
					Type: SchemaTypeNumber,
					Validators: []ServicePropertyValidatorDefinition{
						{Type: SchemaValidatorMin, Value: 1.0},
					},
				},
				"tags": {
					Type: SchemaTypeArray,
					Items: &ServicePropertyDefinition{
						Type: SchemaTypeString,
						Validators: []ServicePropertyValidatorDefinition{
							{Type: SchemaValidatorMinLength, Value: 1},
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
			_, errors := validateServicePropertiesHelper(t, tt.data, schema)
			if tt.expectError {
				assert.Len(t, errors, tt.errorCount)
			} else {
				assert.Len(t, errors, 0)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	schema := ServicePropertySchema{
		"name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Required: true,
		},
		"port": ServicePropertyDefinition{
			Type:    SchemaTypeInteger,
			Default: 8080,
		},
		"enabled": ServicePropertyDefinition{
			Type:    SchemaTypeBoolean,
			Default: true,
		},
		"config": ServicePropertyDefinition{
			Type: SchemaTypeObject,
			Properties: map[string]ServicePropertyDefinition{
				"timeout": {
					Type:    SchemaTypeInteger,
					Default: 30,
				},
				"retries": {
					Type:    SchemaTypeInteger,
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
			result := applyServicePropertiesDefaults(tt.input, schema)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestValidateWithDefaults(t *testing.T) {
	schema := ServicePropertySchema{
		"name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Required: true,
		},
		"port": ServicePropertyDefinition{
			Type:    SchemaTypeInteger,
			Default: 8080,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 1},
				{Type: SchemaValidatorMax, Value: 65535},
			},
		},
	}

	// Test with valid data and defaults
	input := map[string]any{
		"name": "test-service",
	}

	result, errors := validateServicePropertiesHelper(t, input, schema)
	require.Len(t, errors, 0)

	expected := map[string]any{
		"name": "test-service",
		"port": 8080,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}

	// Test with invalid default (this shouldn't happen in practice, but tests edge case)
	schemaWithInvalidDefault := ServicePropertySchema{
		"name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Required: true,
		},
		"port": ServicePropertyDefinition{
			Type:    SchemaTypeInteger,
			Default: 0, // Invalid according to validator
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 1},
			},
		},
	}

	_, errors = validateServicePropertiesHelper(t, input, schemaWithInvalidDefault)
	require.Greater(t, len(errors), 0, "Expected validation error for invalid default value")
}

func TestValidate_UnknownProperties(t *testing.T) {
	schema := ServicePropertySchema{
		"name": ServicePropertyDefinition{
			Type: SchemaTypeString,
		},
	}

	data := map[string]any{
		"name":         "test",
		"unknown_prop": "value",
	}

	_, errors := validateServicePropertiesHelper(t, data, schema)
	require.Len(t, errors, 1)
	assert.Equal(t, "unknown_prop", errors[0].Path)
	assert.Equal(t, ErrSchemaUnknownProperty, errors[0].Message)
}

func TestValidate_ComplexExample(t *testing.T) {
	// This test uses the example schema from the feature specification
	schema := ServicePropertySchema{
		"cpu": ServicePropertyDefinition{
			Type:     SchemaTypeInteger,
			Label:    "CPU Cores",
			Required: true,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 1},
			},
		},
		"image_name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Label:    "Container Image",
			Required: true,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMinLength, Value: 5},
				{Type: SchemaValidatorPattern, Value: "^[a-z0-9-]+$"},
			},
		},
		"environment": ServicePropertyDefinition{
			Type:  SchemaTypeString,
			Label: "Environment",
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorEnum, Value: []any{"development", "staging", "production"}},
			},
		},
		"enable_feature_x": ServicePropertyDefinition{
			Type:  SchemaTypeBoolean,
			Label: "Enable Feature X",
		},
		"metadata": ServicePropertyDefinition{
			Type:  SchemaTypeObject,
			Label: "Service Metadata",
			Properties: map[string]ServicePropertyDefinition{
				"owner": {
					Type:     SchemaTypeString,
					Label:    "Owner",
					Required: true,
				},
				"version": {
					Type:  SchemaTypeNumber,
					Label: "Version",
				},
			},
		},
		"ports": ServicePropertyDefinition{
			Type:  SchemaTypeArray,
			Label: "Port Configuration",
			Items: &ServicePropertyDefinition{
				Type: SchemaTypeInteger,
				Validators: []ServicePropertyValidatorDefinition{
					{Type: SchemaValidatorMin, Value: 1},
					{Type: SchemaValidatorMax, Value: 65535},
				},
			},
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMinItems, Value: 1},
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

	_, errors := validateServicePropertiesHelper(t, validData, schema)
	require.Len(t, errors, 0, "Expected no errors for valid complex data, got: %v", errors)

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

	_, errors = validateServicePropertiesHelper(t, invalidData, schema)
	require.GreaterOrEqual(t, len(errors), 5, "Expected at least 5 errors for invalid complex data, got: %v", errors)
}

// TestValidatePropertyUpdatability tests the ValidatePropertyUpdatability function
func TestValidatePropertyUpdatability(t *testing.T) {
	tests := []struct {
		name          string
		propertyName  string
		currentStatus string
		propDef       ServicePropertyDefinition
		expectErr     bool
		errMsg        string
	}{
		{
			name:          "Never mode always returns error",
			propertyName:  "hostname",
			currentStatus: "Started",
			propDef: ServicePropertyDefinition{
				Type:      "string",
				Updatable: "never",
			},
			expectErr: true,
			errMsg:    "cannot be updated (updatable: never)",
		},
		{
			name:          "Always mode always returns nil",
			propertyName:  "tags",
			currentStatus: "Started",
			propDef: ServicePropertyDefinition{
				Type:      "object",
				Updatable: "always",
			},
			expectErr: false,
		},
		{
			name:          "Default mode (empty) is always",
			propertyName:  "description",
			currentStatus: "Started",
			propDef: ServicePropertyDefinition{
				Type:      "string",
				Updatable: "",
			},
			expectErr: false,
		},
		{
			name:          "Statuses mode allows updates in allowed status",
			propertyName:  "cpu",
			currentStatus: "Stopped",
			propDef: ServicePropertyDefinition{
				Type:        "integer",
				Updatable:   "statuses",
				UpdatableIn: []string{"Stopped"},
			},
			expectErr: false,
		},
		{
			name:          "Statuses mode rejects updates in disallowed status",
			propertyName:  "cpu",
			currentStatus: "Started",
			propDef: ServicePropertyDefinition{
				Type:        "integer",
				Updatable:   "statuses",
				UpdatableIn: []string{"Stopped"},
			},
			expectErr: true,
			errMsg:    "cannot be updated in status 'Started'",
		},
		{
			name:          "Statuses mode with multiple allowed statuses - allowed",
			propertyName:  "memory",
			currentStatus: "Created",
			propDef: ServicePropertyDefinition{
				Type:        "integer",
				Updatable:   "statuses",
				UpdatableIn: []string{"Stopped", "Created"},
			},
			expectErr: false,
		},
		{
			name:          "Statuses mode with multiple allowed statuses - disallowed",
			propertyName:  "memory",
			currentStatus: "Started",
			propDef: ServicePropertyDefinition{
				Type:        "integer",
				Updatable:   "statuses",
				UpdatableIn: []string{"Stopped", "Created"},
			},
			expectErr: true,
			errMsg:    "cannot be updated in status 'Started'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePropertyUpdatability(tt.propertyName, tt.currentStatus, tt.propDef)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePropertiesForUpdate tests the ValidatePropertiesForUpdate function
func TestValidatePropertiesForUpdate(t *testing.T) {
	schema := &ServicePropertySchema{
		"hostname": ServicePropertyDefinition{
			Type:      "string",
			Source:    "input",
			Updatable: "never",
		},
		"cpu": ServicePropertyDefinition{
			Type:        "integer",
			Source:      "input",
			Updatable:   "statuses",
			UpdatableIn: []string{"Stopped"},
		},
		"tags": ServicePropertyDefinition{
			Type:      "object",
			Source:    "input",
			Updatable: "always",
		},
		"internalIp": ServicePropertyDefinition{
			Type:      "string",
			Source:    "agent",
			Updatable: "always",
		},
		"status": ServicePropertyDefinition{
			Type:      "string",
			Source:    "agent",
			Updatable: "never",
		},
	}

	tests := []struct {
		name          string
		updates       map[string]any
		currentStatus string
		schema        *ServicePropertySchema
		updateSource  string
		expectErr     bool
		errMsg        string
	}{
		{
			name:          "Empty updates are valid",
			updates:       map[string]any{},
			currentStatus: "Started",
			schema:        schema,
			updateSource:  "user",
			expectErr:     false,
		},
		{
			name:          "Nil updates are valid",
			updates:       nil,
			currentStatus: "Started",
			schema:        schema,
			updateSource:  "user",
			expectErr:     false,
		},
		{
			name: "User can update input source properties",
			updates: map[string]any{
				"tags": map[string]any{"env": "prod"},
			},
			currentStatus: "Started",
			schema:        schema,
			updateSource:  "user",
			expectErr:     false,
		},
		{
			name: "User cannot update agent source properties",
			updates: map[string]any{
				"internalIp": "10.0.0.1",
			},
			currentStatus: "Started",
			schema:        schema,
			updateSource:  "user",
			expectErr:     true,
			errMsg:        "cannot be updated by user (source: agent)",
		},
		{
			name: "Agent can update agent source properties",
			updates: map[string]any{
				"internalIp": "10.0.0.1",
			},
			currentStatus: "Started",
			schema:        schema,
			updateSource:  "agent",
			expectErr:     false,
		},
		{
			name: "Agent cannot update input source properties",
			updates: map[string]any{
				"tags": map[string]any{"env": "prod"},
			},
			currentStatus: "Started",
			schema:        schema,
			updateSource:  "agent",
			expectErr:     true,
			errMsg:        "cannot be updated by agent (source: input)",
		},
		{
			name: "Unknown property returns error",
			updates: map[string]any{
				"unknownProp": "value",
			},
			currentStatus: "Started",
			schema:        schema,
			updateSource:  "user",
			expectErr:     true,
			errMsg:        "unknown property: unknownProp",
		},
		{
			name: "Respects updatability rules - never",
			updates: map[string]any{
				"hostname": "newhost",
			},
			currentStatus: "Started",
			schema:        schema,
			updateSource:  "user",
			expectErr:     true,
			errMsg:        "cannot be updated (updatable: never)",
		},
		{
			name: "Respects updatability rules - statuses allowed",
			updates: map[string]any{
				"cpu": 4,
			},
			currentStatus: "Stopped",
			schema:        schema,
			updateSource:  "user",
			expectErr:     false,
		},
		{
			name: "Respects updatability rules - statuses disallowed",
			updates: map[string]any{
				"cpu": 4,
			},
			currentStatus: "Started",
			schema:        schema,
			updateSource:  "user",
			expectErr:     true,
			errMsg:        "cannot be updated in status 'Started'",
		},
		{
			name: "Nil schema allows all updates",
			updates: map[string]any{
				"anything": "value",
			},
			currentStatus: "Started",
			schema:        nil,
			updateSource:  "user",
			expectErr:     false,
		},
		{
			name: "Default source is input",
			updates: map[string]any{
				"implicitInput": "value",
			},
			currentStatus: "Started",
			schema: &ServicePropertySchema{
				"implicitInput": ServicePropertyDefinition{
					Type:      "string",
					Updatable: "always",
					// Source not set, defaults to "input"
				},
			},
			updateSource: "user",
			expectErr:    false,
		},
		{
			name: "Default source is input - agent cannot update",
			updates: map[string]any{
				"implicitInput": "value",
			},
			currentStatus: "Started",
			schema: &ServicePropertySchema{
				"implicitInput": ServicePropertyDefinition{
					Type:      "string",
					Updatable: "always",
					// Source not set, defaults to "input"
				},
			},
			updateSource: "agent",
			expectErr:    true,
			errMsg:       "cannot be updated by agent (source: input)",
		},
		{
			name: "Multiple properties - all valid",
			updates: map[string]any{
				"tags": map[string]any{"env": "prod"},
				"cpu":  4,
			},
			currentStatus: "Stopped",
			schema:        schema,
			updateSource:  "user",
			expectErr:     false,
		},
		{
			name: "Multiple properties - one invalid",
			updates: map[string]any{
				"tags":     map[string]any{"env": "prod"},
				"hostname": "newhost",
			},
			currentStatus: "Stopped",
			schema:        schema,
			updateSource:  "user",
			expectErr:     true,
			errMsg:        "cannot be updated (updatable: never)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePropertiesForUpdate(tt.updates, tt.currentStatus, tt.schema, tt.updateSource)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidatePropertiesForCreation tests the ValidatePropertiesForCreation function
func TestValidatePropertiesForCreation(t *testing.T) {
	schema := &ServicePropertySchema{
		"hostname": ServicePropertyDefinition{
			Type:      "string",
			Source:    "input",
			Updatable: "never",
		},
		"cpu": ServicePropertyDefinition{
			Type:        "integer",
			Source:      "input",
			Updatable:   "statuses",
			UpdatableIn: []string{"Stopped"},
		},
		"instanceId": ServicePropertyDefinition{
			Type:      "string",
			Source:    "agent",
			Updatable: "never",
		},
	}

	tests := []struct {
		name       string
		properties map[string]any
		schema     *ServicePropertySchema
		source     string
		expectErr  bool
		errMsg     string
	}{
		{
			name:       "Empty properties are valid",
			properties: map[string]any{},
			schema:     schema,
			source:     "user",
			expectErr:  false,
		},
		{
			name:       "Nil properties are valid",
			properties: nil,
			schema:     schema,
			source:     "user",
			expectErr:  false,
		},
		{
			name: "User can set input source properties",
			properties: map[string]any{
				"hostname": "myhost",
				"cpu":      4,
			},
			schema:    schema,
			source:    "user",
			expectErr: false,
		},
		{
			name: "User can set state-conditional properties during creation",
			properties: map[string]any{
				"cpu": 4,
			},
			schema:    schema,
			source:    "user",
			expectErr: false,
		},
		{
			name: "User can set immutable properties during creation",
			properties: map[string]any{
				"hostname": "myhost",
			},
			schema:    schema,
			source:    "user",
			expectErr: false,
		},
		{
			name: "User cannot set agent source properties",
			properties: map[string]any{
				"instanceId": "i-abc123",
			},
			schema:    schema,
			source:    "user",
			expectErr: true,
			errMsg:    "cannot be set by user (source: agent)",
		},
		{
			name: "Agent can set agent source properties",
			properties: map[string]any{
				"instanceId": "i-abc123",
			},
			schema:    schema,
			source:    "agent",
			expectErr: false,
		},
		{
			name: "Agent cannot set input source properties",
			properties: map[string]any{
				"hostname": "myhost",
			},
			schema:    schema,
			source:    "agent",
			expectErr: true,
			errMsg:    "cannot be set by agent (source: input)",
		},
		{
			name: "Unknown property returns error",
			properties: map[string]any{
				"unknownProp": "value",
			},
			schema:    schema,
			source:    "user",
			expectErr: true,
			errMsg:    "unknown property: unknownProp",
		},
		{
			name: "Nil schema allows all properties",
			properties: map[string]any{
				"anything": "value",
			},
			schema:    nil,
			source:    "user",
			expectErr: false,
		},
		{
			name: "Default source is input",
			properties: map[string]any{
				"implicitInput": "value",
			},
			schema: &ServicePropertySchema{
				"implicitInput": ServicePropertyDefinition{
					Type: "string",
					// Source not set, defaults to "input"
				},
			},
			source:    "user",
			expectErr: false,
		},
		{
			name: "Default source is input - agent cannot set",
			properties: map[string]any{
				"implicitInput": "value",
			},
			schema: &ServicePropertySchema{
				"implicitInput": ServicePropertyDefinition{
					Type: "string",
					// Source not set, defaults to "input"
				},
			},
			source:    "agent",
			expectErr: true,
			errMsg:    "cannot be set by agent (source: input)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePropertiesForCreation(tt.properties, tt.schema, tt.source)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
