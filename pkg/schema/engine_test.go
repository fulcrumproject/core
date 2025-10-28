package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
)

// TestContext is a simple test context
type TestContext struct {
	Actor string
	State string
}

// Helper to create a basic engine for testing
func newTestEngine() *Engine[TestContext] {
	validators := map[string]PropertyValidator[TestContext]{
		"minLength": &MinLengthValidator[TestContext]{},
		"maxLength": &MaxLengthValidator[TestContext]{},
		"pattern":   &PatternValidator[TestContext]{},
		"min":       &MinValidator[TestContext]{},
		"max":       &MaxValidator[TestContext]{},
		"enum":      &EnumValidator[TestContext]{},
		"minItems":  &MinItemsValidator[TestContext]{},
		"maxItems":  &MaxItemsValidator[TestContext]{},
	}

	schemaValidators := map[string]SchemaValidator[TestContext]{
		"exactlyOne": &ExactlyOneValidator[TestContext]{},
	}

	generators := map[string]Generator[TestContext]{}

	authorizers := map[string]Authorizer[TestContext]{}

	return NewEngine(authorizers, validators, schemaValidators, generators, nil)
}

func TestEngine_ApplyCreate_BasicTypes(t *testing.T) {
	engine := newTestEngine()
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	tests := []struct {
		name       string
		schema     Schema
		properties map[string]any
		wantErr    bool
	}{
		{
			name: "string property",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"name": {Type: "string", Required: true},
				},
			},
			properties: map[string]any{"name": "test"},
			wantErr:    false,
		},
		{
			name: "integer property",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"count": {Type: "integer", Required: true},
				},
			},
			properties: map[string]any{"count": 42},
			wantErr:    false,
		},
		{
			name: "integer from float64 (JSON)",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"count": {Type: "integer", Required: true},
				},
			},
			properties: map[string]any{"count": float64(42)},
			wantErr:    false,
		},
		{
			name: "integer from json.Number",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"count": {Type: "integer", Required: true},
				},
			},
			properties: map[string]any{"count": json.Number("42")},
			wantErr:    false,
		},
		{
			name: "number property",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"price": {Type: "number", Required: true},
				},
			},
			properties: map[string]any{"price": 19.99},
			wantErr:    false,
		},
		{
			name: "number from json.Number",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"price": {Type: "number", Required: true},
				},
			},
			properties: map[string]any{"price": json.Number("19.99")},
			wantErr:    false,
		},
		{
			name: "boolean property",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"enabled": {Type: "boolean", Required: true},
				},
			},
			properties: map[string]any{"enabled": true},
			wantErr:    false,
		},
		{
			name: "json property",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"metadata": {Type: "json"},
				},
			},
			properties: map[string]any{"metadata": map[string]any{"key": "value"}},
			wantErr:    false,
		},
		{
			name: "invalid type",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"name": {Type: "string", Required: true},
				},
			},
			properties: map[string]any{"name": 123},
			wantErr:    true,
		},
		{
			name: "missing required property",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"name": {Type: "string", Required: true},
				},
			},
			properties: map[string]any{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.ApplyCreate(ctx, testCtx, tt.schema, tt.properties)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("ApplyCreate() returned nil result")
			}
		})
	}
}

func TestEngine_ApplyCreate_DefaultValues(t *testing.T) {
	engine := newTestEngine()
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	schema := Schema{
		Properties: map[string]PropertyDefinition{
			"name":   {Type: "string", Required: true},
			"status": {Type: "string", Default: "active"},
			"count":  {Type: "integer", Default: 0},
		},
	}

	properties := map[string]any{
		"name": "test",
	}

	result, err := engine.ApplyCreate(ctx, testCtx, schema, properties)
	if err != nil {
		t.Fatalf("ApplyCreate() error = %v", err)
	}

	if result["status"] != "active" {
		t.Errorf("expected default status='active', got %v", result["status"])
	}

	if result["count"] != 0 {
		t.Errorf("expected default count=0, got %v", result["count"])
	}
}

