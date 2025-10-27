// Tests for pool generator
package domain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestSchemaPoolGenerator_Generate(t *testing.T) {
	ctx := context.Background()

	poolSetID := uuid.New()
	serviceID := uuid.New()
	poolID := uuid.New()

	tests := []struct {
		name             string
		config           map[string]any
		setupMock        func(*MockStore)
		servicePoolSetID *uuid.UUID
		serviceID        *uuid.UUID
		wantValue        any
		wantGen          bool
		wantErr          bool
		errSubstr        string
	}{
		{
			name:             "successful allocation",
			config:           map[string]any{"poolType": "public_ip"},
			servicePoolSetID: &poolSetID,
			serviceID:        &serviceID,
			setupMock: func(store *MockStore) {
				poolRepo := NewMockServicePoolRepository(t)
				valueRepo := NewMockServicePoolValueRepository(t)

				pool := &ServicePool{
					BaseEntity:    BaseEntity{ID: poolID},
					Type:          "public_ip",
					PropertyType:  "string",
					GeneratorType: PoolGeneratorList,
				}

				poolRepo.On("ListByPoolSet", ctx, poolSetID).Return([]*ServicePool{pool}, nil)
				valueRepo.On("FindAvailable", ctx, poolID).Return([]*ServicePoolValue{
					{BaseEntity: BaseEntity{ID: uuid.New()}, Value: "192.168.1.10"},
				}, nil)
				valueRepo.On("Update", ctx, mock.AnythingOfType("*domain.ServicePoolValue")).Return(nil)

				store.On("ServicePoolRepo").Return(poolRepo)
				store.On("ServicePoolValueRepo").Return(valueRepo)
			},
			wantValue: "192.168.1.10",
			wantGen:   true,
			wantErr:   false,
		},
		{
			name:             "missing poolType config",
			config:           map[string]any{},
			servicePoolSetID: &poolSetID,
			serviceID:        &serviceID,
			setupMock:        func(store *MockStore) {},
			wantErr:          true,
			errSubstr:        "missing 'poolType'",
		},
		{
			name:             "poolType not a string",
			config:           map[string]any{"poolType": 123},
			servicePoolSetID: &poolSetID,
			serviceID:        &serviceID,
			setupMock:        func(store *MockStore) {},
			wantErr:          true,
			errSubstr:        "must be a string",
		},
		{
			name:             "service ID required",
			config:           map[string]any{"poolType": "public_ip"},
			servicePoolSetID: &poolSetID,
			serviceID:        nil,
			setupMock:        func(store *MockStore) {},
			wantErr:          true,
			errSubstr:        "service ID required",
		},
		{
			name:             "agent without pool set",
			config:           map[string]any{"poolType": "public_ip"},
			servicePoolSetID: nil,
			serviceID:        &serviceID,
			setupMock:        func(store *MockStore) {},
			wantErr:          true,
			errSubstr:        "does not have a pool set",
		},
		{
			name:             "pool type not found in pool set",
			config:           map[string]any{"poolType": "nonexistent"},
			servicePoolSetID: &poolSetID,
			serviceID:        &serviceID,
			setupMock: func(store *MockStore) {
				poolRepo := NewMockServicePoolRepository(t)

				pool := &ServicePool{
					BaseEntity:    BaseEntity{ID: poolID},
					Type:          "public_ip", // Different type
					PropertyType:  "string",
					GeneratorType: PoolGeneratorList,
				}

				poolRepo.On("ListByPoolSet", ctx, poolSetID).Return([]*ServicePool{pool}, nil)
				store.On("ServicePoolRepo").Return(poolRepo)
			},
			wantErr:   true,
			errSubstr: "no pool found with type 'nonexistent'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore(t)
			tt.setupMock(store)

			generator := NewSchemaPoolGenerator()

			schemaCtx := ServicePropertyContext{
				Actor:            ActorUser,
				Store:            store,
				ServicePoolSetID: tt.servicePoolSetID,
				ServiceID:        tt.serviceID,
			}

			value, generated, err := generator.Generate(ctx, schemaCtx, "testProp", nil, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !stringContains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if generated != tt.wantGen {
					t.Errorf("expected generated=%v, got %v", tt.wantGen, generated)
				}
				if value != tt.wantValue {
					t.Errorf("expected value=%v, got %v", tt.wantValue, value)
				}
			}
		})
	}
}

func TestSchemaPoolGenerator_ValidateConfig(t *testing.T) {
	generator := &SchemaPoolGenerator{}

	tests := []struct {
		name      string
		config    map[string]any
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid config",
			config:  map[string]any{"poolType": "public_ip"},
			wantErr: false,
		},
		{
			name:      "missing poolType",
			config:    map[string]any{},
			wantErr:   true,
			errSubstr: "missing 'poolType'",
		},
		{
			name:      "poolType not a string",
			config:    map[string]any{"poolType": 123},
			wantErr:   true,
			errSubstr: "must be a string",
		},
		{
			name:      "empty poolType",
			config:    map[string]any{"poolType": ""},
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.ValidateConfig("testProp", tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errSubstr)
				} else if tt.errSubstr != "" && !stringContains(err.Error(), tt.errSubstr) {
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
