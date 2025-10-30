package schema

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMinValidator_Validate(t *testing.T) {
	validator := &MinValidator[TestContext]{}
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	tests := []struct {
		name    string
		value   any
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid - integer above minimum",
			value:   25,
			config:  map[string]any{"value": 18},
			wantErr: false,
		},
		{
			name:    "valid - integer at minimum",
			value:   18,
			config:  map[string]any{"value": 18},
			wantErr: false,
		},
		{
			name:    "invalid - integer below minimum",
			value:   15,
			config:  map[string]any{"value": 18},
			wantErr: true,
		},
		{
			name:    "valid - float above minimum",
			value:   19.5,
			config:  map[string]any{"value": 18.0},
			wantErr: false,
		},
		{
			name:    "invalid - float below minimum",
			value:   17.5,
			config:  map[string]any{"value": 18.0},
			wantErr: true,
		},
		{
			name:    "valid - json.Number above minimum",
			value:   json.Number("25"),
			config:  map[string]any{"value": 18},
			wantErr: false,
		},
		{
			name:    "invalid - non-numeric value",
			value:   "not a number",
			config:  map[string]any{"value": 18},
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

func TestMinValidator_ValidateConfig(t *testing.T) {
	validator := &MinValidator[TestContext]{}

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid integer config",
			config:  map[string]any{"value": 10},
			wantErr: false,
		},
		{
			name:    "valid float config",
			config:  map[string]any{"value": 10.5},
			wantErr: false,
		},
		{
			name:    "valid negative",
			config:  map[string]any{"value": -100},
			wantErr: false,
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

func TestMaxValidator_Validate(t *testing.T) {
	validator := &MaxValidator[TestContext]{}
	ctx := context.Background()
	testCtx := TestContext{Actor: "user"}

	tests := []struct {
		name    string
		value   any
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid - integer below maximum",
			value:   25,
			config:  map[string]any{"value": 100},
			wantErr: false,
		},
		{
			name:    "valid - integer at maximum",
			value:   100,
			config:  map[string]any{"value": 100},
			wantErr: false,
		},
		{
			name:    "invalid - integer above maximum",
			value:   150,
			config:  map[string]any{"value": 100},
			wantErr: true,
		},
		{
			name:    "valid - float below maximum",
			value:   99.5,
			config:  map[string]any{"value": 100.0},
			wantErr: false,
		},
		{
			name:    "invalid - float above maximum",
			value:   100.5,
			config:  map[string]any{"value": 100.0},
			wantErr: true,
		},
		{
			name:    "valid - json.Number below maximum",
			value:   json.Number("25"),
			config:  map[string]any{"value": 100},
			wantErr: false,
		},
		{
			name:    "invalid - non-numeric value",
			value:   "not a number",
			config:  map[string]any{"value": 100},
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

func TestMaxValidator_ValidateConfig(t *testing.T) {
	validator := &MaxValidator[TestContext]{}

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name:    "valid integer config",
			config:  map[string]any{"value": 1000},
			wantErr: false,
		},
		{
			name:    "valid float config",
			config:  map[string]any{"value": 99.99},
			wantErr: false,
		},
		{
			name:    "valid negative",
			config:  map[string]any{"value": -10},
			wantErr: false,
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
