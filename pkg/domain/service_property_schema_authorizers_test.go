package domain

import (
	"context"
	"testing"

	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
)

func TestActorAuthorizer_Authorize(t *testing.T) {
	authorizer := &ActorAuthorizer{}
	ctx := context.Background()

	tests := []struct {
		name        string
		schemaCtx   ServicePropertyContext
		operation   schema.Operation
		hasNewValue bool
		config      map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "user actor allowed",
			schemaCtx: ServicePropertyContext{
				Actor: ActorUser,
			},
			operation:   schema.OperationCreate,
			hasNewValue: true,
			config: map[string]any{
				"actors": []any{"user", "agent"},
			},
			wantErr: false,
		},
		{
			name: "agent actor allowed",
			schemaCtx: ServicePropertyContext{
				Actor: ActorAgent,
			},
			operation:   schema.OperationCreate,
			hasNewValue: true,
			config: map[string]any{
				"actors": []any{"user", "agent"},
			},
			wantErr: false,
		},
		{
			name: "system actor allowed",
			schemaCtx: ServicePropertyContext{
				Actor: ActorSystem,
			},
			operation:   schema.OperationCreate,
			hasNewValue: true,
			config: map[string]any{
				"actors": []any{"system"},
			},
			wantErr: false,
		},
		{
			name: "user actor not allowed",
			schemaCtx: ServicePropertyContext{
				Actor: ActorUser,
			},
			operation:   schema.OperationCreate,
			hasNewValue: true,
			config: map[string]any{
				"actors": []any{"agent", "system"},
			},
			wantErr:     true,
			errContains: "can only be set by",
		},
		{
			name: "no value provided - should pass",
			schemaCtx: ServicePropertyContext{
				Actor: ActorUser,
			},
			operation:   schema.OperationCreate,
			hasNewValue: false,
			config: map[string]any{
				"actors": []any{"agent"},
			},
			wantErr: false,
		},
		{
			name: "missing actors config",
			schemaCtx: ServicePropertyContext{
				Actor: ActorUser,
			},
			operation:   schema.OperationCreate,
			hasNewValue: true,
			config:      map[string]any{},
			wantErr:     true,
			errContains: "missing 'actors'",
		},
		{
			name: "actors config not array",
			schemaCtx: ServicePropertyContext{
				Actor: ActorUser,
			},
			operation:   schema.OperationCreate,
			hasNewValue: true,
			config: map[string]any{
				"actors": "user",
			},
			wantErr:     true,
			errContains: "must be an array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authorizer.Authorize(ctx, tt.schemaCtx, tt.operation, "testProp", tt.hasNewValue, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authorize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("Authorize() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestActorAuthorizer_ValidateConfig(t *testing.T) {
	authorizer := &ActorAuthorizer{}

	tests := []struct {
		name        string
		config      map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config - single actor",
			config: map[string]any{
				"actors": []any{"user"},
			},
			wantErr: false,
		},
		{
			name: "valid config - multiple actors",
			config: map[string]any{
				"actors": []any{"user", "agent", "system"},
			},
			wantErr: false,
		},
		{
			name:        "missing actors",
			config:      map[string]any{},
			wantErr:     true,
			errContains: "missing 'actors'",
		},
		{
			name: "actors not array",
			config: map[string]any{
				"actors": "user",
			},
			wantErr:     true,
			errContains: "must be an array",
		},
		{
			name: "empty actors array",
			config: map[string]any{
				"actors": []any{},
			},
			wantErr:     true,
			errContains: "must not be empty",
		},
		{
			name: "invalid actor value",
			config: map[string]any{
				"actors": []any{"user", "invalid"},
			},
			wantErr:     true,
			errContains: "invalid actor 'invalid'",
		},
		{
			name: "non-string actor",
			config: map[string]any{
				"actors": []any{"user", 123},
			},
			wantErr:     true,
			errContains: "must contain only strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authorizer.ValidateConfig("testProp", tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateConfig() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestStateAuthorizer_Authorize(t *testing.T) {
	authorizer := &StateAuthorizer{}
	ctx := context.Background()
	serviceID := uuid.New()

	tests := []struct {
		name        string
		schemaCtx   ServicePropertyContext
		operation   schema.Operation
		hasNewValue bool
		config      map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "update in allowed state",
			schemaCtx: ServicePropertyContext{
				ServiceID:     &serviceID,
				ServiceStatus: "Stopped",
			},
			operation:   schema.OperationUpdate,
			hasNewValue: true,
			config: map[string]any{
				"allowedStates": []any{"Stopped", "New"},
			},
			wantErr: false,
		},
		{
			name: "update in different allowed state",
			schemaCtx: ServicePropertyContext{
				ServiceID:     &serviceID,
				ServiceStatus: "New",
			},
			operation:   schema.OperationUpdate,
			hasNewValue: true,
			config: map[string]any{
				"allowedStates": []any{"Stopped", "New"},
			},
			wantErr: false,
		},
		{
			name: "update in disallowed state",
			schemaCtx: ServicePropertyContext{
				ServiceID:     &serviceID,
				ServiceStatus: "Started",
			},
			operation:   schema.OperationUpdate,
			hasNewValue: true,
			config: map[string]any{
				"allowedStates": []any{"Stopped", "New"},
			},
			wantErr:     true,
			errContains: "cannot be updated in state 'Started'",
		},
		{
			name: "create operation - should pass",
			schemaCtx: ServicePropertyContext{
				ServiceStatus: "Started",
			},
			operation:   schema.OperationCreate,
			hasNewValue: true,
			config: map[string]any{
				"allowedStates": []any{"Stopped"},
			},
			wantErr: false,
		},
		{
			name: "no value provided - should pass",
			schemaCtx: ServicePropertyContext{
				ServiceID:     &serviceID,
				ServiceStatus: "Started",
			},
			operation:   schema.OperationUpdate,
			hasNewValue: false,
			config: map[string]any{
				"allowedStates": []any{"Stopped"},
			},
			wantErr: false,
		},
		{
			name: "missing service status on update",
			schemaCtx: ServicePropertyContext{
				ServiceID: &serviceID,
			},
			operation:   schema.OperationUpdate,
			hasNewValue: true,
			config: map[string]any{
				"allowedStates": []any{"Stopped"},
			},
			wantErr:     true,
			errContains: "requires service status",
		},
		{
			name: "missing allowedStates config",
			schemaCtx: ServicePropertyContext{
				ServiceID:     &serviceID,
				ServiceStatus: "Stopped",
			},
			operation:   schema.OperationUpdate,
			hasNewValue: true,
			config:      map[string]any{},
			wantErr:     true,
			errContains: "missing 'allowedStates'",
		},
		{
			name: "allowedStates not array",
			schemaCtx: ServicePropertyContext{
				ServiceID:     &serviceID,
				ServiceStatus: "Stopped",
			},
			operation:   schema.OperationUpdate,
			hasNewValue: true,
			config: map[string]any{
				"allowedStates": "Stopped",
			},
			wantErr:     true,
			errContains: "must be an array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authorizer.Authorize(ctx, tt.schemaCtx, tt.operation, "testProp", tt.hasNewValue, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Authorize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("Authorize() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestStateAuthorizer_ValidateConfig(t *testing.T) {
	authorizer := &StateAuthorizer{}

	tests := []struct {
		name        string
		config      map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config - single state",
			config: map[string]any{
				"allowedStates": []any{"Stopped"},
			},
			wantErr: false,
		},
		{
			name: "valid config - multiple states",
			config: map[string]any{
				"allowedStates": []any{"Stopped", "New", "Started"},
			},
			wantErr: false,
		},
		{
			name:        "missing allowedStates",
			config:      map[string]any{},
			wantErr:     true,
			errContains: "missing 'allowedStates'",
		},
		{
			name: "allowedStates not array",
			config: map[string]any{
				"allowedStates": "Stopped",
			},
			wantErr:     true,
			errContains: "must be an array",
		},
		{
			name: "empty allowedStates array",
			config: map[string]any{
				"allowedStates": []any{},
			},
			wantErr:     true,
			errContains: "must not be empty",
		},
		{
			name: "non-string state",
			config: map[string]any{
				"allowedStates": []any{"Stopped", 123},
			},
			wantErr:     true,
			errContains: "must contain only strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authorizer.ValidateConfig("testProp", tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateConfig() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}
