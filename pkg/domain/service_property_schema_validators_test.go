// Tests for domain-specific validators
package domain

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
)

func TestSourceValidator_Validate(t *testing.T) {
	ctx := context.Background()
	validator := &SourceValidator{}

	tests := []struct {
		name      string
		actor     ActorType
		newValue  any
		config    map[string]any
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "user can set property with no source config",
			actor:    ActorUser,
			newValue: "value",
			config:   map[string]any{},
			wantErr:  false,
		},
		{
			name:     "nil value always passes",
			actor:    ActorUser,
			newValue: nil,
			config:   map[string]any{"source": "agent"},
			wantErr:  false,
		},
		{
			name:     "agent can set agent property",
			actor:    ActorAgent,
			newValue: "value",
			config:   map[string]any{"source": "agent"},
			wantErr:  false,
		},
		{
			name:      "user cannot set agent property",
			actor:     ActorUser,
			newValue:  "value",
			config:    map[string]any{"source": "agent"},
			wantErr:   true,
			errSubstr: "can only be set by agents",
		},
		{
			name:      "system source is rejected",
			actor:     ActorUser,
			newValue:  "value",
			config:    map[string]any{"source": "system"},
			wantErr:   true,
			errSubstr: "system-generated",
		},
		{
			name:      "invalid source value",
			actor:     ActorUser,
			newValue:  "value",
			config:    map[string]any{"source": "invalid"},
			wantErr:   true,
			errSubstr: "invalid source",
		},
		{
			name:      "source config not a string",
			actor:     ActorUser,
			newValue:  "value",
			config:    map[string]any{"source": 123},
			wantErr:   true,
			errSubstr: "must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaCtx := ServicePropertyContext{
				Actor:   tt.actor,
				Service: nil,
			}

			err := validator.Validate(ctx, schemaCtx, schema.OperationCreate, "testProp", nil, tt.newValue, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestSourceValidator_ValidateConfig(t *testing.T) {
	validator := &SourceValidator{}

	tests := []struct {
		name      string
		config    map[string]any
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "empty config is valid",
			config:  map[string]any{},
			wantErr: false,
		},
		{
			name:    "no source key is valid",
			config:  map[string]any{"other": "value"},
			wantErr: false,
		},
		{
			name:    "agent source is valid",
			config:  map[string]any{"source": "agent"},
			wantErr: false,
		},
		{
			name:    "system source is valid",
			config:  map[string]any{"source": "system"},
			wantErr: false,
		},
		{
			name:      "invalid source value",
			config:    map[string]any{"source": "user"},
			wantErr:   true,
			errSubstr: "must be 'agent' or 'system'",
		},
		{
			name:      "source not a string",
			config:    map[string]any{"source": 123},
			wantErr:   true,
			errSubstr: "must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig("testProp", tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMutableValidator_Validate(t *testing.T) {
	ctx := context.Background()
	validator := &MutableValidator{}

	serviceID := uuid.New()

	tests := []struct {
		name      string
		operation schema.Operation
		service   *Service
		newValue  any
		config    map[string]any
		wantErr   bool
		errSubstr string
	}{
		{
			name:      "create operation always passes",
			operation: schema.OperationCreate,
			service:   nil,
			newValue:  "value",
			config:    map[string]any{"updatableIn": []any{"Started"}},
			wantErr:   false,
		},
		{
			name:      "nil value always passes",
			operation: schema.OperationUpdate,
			service:   &Service{BaseEntity: BaseEntity{ID: serviceID}, Status: "Stopped"},
			newValue:  nil,
			config:    map[string]any{"updatableIn": []any{"Started"}},
			wantErr:   false,
		},
		{
			name:      "update in allowed state passes",
			operation: schema.OperationUpdate,
			service:   &Service{BaseEntity: BaseEntity{ID: serviceID}, Status: "Started"},
			newValue:  "newValue",
			config:    map[string]any{"updatableIn": []any{"Started", "Stopped"}},
			wantErr:   false,
		},
		{
			name:      "update in disallowed state fails",
			operation: schema.OperationUpdate,
			service:   &Service{BaseEntity: BaseEntity{ID: serviceID}, Status: "Running"},
			newValue:  "newValue",
			config:    map[string]any{"updatableIn": []any{"Started", "Stopped"}},
			wantErr:   true,
			errSubstr: "cannot be updated in state 'Running'",
		},
		{
			name:      "update with single allowed state",
			operation: schema.OperationUpdate,
			service:   &Service{BaseEntity: BaseEntity{ID: serviceID}, Status: "New"},
			newValue:  "newValue",
			config:    map[string]any{"updatableIn": []any{"New"}},
			wantErr:   false,
		},
		{
			name:      "update requires service context",
			operation: schema.OperationUpdate,
			service:   nil,
			newValue:  "newValue",
			config:    map[string]any{"updatableIn": []any{"Started"}},
			wantErr:   true,
			errSubstr: "requires service context",
		},
		{
			name:      "missing updatableIn config",
			operation: schema.OperationUpdate,
			service:   &Service{BaseEntity: BaseEntity{ID: serviceID}, Status: "Started"},
			newValue:  "newValue",
			config:    map[string]any{},
			wantErr:   true,
			errSubstr: "missing 'updatableIn'",
		},
		{
			name:      "updatableIn not an array",
			operation: schema.OperationUpdate,
			service:   &Service{BaseEntity: BaseEntity{ID: serviceID}, Status: "Started"},
			newValue:  "newValue",
			config:    map[string]any{"updatableIn": "Started"},
			wantErr:   true,
			errSubstr: "must be an array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaCtx := ServicePropertyContext{
				Actor:   ActorUser,
				Service: tt.service,
			}

			err := validator.Validate(ctx, schemaCtx, tt.operation, "testProp", nil, tt.newValue, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestMutableValidator_ValidateConfig(t *testing.T) {
	validator := &MutableValidator{}

	tests := []struct {
		name      string
		config    map[string]any
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid config with multiple states",
			config:  map[string]any{"updatableIn": []any{"Started", "Stopped"}},
			wantErr: false,
		},
		{
			name:    "valid config with single state",
			config:  map[string]any{"updatableIn": []any{"New"}},
			wantErr: false,
		},
		{
			name:      "missing updatableIn",
			config:    map[string]any{},
			wantErr:   true,
			errSubstr: "missing 'updatableIn'",
		},
		{
			name:      "updatableIn not an array",
			config:    map[string]any{"updatableIn": "Started"},
			wantErr:   true,
			errSubstr: "must be an array",
		},
		{
			name:      "empty updatableIn array",
			config:    map[string]any{"updatableIn": []any{}},
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
		{
			name:      "non-string in updatableIn array",
			config:    map[string]any{"updatableIn": []any{"Started", 123}},
			wantErr:   true,
			errSubstr: "must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateConfig("testProp", tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
