package schema

import (
	"context"
	"testing"
)

func TestEnumValidator_Validate(t *testing.T) {
	validator := &EnumValidator[TestContext]{}
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	tests := []struct {
		name    string
		value   any
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid - string in enum",
			value:   "active",
			config:  map[string]any{"values": []any{"active", "inactive", "pending"}},
			wantErr: false,
		},
		{
			name:    "invalid - string not in enum",
			value:   "unknown",
			config:  map[string]any{"values": []any{"active", "inactive", "pending"}},
			wantErr: true,
		},
		{
			name:    "valid - integer in enum",
			value:   2,
			config:  map[string]any{"values": []any{1, 2, 3}},
			wantErr: false,
		},
		{
			name:    "invalid - integer not in enum",
			value:   5,
			config:  map[string]any{"values": []any{1, 2, 3}},
			wantErr: true,
		},
		{
			name:    "invalid - missing values config",
			value:   "active",
			config:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, testCtx, OperationCreate, "testProp", nil, tt.value, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnumValidator_ValidateConfig(t *testing.T) {
	validator := &EnumValidator[TestContext]{}

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  map[string]any{"values": []any{"a", "b", "c"}},
			wantErr: false,
		},
		{
			name:    "valid single value",
			config:  map[string]any{"values": []any{"only-one"}},
			wantErr: false,
		},
		{
			name:    "invalid - empty array",
			config:  map[string]any{"values": []any{}},
			wantErr: true,
		},
		{
			name:    "invalid - missing values",
			config:  map[string]any{},
			wantErr: true,
		},
		{
			name:    "invalid - wrong type",
			config:  map[string]any{"values": "not-an-array"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig("testProp", tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinItemsValidator_Validate(t *testing.T) {
	validator := &MinItemsValidator[TestContext]{}
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	tests := []struct {
		name    string
		value   any
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid - meets minimum",
			value:   []any{"a", "b", "c"},
			config:  map[string]any{"value": 2},
			wantErr: false,
		},
		{
			name:    "valid - at minimum",
			value:   []any{"a", "b"},
			config:  map[string]any{"value": 2},
			wantErr: false,
		},
		{
			name:    "invalid - below minimum",
			value:   []any{"a"},
			config:  map[string]any{"value": 2},
			wantErr: true,
		},
		{
			name:    "valid - empty array with zero minimum",
			value:   []any{},
			config:  map[string]any{"value": 0},
			wantErr: false,
		},
		{
			name:    "invalid - non-array value",
			value:   "not-an-array",
			config:  map[string]any{"value": 2},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, testCtx, OperationCreate, "testProp", nil, tt.value, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMinItemsValidator_ValidateConfig(t *testing.T) {
	validator := &MinItemsValidator[TestContext]{}

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  map[string]any{"value": 5},
			wantErr: false,
		},
		{
			name:    "valid zero",
			config:  map[string]any{"value": 0},
			wantErr: false,
		},
		{
			name:    "invalid - negative value",
			config:  map[string]any{"value": -1},
			wantErr: true,
		},
		{
			name:    "invalid - missing value",
			config:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig("testProp", tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxItemsValidator_Validate(t *testing.T) {
	validator := &MaxItemsValidator[TestContext]{}
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	tests := []struct {
		name    string
		value   any
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid - within maximum",
			value:   []any{"a", "b"},
			config:  map[string]any{"value": 5},
			wantErr: false,
		},
		{
			name:    "valid - at maximum",
			value:   []any{"a", "b", "c"},
			config:  map[string]any{"value": 3},
			wantErr: false,
		},
		{
			name:    "invalid - exceeds maximum",
			value:   []any{"a", "b", "c", "d"},
			config:  map[string]any{"value": 3},
			wantErr: true,
		},
		{
			name:    "valid - empty array",
			value:   []any{},
			config:  map[string]any{"value": 5},
			wantErr: false,
		},
		{
			name:    "invalid - non-array value",
			value:   "not-an-array",
			config:  map[string]any{"value": 5},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, testCtx, OperationCreate, "testProp", nil, tt.value, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaxItemsValidator_ValidateConfig(t *testing.T) {
	validator := &MaxItemsValidator[TestContext]{}

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  map[string]any{"value": 10},
			wantErr: false,
		},
		{
			name:    "valid zero",
			config:  map[string]any{"value": 0},
			wantErr: false,
		},
		{
			name:    "invalid - negative value",
			config:  map[string]any{"value": -5},
			wantErr: true,
		},
		{
			name:    "invalid - missing value",
			config:  map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig("testProp", tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