func TestEngine_ApplyCreate_Immutable(t *testing.T) {
	engine := newTestEngine()
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	schema := Schema{
		Properties: map[string]PropertyDefinition{
			"id":   {Type: "string", Required: true, Immutable: true},
			"name": {Type: "string", Required: true},
		},
	}

	// Create
	properties := map[string]any{
		"id":   "123",
		"name": "test",
	}

	result, err := engine.ApplyCreate(ctx, testCtx, schema, properties)
	if err != nil {
		t.Fatalf("ApplyCreate() error = %v", err)
	}

	// Update - try to change immutable property
	updateProps := map[string]any{
		"id":   "456", // Try to change immutable
		"name": "new name",
	}

	_, err = engine.ApplyUpdate(ctx, testCtx, schema, result, updateProps)
	if err == nil {
		t.Error("expected error when changing immutable property")
	}

	// Update - same value should be allowed (idempotent)
	updateProps2 := map[string]any{
		"id":   "123", // Same value
		"name": "new name",
	}

	result2, err := engine.ApplyUpdate(ctx, testCtx, schema, result, updateProps2)
	if err != nil {
		t.Errorf("ApplyUpdate() with same immutable value error = %v", err)
	}
	if result2["name"] != "new name" {
		t.Errorf("expected name='new name', got %v", result2["name"])
	}
}

