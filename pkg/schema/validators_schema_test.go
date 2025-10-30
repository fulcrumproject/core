package schema

import (
	"context"
	"testing"
)

func TestExactlyOneValidator_Validate(t *testing.T) {
	validator := &ExactlyOneValidator[TestContext]{}
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	tests := []struct {
		name          string
		oldProperties map[string]any
		newProperties map[string]any
		config        map[string]any
		wantErr       bool
	}{
		{
			name:          "valid - first property provided",
			oldProperties: nil,
			newProperties: map[string]any{"password": "secret"},
			config:        map[string]any{"properties": []any{"password", "sshKey"}},
			wantErr:       false,
		},
		{
			name:          "valid - second property provided",
			oldProperties: nil,
			newProperties: map[string]any{"sshKey": "ssh-rsa ..."},
			config:        map[string]any{"properties": []any{"password", "sshKey"}},
			wantErr:       false,
		},
		{
			name:          "invalid - both properties provided",
			oldProperties: nil,
			newProperties: map[string]any{"password": "secret", "sshKey": "ssh-rsa ..."},
			config:        map[string]any{"properties": []any{"password", "sshKey"}},
			wantErr:       true,
		},
		{
			name:          "invalid - none provided",
			oldProperties: nil,
			newProperties: map[string]any{},
			config:        map[string]any{"properties": []any{"password", "sshKey"}},
			wantErr:       true,
		},
		{
			name:          "invalid - nil values don't count",
			oldProperties: nil,
			newProperties: map[string]any{"password": nil, "sshKey": nil},
			config:        map[string]any{"properties": []any{"password", "sshKey"}},
			wantErr:       true,
		},
		{
			name:          "valid - one non-nil and one nil",
			oldProperties: nil,
			newProperties: map[string]any{"password": "secret", "sshKey": nil},
			config:        map[string]any{"properties": []any{"password", "sshKey"}},
			wantErr:       false,
		},
		{
			name:          "valid - three properties, exactly one provided",
			oldProperties: nil,
			newProperties: map[string]any{"method1": "value"},
			config:        map[string]any{"properties": []any{"method1", "method2", "method3"}},
			wantErr:       false,
		},
		{
			name:          "invalid - three properties, two provided",
			oldProperties: nil,
			newProperties: map[string]any{"method1": "value1", "method2": "value2"},
			config:        map[string]any{"properties": []any{"method1", "method2", "method3"}},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, testCtx, OperationCreate, tt.oldProperties, tt.newProperties, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExactlyOneValidator_ValidateConfig(t *testing.T) {
	validator := &ExactlyOneValidator[TestContext]{}

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid config - two properties",
			config:  map[string]any{"properties": []any{"prop1", "prop2"}},
			wantErr: false,
		},
		{
			name:    "valid config - three properties",
			config:  map[string]any{"properties": []any{"prop1", "prop2", "prop3"}},
			wantErr: false,
		},
		{
			name:    "invalid - only one property",
			config:  map[string]any{"properties": []any{"prop1"}},
			wantErr: true,
		},
		{
			name:    "invalid - empty array",
			config:  map[string]any{"properties": []any{}},
			wantErr: true,
		},
		{
			name:    "invalid - missing properties",
			config:  map[string]any{},
			wantErr: true,
		},
		{
			name:    "invalid - wrong type",
			config:  map[string]any{"properties": "not-an-array"},
			wantErr: true,
		},
		{
			name:    "invalid - non-string in array",
			config:  map[string]any{"properties": []any{"prop1", 123, "prop3"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
