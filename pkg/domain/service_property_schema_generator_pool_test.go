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
	agentID := uuid.New()
	poolID := uuid.New()

	tests := []struct {
		name      string
		config    map[string]any
		setupMock func(*MockStore)
		service   *Service
		wantValue any
		wantGen   bool
		wantErr   bool
		errSubstr string
	}{
		{
			name:   "successful allocation",
			config: map[string]any{"poolType": "public_ip"},
			service: &Service{
				BaseEntity: BaseEntity{ID: serviceID},
				AgentID:    agentID,
				Agent: &Agent{
					BaseEntity:       BaseEntity{ID: agentID},
					ServicePoolSetID: &poolSetID,
				},
			},
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
			name:      "missing poolType config",
			config:    map[string]any{},
			service:   &Service{BaseEntity: BaseEntity{ID: serviceID}},
			setupMock: func(store *MockStore) {},
			wantErr:   true,
			errSubstr: "missing 'poolType'",
		},
		{
			name:      "poolType not a string",
			config:    map[string]any{"poolType": 123},
			service:   &Service{BaseEntity: BaseEntity{ID: serviceID}},
			setupMock: func(store *MockStore) {},
			wantErr:   true,
			errSubstr: "must be a string",
		},
		{
			name:      "service context required",
			config:    map[string]any{"poolType": "public_ip"},
			service:   nil,
			setupMock: func(store *MockStore) {},
			wantErr:   true,
			errSubstr: "requires service context",
		},
		{
			name:   "agent lazy loading",
			config: map[string]any{"poolType": "public_ip"},
			service: &Service{
				BaseEntity: BaseEntity{ID: serviceID},
				AgentID:    agentID,
				Agent:      nil, // Not loaded yet
			},
			setupMock: func(store *MockStore) {
				agentRepo := NewMockAgentRepository(t)
				poolRepo := NewMockServicePoolRepository(t)
				valueRepo := NewMockServicePoolValueRepository(t)

				agent := &Agent{
					BaseEntity:       BaseEntity{ID: agentID},
					ServicePoolSetID: &poolSetID,
				}

				pool := &ServicePool{
					BaseEntity:    BaseEntity{ID: poolID},
					Type:          "public_ip",
					PropertyType:  "string",
					GeneratorType: PoolGeneratorList,
				}

				agentRepo.On("Get", ctx, agentID).Return(agent, nil)
				poolRepo.On("ListByPoolSet", ctx, poolSetID).Return([]*ServicePool{pool}, nil)
				valueRepo.On("FindAvailable", ctx, poolID).Return([]*ServicePoolValue{
					{BaseEntity: BaseEntity{ID: uuid.New()}, Value: "10.0.0.1"},
				}, nil)
				valueRepo.On("Update", ctx, mock.AnythingOfType("*domain.ServicePoolValue")).Return(nil)

				store.On("AgentRepo").Return(agentRepo)
				store.On("ServicePoolRepo").Return(poolRepo)
				store.On("ServicePoolValueRepo").Return(valueRepo)
			},
			wantValue: "10.0.0.1",
			wantGen:   true,
			wantErr:   false,
		},
		{
			name:   "agent without pool set",
			config: map[string]any{"poolType": "public_ip"},
			service: &Service{
				BaseEntity: BaseEntity{ID: serviceID},
				AgentID:    agentID,
				Agent: &Agent{
					BaseEntity:       BaseEntity{ID: agentID},
					ServicePoolSetID: nil,
				},
			},
			setupMock: func(store *MockStore) {},
			wantErr:   true,
			errSubstr: "does not have a pool set",
		},
		{
			name:   "pool type not found in pool set",
			config: map[string]any{"poolType": "nonexistent"},
			service: &Service{
				BaseEntity: BaseEntity{ID: serviceID},
				AgentID:    agentID,
				Agent: &Agent{
					BaseEntity:       BaseEntity{ID: agentID},
					ServicePoolSetID: &poolSetID,
				},
			},
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

			generator := NewSchemaPoolGenerator(store)

			schemaCtx := ServicePropertyContext{
				Actor:   ActorUser,
				Service: tt.service,
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
