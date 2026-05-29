package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

func TestSchemaConfigPoolGenerator_Generate(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	agentID := properties.UUID(uuid.New())
	providerID := properties.UUID(uuid.New())

	matchProvider := mock.MatchedBy(func(p *properties.UUID) bool {
		return p != nil && *p == providerID
	})

	tests := []struct {
		name         string
		config       map[string]any
		currentValue any
		agentID      *properties.UUID
		withStore    bool
		setupMock    func(*MockStore)
		wantValue    any
		wantGen      bool
		wantErr      bool
		errSubstr    string
	}{
		{
			name:         "resolves pool by type and allocates",
			config:       map[string]any{"poolType": "public_ip"},
			currentValue: nil,
			agentID:      &agentID,
			withStore:    true,
			setupMock: func(store *MockStore) {
				poolRepo := NewMockConfigPoolRepository(t)
				valueRepo := NewMockConfigPoolValueRepository(t)

				pool := &ConfigPool{
					BaseEntity:    BaseEntity{ID: poolID},
					Type:          "public_ip",
					PropertyType:  "string",
					GeneratorType: PoolGeneratorList,
				}
				poolRepo.On("FindByTypeAndProvider", ctx, "public_ip", matchProvider).Return(pool, nil)
				valueRepo.On("FindAvailable", ctx, poolID).Return([]*ConfigPoolValue{
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: poolID, Value: "192.168.1.10"},
				}, nil)
				valueRepo.On("Update", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
					return v.AgentID != nil && *v.AgentID == agentID &&
						v.PropertyName != nil && *v.PropertyName == "testProp" &&
						v.AllocatedAt != nil
				})).Return(nil)

				store.On("ConfigPoolRepo").Return(poolRepo)
				store.On("ConfigPoolValueRepo").Return(valueRepo)
			},
			wantValue: "192.168.1.10",
			wantGen:   true,
		},
		{
			name:         "surfaces error from pool lookup",
			config:       map[string]any{"poolType": "public_ip"},
			currentValue: nil,
			agentID:      &agentID,
			withStore:    true,
			setupMock: func(store *MockStore) {
				poolRepo := NewMockConfigPoolRepository(t)
				poolRepo.On("FindByTypeAndProvider", ctx, "public_ip", matchProvider).
					Return(nil, errors.New("db down"))
				store.On("ConfigPoolRepo").Return(poolRepo)
			},
			wantErr:   true,
			errSubstr: "db down",
		},
		{
			name:         "surfaces NotFound when no pool of type exists",
			config:       map[string]any{"poolType": "public_ip"},
			currentValue: nil,
			agentID:      &agentID,
			withStore:    true,
			setupMock: func(store *MockStore) {
				poolRepo := NewMockConfigPoolRepository(t)
				poolRepo.On("FindByTypeAndProvider", ctx, "public_ip", matchProvider).
					Return(nil, NewNotFoundErrorf("no config pool with type public_ip for provider"))
				store.On("ConfigPoolRepo").Return(poolRepo)
			},
			wantErr:   true,
			errSubstr: "public_ip",
		},
		{
			name:         "skip when value exists",
			config:       map[string]any{"poolType": "public_ip"},
			currentValue: "already-set",
			agentID:      &agentID,
			withStore:    true,
			setupMock:    func(store *MockStore) {},
			wantValue:    "already-set",
			wantGen:      false,
		},
		{
			name:         "missing poolType",
			config:       map[string]any{},
			currentValue: nil,
			agentID:      &agentID,
			withStore:    true,
			setupMock:    func(store *MockStore) {},
			wantErr:      true,
			errSubstr:    "missing 'poolType'",
		},
		{
			name:         "poolType not a string",
			config:       map[string]any{"poolType": 42},
			currentValue: nil,
			agentID:      &agentID,
			withStore:    true,
			setupMock:    func(store *MockStore) {},
			wantErr:      true,
			errSubstr:    "must be a string",
		},
		{
			name:         "poolType empty string",
			config:       map[string]any{"poolType": ""},
			currentValue: nil,
			agentID:      &agentID,
			withStore:    true,
			setupMock:    func(store *MockStore) {},
			wantErr:      true,
			errSubstr:    "cannot be empty",
		},
		{
			name:         "missing agent ID",
			config:       map[string]any{"poolType": "public_ip"},
			currentValue: nil,
			agentID:      nil,
			withStore:    true,
			setupMock:    func(store *MockStore) {},
			wantErr:      true,
			errSubstr:    "entity ID required",
		},
		{
			name:         "missing store",
			config:       map[string]any{"poolType": "public_ip"},
			currentValue: nil,
			agentID:      &agentID,
			withStore:    false,
			setupMock:    func(store *MockStore) {},
			wantErr:      true,
			errSubstr:    "missing store",
		},
		{
			name:         "no available values",
			config:       map[string]any{"poolType": "public_ip"},
			currentValue: nil,
			agentID:      &agentID,
			withStore:    true,
			setupMock: func(store *MockStore) {
				poolRepo := NewMockConfigPoolRepository(t)
				valueRepo := NewMockConfigPoolValueRepository(t)

				pool := &ConfigPool{
					BaseEntity:    BaseEntity{ID: poolID},
					Type:          "public_ip",
					PropertyType:  "string",
					GeneratorType: PoolGeneratorList,
				}
				poolRepo.On("FindByTypeAndProvider", ctx, "public_ip", matchProvider).Return(pool, nil)
				valueRepo.On("FindAvailable", ctx, poolID).Return([]*ConfigPoolValue{}, nil)

				store.On("ConfigPoolRepo").Return(poolRepo)
				store.On("ConfigPoolValueRepo").Return(valueRepo)
			},
			wantErr:   true,
			errSubstr: "no available values",
		},
		{
			name:         "Update errors",
			config:       map[string]any{"poolType": "public_ip"},
			currentValue: nil,
			agentID:      &agentID,
			withStore:    true,
			setupMock: func(store *MockStore) {
				poolRepo := NewMockConfigPoolRepository(t)
				valueRepo := NewMockConfigPoolValueRepository(t)

				pool := &ConfigPool{
					BaseEntity:    BaseEntity{ID: poolID},
					Type:          "public_ip",
					PropertyType:  "string",
					GeneratorType: PoolGeneratorList,
				}
				poolRepo.On("FindByTypeAndProvider", ctx, "public_ip", matchProvider).Return(pool, nil)
				valueRepo.On("FindAvailable", ctx, poolID).Return([]*ConfigPoolValue{
					{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: poolID, Value: "x"},
				}, nil)
				valueRepo.On("Update", ctx, mock.AnythingOfType("*domain.ConfigPoolValue")).Return(errors.New("update boom"))

				store.On("ConfigPoolRepo").Return(poolRepo)
				store.On("ConfigPoolValueRepo").Return(valueRepo)
			},
			wantErr:   true,
			errSubstr: "update boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var store *MockStore
			schemaCtx := AgentConfigContext{AgentID: tt.agentID, AgentProviderID: providerID}
			if tt.withStore {
				store = NewMockStore(t)
				tt.setupMock(store)
				schemaCtx.Store = store
			}

			gen := NewSchemaConfigPoolGenerator[AgentConfigContext]()
			got, generated, err := gen.Generate(ctx, schemaCtx, "testProp", tt.currentValue, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !stringContains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if generated != tt.wantGen {
				t.Errorf("expected generated=%v, got %v", tt.wantGen, generated)
			}
			if got != tt.wantValue {
				t.Errorf("expected value=%v, got %v", tt.wantValue, got)
			}
		})
	}
}