func TestEngine_ApplyCreate_NestedObjects(t *testing.T) {
	engine := newTestEngine()
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	schema := Schema{
		Properties: map[string]PropertyDefinition{
			"user": {
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"name": {Type: "string", Required: true},
					"email": {
						Type: "string",
						Validators: []ValidatorConfig{
							{Type: "pattern", Config: map[string]any{"pattern": "^[a-z]+@[a-z]+\\.[a-z]+$"}},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name       string
		properties map[string]any
		wantErr    bool
	}{
		{
			name: "valid nested object",
			properties: map[string]any{
				"user": map[string]any{
					"name":  "John",
					"email": "john@example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "missing required nested property",
			properties: map[string]any{
				"user": map[string]any{
					"email": "john@example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid nested validator",
			properties: map[string]any{
				"user": map[string]any{
					"name":  "John",
					"email": "invalid-email",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.ApplyCreate(ctx, testCtx, schema, tt.properties)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_ApplyCreate_Arrays(t *testing.T) {
	engine := newTestEngine()
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	schema := Schema{
		Properties: map[string]PropertyDefinition{
			"ports": {
				Type: "array",
				Items: &PropertyDefinition{
					Type: "integer",
					Validators: []ValidatorConfig{
						{Type: "min", Config: map[string]any{"value": 1}},
						{Type: "max", Config: map[string]any{"value": 65535}},
					},
				},
			},
		},
	}

	tests := []struct {
		name       string
		properties map[string]any
		wantErr    bool
	}{
		{
			name:       "valid array",
			properties: map[string]any{"ports": []any{80, 443, 8080}},
			wantErr:    false,
		},
		{
			name:       "invalid array item",
			properties: map[string]any{"ports": []any{80, 70000, 8080}},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.ApplyCreate(ctx, testCtx, schema, tt.properties)
			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_ApplyCreate_WithSecrets(t *testing.T) {
	vault := &MockVault{}

	// Mock vault to accept the secret save
	vault.On("Save", mock.Anything, mock.Anything, "my-secret-key-123", mock.Anything).Return(nil).Once()

	validators := map[string]PropertyValidator[TestContext]{
		"minLength": &MinLengthValidator[TestContext]{},
	}

	engine := NewEngine(nil, validators, nil, nil, vault)
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	schema := Schema{
		Properties: map[string]PropertyDefinition{
			"apiKey": {
				Type:   "string",
				Secret: &SecretConfig{Type: "persistent"},
				Validators: []ValidatorConfig{
					{Type: "minLength", Config: map[string]any{"value": 10}},
				},
			},
		},
	}

	properties := map[string]any{
		"apiKey": "my-secret-key-123",
	}

	result, err := engine.ApplyCreate(ctx, testCtx, schema, properties)
	if err != nil {
		t.Fatalf("ApplyCreate() error = %v", err)
	}

	// Check that apiKey is now a vault reference
	apiKey, ok := result["apiKey"].(string)
	if !ok {
		t.Fatal("apiKey is not a string")
	}

	if len(apiKey) < 10 || apiKey[:8] != "vault://" {
		t.Errorf("expected vault:// reference, got %s", apiKey)
	}

	// Verify that the vault Save method was called
	vault.AssertExpectations(t)
}

func TestEngine_ApplyCreate_MultipleValidationErrors(t *testing.T) {
	engine := newTestEngine()
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	schema := Schema{
		Properties: map[string]PropertyDefinition{
			"name": {
				Type:     "string",
				Required: true,
				Validators: []ValidatorConfig{
					{Type: "minLength", Config: map[string]any{"value": 5}},
				},
			},
			"age": {
				Type:     "integer",
				Required: true,
				Validators: []ValidatorConfig{
					{Type: "min", Config: map[string]any{"value": 18}},
				},
			},
			"email": {
				Type:     "string",
				Required: true,
				Validators: []ValidatorConfig{
					{Type: "pattern", Config: map[string]any{"pattern": "^[^@]+@[^@]+\\.[^@]+$"}},
				},
			},
		},
	}

	properties := map[string]any{
		"name":  "Bob",     // Too short (< 5 chars)
		"age":   15,        // Too young (< 18)
		"email": "invalid", // Invalid email format
	}

	_, err := engine.ApplyCreate(ctx, testCtx, schema, properties)
	if err == nil {
		t.Fatal("ApplyCreate() expected error, got nil")
	}

	// Should be a ValidationError with all 3 errors
	validationErr, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}

	if len(validationErr.Errors) != 3 {
		t.Errorf("expected 3 validation errors, got %d", len(validationErr.Errors))
	}

	// Verify each error is present
	errorPaths := make(map[string]bool)
	for _, e := range validationErr.Errors {
		errorPaths[e.Path] = true
	}

	if !errorPaths["name"] {
		t.Error("expected validation error for 'name' property")
	}
	if !errorPaths["age"] {
		t.Error("expected validation error for 'age' property")
	}
	if !errorPaths["email"] {
		t.Error("expected validation error for 'email' property")
	}
}

func TestEngine_ValidateSchema(t *testing.T) {
	engine := newTestEngine()

	tests := []struct {
		name    string
		schema  Schema
		wantErr bool
	}{
		{
			name: "valid schema",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"name": {Type: "string", Required: true},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"name": {Type: "invalid"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty property name",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"": {Type: "string"},
				},
			},
			wantErr: true,
		},
		{
			name: "nil properties map",
			schema: Schema{
				Properties: nil,
			},
			wantErr: true,
		},
		{
			name: "empty properties map",
			schema: Schema{
				Properties: map[string]PropertyDefinition{},
			},
			wantErr: true,
		},
		{
			name: "both default and generator",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"value": {
						Type:      "string",
						Default:   "test",
						Generator: &GeneratorConfig{Type: "test"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "unknown validator",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"name": {
						Type: "string",
						Validators: []ValidatorConfig{
							{Type: "unknownValidator", Config: map[string]any{}},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateSchema(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_WithMockValidator(t *testing.T) {
	// This test demonstrates using MockPropertyValidator
	mockValidator := &MockPropertyValidator[TestContext]{}
	mockValidator.On("Validate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	mockValidator.On("ValidateConfig", mock.Anything, mock.Anything).Return(nil)

	validators := map[string]PropertyValidator[TestContext]{
		"custom": mockValidator,
	}

	engine := NewEngine(nil, validators, nil, nil, nil)

	// First validate the schema (calls ValidateConfig)
	schema := Schema{
		Properties: map[string]PropertyDefinition{
			"field": {
				Type: "string",
				Validators: []ValidatorConfig{
					{Type: "custom", Config: map[string]any{"key": "value"}},
				},
			},
		},
	}

	err := engine.ValidateSchema(schema)
	if err != nil {
		t.Fatalf("ValidateSchema() error = %v", err)
	}

	// Then apply (calls Validate)
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	result, err := engine.ApplyCreate(ctx, testCtx, schema, map[string]any{"field": "test"})
	if err != nil {
		t.Fatalf("ApplyCreate() error = %v", err)
	}

	if result["field"] != "test" {
		t.Errorf("expected field='test', got %v", result["field"])
	}

	// Verify mock was called
	mockValidator.AssertExpectations(t)
}

func TestEngine_WithMockGenerator(t *testing.T) {
	// This test demonstrates using MockGenerator
	mockGenerator := &MockGenerator[TestContext]{}
	mockGenerator.On("Generate", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("generated-value", true, nil).Once()
	mockGenerator.On("ValidateConfig", mock.Anything, mock.Anything).Return(nil)

	generators := map[string]Generator[TestContext]{
		"testGen": mockGenerator,
	}

	engine := NewEngine(nil, nil, nil, generators, nil)

	// First validate the schema (calls ValidateConfig)
	schema := Schema{
		Properties: map[string]PropertyDefinition{
			"field": {
				Type:      "string",
				Generator: &GeneratorConfig{Type: "testGen", Config: map[string]any{}},
			},
		},
	}

	err := engine.ValidateSchema(schema)
	if err != nil {
		t.Fatalf("ValidateSchema() error = %v", err)
	}

	// Then apply (calls Generate)
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	result, err := engine.ApplyCreate(ctx, testCtx, schema, map[string]any{})
	if err != nil {
		t.Fatalf("ApplyCreate() error = %v", err)
	}

	if result["field"] != "generated-value" {
		t.Errorf("expected field='generated-value', got %v", result["field"])
	}

	// Verify mock was called
	mockGenerator.AssertExpectations(t)
}

func TestExtractVaultReferences(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]any
		expected   []string
	}{
		{
			name: "single string reference",
			properties: map[string]any{
				"apiKey": "vault://abc123",
			},
			expected: []string{"abc123"},
		},
		{
			name: "multiple references",
			properties: map[string]any{
				"apiKey":      "vault://abc123",
				"password":    "vault://def456",
				"regularProp": "not-a-secret",
			},
			expected: []string{"abc123", "def456"},
		},
		{
			name: "nested object with reference",
			properties: map[string]any{
				"database": map[string]any{
					"host":     "localhost",
					"password": "vault://xyz789",
				},
			},
			expected: []string{"xyz789"},
		},
		{
			name: "array with references",
			properties: map[string]any{
				"secrets": []any{
					"vault://ref1",
					"vault://ref2",
					"not-a-secret",
				},
			},
			expected: []string{"ref1", "ref2"},
		},
		{
			name: "deeply nested structure",
			properties: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": []any{
							map[string]any{
								"secret": "vault://deep123",
							},
						},
					},
				},
			},
			expected: []string{"deep123"},
		},
		{
			name: "no vault references",
			properties: map[string]any{
				"name":   "test",
				"age":    30,
				"active": true,
			},
			expected: []string{},
		},
		{
			name:       "empty properties",
			properties: map[string]any{},
			expected:   []string{},
		},
		{
			name: "mixed types",
			properties: map[string]any{
				"secret1": "vault://secret1",
				"number":  42,
				"bool":    true,
				"null":    nil,
				"secret2": "vault://secret2",
				"object": map[string]any{
					"nested": "vault://secret3",
				},
			},
			expected: []string{"secret1", "secret2", "secret3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVaultReferences(tt.properties)
			if len(tt.expected) == 0 && len(result) == 0 {
				return // Both empty, test passes
			}
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d references, got %d", len(tt.expected), len(result))
			}
			// Check all expected references are present
			for _, exp := range tt.expected {
				found := false
				for _, res := range result {
					if res == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected reference %s not found in result", exp)
				}
			}
		})
	}
}

func TestExtractVaultReferences_EmptyPrefix(t *testing.T) {
	properties := map[string]any{
		"notAVaultRef": "vault://", // Empty reference after prefix
	}

	result := extractVaultReferences(properties)
	if len(result) != 1 || result[0] != "" {
		t.Errorf("expected one empty string reference, got %v", result)
	}
}

func TestEngine_CleanupVaultSecrets(t *testing.T) {
	mockVault := NewMockVault(t)
	engine := NewEngine[TestContext](nil, nil, nil, nil, mockVault)
	ctx := context.Background()

	tests := []struct {
		name       string
		properties map[string]any
		setupMock  func()
	}{
		{
			name: "cleanup single reference",
			properties: map[string]any{
				"apiKey": "vault://abc123",
			},
			setupMock: func() {
				mockVault.On("Delete", mock.Anything, "abc123").Return(nil).Once()
			},
		},
		{
			name: "cleanup multiple references",
			properties: map[string]any{
				"apiKey":   "vault://abc123",
				"password": "vault://def456",
			},
			setupMock: func() {
				mockVault.On("Delete", mock.Anything, "abc123").Return(nil).Once()
				mockVault.On("Delete", mock.Anything, "def456").Return(nil).Once()
			},
		},
		{
			name: "cleanup nested references",
			properties: map[string]any{
				"database": map[string]any{
					"password": "vault://xyz789",
				},
			},
			setupMock: func() {
				mockVault.On("Delete", mock.Anything, "xyz789").Return(nil).Once()
			},
		},
		{
			name: "no references to cleanup",
			properties: map[string]any{
				"name": "test",
			},
			setupMock: func() {
				// No mock expectations - vault should not be called
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock for each test
			mockVault.ExpectedCalls = nil
			mockVault.Calls = nil

			tt.setupMock()

			// Call cleanup - should not return error (best-effort)
			engine.CleanupVaultSecrets(ctx, tt.properties)

			// Verify mock expectations
			mockVault.AssertExpectations(t)
		})
	}
}

func TestEngine_CleanupVaultSecrets_WithErrors(t *testing.T) {
	mockVault := NewMockVault(t)
	engine := NewEngine[TestContext](nil, nil, nil, nil, mockVault)
	ctx := context.Background()

	properties := map[string]any{
		"secret1": "vault://abc123",
		"secret2": "vault://def456",
	}

	// First delete succeeds, second fails
	mockVault.On("Delete", mock.Anything, "abc123").Return(nil).Once()
	mockVault.On("Delete", mock.Anything, "def456").Return(fmt.Errorf("vault error")).Once()

	// Should not panic or fail - best effort cleanup
	engine.CleanupVaultSecrets(ctx, properties)

	mockVault.AssertExpectations(t)
}

func TestEngine_CleanupVaultSecrets_NilVault(t *testing.T) {
	engine := NewEngine[TestContext](nil, nil, nil, nil, nil)
	ctx := context.Background()

	properties := map[string]any{
		"secret": "vault://abc123",
	}

	// Should not panic when vault is nil
	engine.CleanupVaultSecrets(ctx, properties)
}

func TestEngine_CleanupVaultSecrets_NilProperties(t *testing.T) {
	mockVault := NewMockVault(t)
	engine := NewEngine[TestContext](nil, nil, nil, nil, mockVault)
	ctx := context.Background()

	// Should not panic or call vault when properties is nil
	engine.CleanupVaultSecrets(ctx, nil)

	// Verify vault was not called
	mockVault.AssertNotCalled(t, "Delete")
}

func TestExtractEphemeralSecretProperties(t *testing.T) {
	tests := []struct {
		name       string
		schema     Schema
		properties map[string]any
		expected   map[string]any
	}{
		{
			name: "single ephemeral secret",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"setupPassword": {
						Type:   "string",
						Secret: &SecretConfig{Type: "ephemeral"},
					},
					"apiKey": {
						Type:   "string",
						Secret: &SecretConfig{Type: "persistent"},
					},
				},
			},
			properties: map[string]any{
				"setupPassword": "vault://setup123",
				"apiKey":        "vault://api456",
			},
			expected: map[string]any{
				"setupPassword": "vault://setup123",
				// apiKey should NOT be included (it's persistent)
			},
		},
		{
			name: "multiple ephemeral secrets",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"tempToken": {
						Type:   "string",
						Secret: &SecretConfig{Type: "ephemeral"},
					},
					"installKey": {
						Type:   "string",
						Secret: &SecretConfig{Type: "ephemeral"},
					},
				},
			},
			properties: map[string]any{
				"tempToken":  "vault://temp123",
				"installKey": "vault://install456",
			},
			expected: map[string]any{
				"tempToken":  "vault://temp123",
				"installKey": "vault://install456",
			},
		},
		{
			name: "nested ephemeral secret in object",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"database": {
						Type: "object",
						Properties: map[string]PropertyDefinition{
							"setupPassword": {
								Type:   "string",
								Secret: &SecretConfig{Type: "ephemeral"},
							},
							"host": {
								Type: "string",
							},
						},
					},
				},
			},
			properties: map[string]any{
				"database": map[string]any{
					"setupPassword": "vault://setup123",
					"host":          "localhost",
				},
			},
			expected: map[string]any{
				"database": map[string]any{
					"setupPassword": "vault://setup123",
				},
			},
		},
		{
			name: "array of objects with ephemeral secrets",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"users": {
						Type: "array",
						Items: &PropertyDefinition{
							Type: "object",
							Properties: map[string]PropertyDefinition{
								"name": {
									Type: "string",
								},
								"tempPassword": {
									Type:   "string",
									Secret: &SecretConfig{Type: "ephemeral"},
								},
							},
						},
					},
				},
			},
			properties: map[string]any{
				"users": []any{
					map[string]any{
						"name":         "user1",
						"tempPassword": "vault://pass1",
					},
					map[string]any{
						"name":         "user2",
						"tempPassword": "vault://pass2",
					},
				},
			},
			expected: map[string]any{
				"users": []any{
					map[string]any{
						"tempPassword": "vault://pass1",
					},
					map[string]any{
						"tempPassword": "vault://pass2",
					},
				},
			},
		},
		{
			name: "no ephemeral secrets",
			schema: Schema{
				Properties: map[string]PropertyDefinition{
					"apiKey": {
						Type:   "string",
						Secret: &SecretConfig{Type: "persistent"},
					},
					"regularProp": {
						Type: "string",
					},
				},
			},
			properties: map[string]any{
				"apiKey":      "vault://api456",
				"regularProp": "value",
			},
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEphemeralSecretProperties(tt.schema, tt.properties)

			if len(tt.expected) == 0 {
				if len(result) != 0 {
					t.Errorf("expected empty result, got %v", result)
				}
				return
			}

			// Convert to JSON for easier comparison
			expectedJSON, _ := json.Marshal(tt.expected)
			resultJSON, _ := json.Marshal(result)

			if string(expectedJSON) != string(resultJSON) {
				t.Errorf("extractEphemeralSecretProperties() mismatch\nExpected: %s\nGot: %s", expectedJSON, resultJSON)
			}
		})
	}
}

func TestEngine_CleanupEphemeralSecrets(t *testing.T) {
	mockVault := NewMockVault(t)
	engine := NewEngine[TestContext](nil, nil, nil, nil, mockVault)
	ctx := context.Background()

	schema := Schema{
		Properties: map[string]PropertyDefinition{
			"setupPassword": {
				Type:   "string",
				Secret: &SecretConfig{Type: "ephemeral"},
			},
			"apiKey": {
				Type:   "string",
				Secret: &SecretConfig{Type: "persistent"},
			},
			"regularProp": {
				Type: "string",
			},
		},
	}

	properties := map[string]any{
		"setupPassword": "vault://setup123",
		"apiKey":        "vault://api456",
		"regularProp":   "some-value",
	}

	// Only setupPassword should be deleted (ephemeral)
	mockVault.On("Delete", mock.Anything, "setup123").Return(nil).Once()

	// Call cleanup
	engine.CleanupEphemeralSecrets(ctx, schema, properties)

	// Verify only ephemeral secret was deleted
	mockVault.AssertExpectations(t)
	mockVault.AssertNotCalled(t, "Delete", mock.Anything, "api456") // persistent should not be deleted
}

func TestEngine_CleanupEphemeralSecrets_NoEphemeral(t *testing.T) {
	mockVault := NewMockVault(t)
	engine := NewEngine[TestContext](nil, nil, nil, nil, mockVault)
	ctx := context.Background()

	schema := Schema{
		Properties: map[string]PropertyDefinition{
			"apiKey": {
				Type:   "string",
				Secret: &SecretConfig{Type: "persistent"},
			},
		},
	}

	properties := map[string]any{
		"apiKey": "vault://api456",
	}

	// Should not call vault.Delete since there are no ephemeral secrets
	engine.CleanupEphemeralSecrets(ctx, schema, properties)

	// Verify vault was not called
	mockVault.AssertNotCalled(t, "Delete")
}