// TestSchemaConfigPoolGenerator_Generate_Infrastructure proves the same generic
// generator works for the infrastructure context: it resolves the pool by the
// infrastructure's provider and stamps the allocated value with the infrastructure id.
func TestSchemaConfigPoolGenerator_Generate_Infrastructure(t *testing.T) {
	ctx := context.Background()
	poolID := properties.UUID(uuid.New())
	infraID := properties.UUID(uuid.New())
	providerID := properties.UUID(uuid.New())

	matchProvider := mock.MatchedBy(func(p *properties.UUID) bool {
		return p != nil && *p == providerID
	})

	store := NewMockStore(t)
	poolRepo := NewMockConfigPoolRepository(t)
	valueRepo := NewMockConfigPoolValueRepository(t)

	pool := &ConfigPool{
		BaseEntity:    BaseEntity{ID: poolID},
		Type:          "ptp_pool",
		PropertyType:  "string",
		GeneratorType: PoolGeneratorList,
	}
	poolRepo.On("FindByTypeAndProvider", ctx, "ptp_pool", matchProvider).Return(pool, nil)
	valueRepo.On("FindAvailable", ctx, poolID).Return([]*ConfigPoolValue{
		{BaseEntity: BaseEntity{ID: properties.UUID(uuid.New())}, ConfigPoolID: poolID, Value: "10.0.0.1"},
	}, nil)
	valueRepo.On("Update", ctx, mock.MatchedBy(func(v *ConfigPoolValue) bool {
		return v.AgentID != nil && *v.AgentID == infraID &&
			v.PropertyName != nil && *v.PropertyName == "ptp" &&
			v.AllocatedAt != nil
	})).Return(nil)
	store.On("ConfigPoolRepo").Return(poolRepo)
	store.On("ConfigPoolValueRepo").Return(valueRepo)

	schemaCtx := InfrastructureConfigContext{
		Store:                    store,
		InfrastructureID:         &infraID,
		InfrastructureProviderID: providerID,
	}

	gen := NewSchemaConfigPoolGenerator[InfrastructureConfigContext]()
	got, generated, err := gen.Generate(ctx, schemaCtx, "ptp", nil, map[string]any{"poolType": "ptp_pool"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !generated {
		t.Errorf("expected generated=true")
	}
	if got != "10.0.0.1" {
		t.Errorf("expected value=10.0.0.1, got %v", got)
	}
}

func TestSchemaConfigPoolGenerator_ValidateConfig(t *testing.T) {
	gen := NewSchemaConfigPoolGenerator[AgentConfigContext]()

	tests := []struct {
		name      string
		config    map[string]any
		wantErr   bool
		errSubstr string
	}{
		{name: "valid", config: map[string]any{"poolType": "foo"}},
		{name: "missing poolType", config: map[string]any{}, wantErr: true, errSubstr: "missing 'poolType'"},
		{name: "poolType not a string", config: map[string]any{"poolType": 1}, wantErr: true, errSubstr: "must be a string"},
		{name: "empty poolType", config: map[string]any{"poolType": ""}, wantErr: true, errSubstr: "cannot be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.ValidateConfig("testProp", tt.config)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if tt.errSubstr != "" && !stringContains(err.Error(), tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
