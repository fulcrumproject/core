// Tests for domain-specific validators
package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/fulcrumproject/core/pkg/properties"
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
				Actor: tt.actor,
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
				Actor: ActorUser,
			}
			if tt.service != nil {
				schemaCtx.ServiceStatus = tt.service.Status
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

func TestServiceOptionValidator_Validate(t *testing.T) {
	ctx := context.Background()

	providerID := uuid.New()
	optionTypeID := uuid.New()
	serviceID := uuid.New()

	tests := []struct {
		name       string
		newValue   any
		config     map[string]any
		service    *Service
		setupMocks func(*MockStore, *MockServiceOptionTypeRepository, *MockServiceOptionRepository)
		wantErr    bool
		errSubstr  string
	}{
		{
			name:     "nil value always passes",
			newValue: nil,
			config:   map[string]any{"value": "os"},
			service:  &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: providerID},
			setupMocks: func(store *MockStore, optionTypeRepo *MockServiceOptionTypeRepository, optionRepo *MockServiceOptionRepository) {
				// No mocks needed - nil value should pass without DB calls
			},
			wantErr: false,
		},
		{
			name:     "valid string value matches enabled option",
			newValue: "ubuntu:20.04",
			config:   map[string]any{"value": "os"},
			service:  &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: providerID},
			setupMocks: func(store *MockStore, optionTypeRepo *MockServiceOptionTypeRepository, optionRepo *MockServiceOptionRepository) {
				store.EXPECT().ServiceOptionTypeRepo().Return(optionTypeRepo)
				optionTypeRepo.EXPECT().FindByType(ctx, "os").Return(&ServiceOptionType{
					BaseEntity: BaseEntity{ID: optionTypeID},
					Type:       "os",
				}, nil)

				store.EXPECT().ServiceOptionRepo().Return(optionRepo)
				optionRepo.EXPECT().ListByProviderAndType(ctx, providerID, optionTypeID).Return([]*ServiceOption{
					{Enabled: true, Value: "ubuntu:20.04"},
					{Enabled: false, Value: "ubuntu:22.04"}, // Disabled option
				}, nil)
			},
			wantErr: false,
		},
		{
			name:     "valid complex JSON value matches enabled option",
			newValue: map[string]any{"image": "ubuntu:20.04", "arch": "amd64"},
			config:   map[string]any{"value": "os"},
			service:  &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: providerID},
			setupMocks: func(store *MockStore, optionTypeRepo *MockServiceOptionTypeRepository, optionRepo *MockServiceOptionRepository) {
				store.EXPECT().ServiceOptionTypeRepo().Return(optionTypeRepo)
				optionTypeRepo.EXPECT().FindByType(ctx, "os").Return(&ServiceOptionType{
					BaseEntity: BaseEntity{ID: optionTypeID},
					Type:       "os",
				}, nil)

				store.EXPECT().ServiceOptionRepo().Return(optionRepo)
				optionRepo.EXPECT().ListByProviderAndType(ctx, providerID, optionTypeID).Return([]*ServiceOption{
					{Enabled: true, Value: map[string]any{"image": "ubuntu:20.04", "arch": "amd64"}},
					{Enabled: true, Value: "simple-value"},
				}, nil)
			},
			wantErr: false,
		},
		{
			name:     "invalid value does not match any enabled option",
			newValue: "windows:2022",
			config:   map[string]any{"value": "os"},
			service:  &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: providerID},
			setupMocks: func(store *MockStore, optionTypeRepo *MockServiceOptionTypeRepository, optionRepo *MockServiceOptionRepository) {
				store.EXPECT().ServiceOptionTypeRepo().Return(optionTypeRepo)
				optionTypeRepo.EXPECT().FindByType(ctx, "os").Return(&ServiceOptionType{
					BaseEntity: BaseEntity{ID: optionTypeID},
					Type:       "os",
				}, nil)

				store.EXPECT().ServiceOptionRepo().Return(optionRepo)
				optionRepo.EXPECT().ListByProviderAndType(ctx, providerID, optionTypeID).Return([]*ServiceOption{
					{Enabled: true, Value: "ubuntu:20.04"},
					{Enabled: true, Value: "ubuntu:22.04"},
				}, nil)
			},
			wantErr:   true,
			errSubstr: "must match one of the enabled service options",
		},
		{
			name:     "no enabled options available",
			newValue: "ubuntu:20.04",
			config:   map[string]any{"value": "os"},
			service:  &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: providerID},
			setupMocks: func(store *MockStore, optionTypeRepo *MockServiceOptionTypeRepository, optionRepo *MockServiceOptionRepository) {
				store.EXPECT().ServiceOptionTypeRepo().Return(optionTypeRepo)
				optionTypeRepo.EXPECT().FindByType(ctx, "os").Return(&ServiceOptionType{
					BaseEntity: BaseEntity{ID: optionTypeID},
					Type:       "os",
				}, nil)

				store.EXPECT().ServiceOptionRepo().Return(optionRepo)
				optionRepo.EXPECT().ListByProviderAndType(ctx, providerID, optionTypeID).Return([]*ServiceOption{
					{Enabled: false, Value: "ubuntu:20.04"},
					{Enabled: false, Value: "ubuntu:22.04"},
				}, nil)
			},
			wantErr:   true,
			errSubstr: "no enabled service options available",
		},
		{
			name:     "service option type not found",
			newValue: "ubuntu:20.04",
			config:   map[string]any{"value": "nonexistent"},
			service:  &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: providerID},
			setupMocks: func(store *MockStore, optionTypeRepo *MockServiceOptionTypeRepository, optionRepo *MockServiceOptionRepository) {
				store.EXPECT().ServiceOptionTypeRepo().Return(optionTypeRepo)
				optionTypeRepo.EXPECT().FindByType(ctx, "nonexistent").Return(nil, errors.New("not found"))
			},
			wantErr:   true,
			errSubstr: "failed to find service option type",
		},
		{
			name:     "error retrieving service options",
			newValue: "ubuntu:20.04",
			config:   map[string]any{"value": "os"},
			service:  &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: providerID},
			setupMocks: func(store *MockStore, optionTypeRepo *MockServiceOptionTypeRepository, optionRepo *MockServiceOptionRepository) {
				store.EXPECT().ServiceOptionTypeRepo().Return(optionTypeRepo)
				optionTypeRepo.EXPECT().FindByType(ctx, "os").Return(&ServiceOptionType{
					BaseEntity: BaseEntity{ID: optionTypeID},
					Type:       "os",
				}, nil)

				store.EXPECT().ServiceOptionRepo().Return(optionRepo)
				optionRepo.EXPECT().ListByProviderAndType(ctx, providerID, optionTypeID).Return(nil, errors.New("database error"))
			},
			wantErr:   true,
			errSubstr: "failed to retrieve service options",
		},
		{
			name:     "missing value in config",
			newValue: "ubuntu:20.04",
			config:   map[string]any{},
			service:  &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: providerID},
			setupMocks: func(store *MockStore, optionTypeRepo *MockServiceOptionTypeRepository, optionRepo *MockServiceOptionRepository) {
				// No mocks needed - should fail before DB calls
			},
			wantErr:   true,
			errSubstr: "missing 'value'",
		},
		{
			name:     "config value not a string",
			newValue: "ubuntu:20.04",
			config:   map[string]any{"value": 123},
			service:  &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: providerID},
			setupMocks: func(store *MockStore, optionTypeRepo *MockServiceOptionTypeRepository, optionRepo *MockServiceOptionRepository) {
				// No mocks needed - should fail before DB calls
			},
			wantErr:   true,
			errSubstr: "must be a string",
		},
		{
			name:     "skips nil option values",
			newValue: "ubuntu:20.04",
			config:   map[string]any{"value": "os"},
			service:  &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: providerID},
			setupMocks: func(store *MockStore, optionTypeRepo *MockServiceOptionTypeRepository, optionRepo *MockServiceOptionRepository) {
				store.EXPECT().ServiceOptionTypeRepo().Return(optionTypeRepo)
				optionTypeRepo.EXPECT().FindByType(ctx, "os").Return(&ServiceOptionType{
					BaseEntity: BaseEntity{ID: optionTypeID},
					Type:       "os",
				}, nil)

				store.EXPECT().ServiceOptionRepo().Return(optionRepo)
				optionRepo.EXPECT().ListByProviderAndType(ctx, providerID, optionTypeID).Return([]*ServiceOption{
					{Enabled: true, Value: nil}, // Should be skipped
					{Enabled: true, Value: "ubuntu:20.04"},
				}, nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := NewMockStore(t)
			mockOptionTypeRepo := NewMockServiceOptionTypeRepository(t)
			mockOptionRepo := NewMockServiceOptionRepository(t)

			tt.setupMocks(mockStore, mockOptionTypeRepo, mockOptionRepo)

			validator := NewServiceOptionValidator()
			schemaCtx := ServicePropertyContext{
				Actor: ActorUser,
				Store: mockStore,
			}
			if tt.service != nil {
				schemaCtx.ProviderID = tt.service.ProviderID
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

func TestServiceOptionValidator_ValidateConfig(t *testing.T) {
	validator := &ServiceOptionValidator{}

	tests := []struct {
		name      string
		config    map[string]any
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "valid config with value",
			config:  map[string]any{"value": "os"},
			wantErr: false,
		},
		{
			name:    "valid config with different value",
			config:  map[string]any{"value": "machine_type"},
			wantErr: false,
		},
		{
			name:      "empty config",
			config:    map[string]any{},
			wantErr:   true,
			errSubstr: "missing 'value'",
		},
		{
			name:      "missing value key",
			config:    map[string]any{"other": "something"},
			wantErr:   true,
			errSubstr: "missing 'value'",
		},
		{
			name:      "value not a string",
			config:    map[string]any{"value": 123},
			wantErr:   true,
			errSubstr: "must be a string",
		},
		{
			name:      "value is empty string",
			config:    map[string]any{"value": ""},
			wantErr:   true,
			errSubstr: "cannot be empty",
		},
		{
			name:      "value is null",
			config:    map[string]any{"value": nil},
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

func TestValuesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        any
		b        any
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "one nil",
			a:        "value",
			b:        nil,
			expected: false,
		},
		{
			name:     "simple string equality",
			a:        "ubuntu:20.04",
			b:        "ubuntu:20.04",
			expected: true,
		},
		{
			name:     "simple string inequality",
			a:        "ubuntu:20.04",
			b:        "ubuntu:22.04",
			expected: false,
		},
		{
			name:     "simple integer equality",
			a:        42,
			b:        42,
			expected: true,
		},
		{
			name:     "map equality",
			a:        map[string]any{"image": "ubuntu:20.04", "arch": "amd64"},
			b:        map[string]any{"image": "ubuntu:20.04", "arch": "amd64"},
			expected: true,
		},
		{
			name:     "map inequality - different values",
			a:        map[string]any{"image": "ubuntu:20.04", "arch": "amd64"},
			b:        map[string]any{"image": "ubuntu:22.04", "arch": "amd64"},
			expected: false,
		},
		{
			name:     "map inequality - different keys",
			a:        map[string]any{"image": "ubuntu:20.04"},
			b:        map[string]any{"image": "ubuntu:20.04", "arch": "amd64"},
			expected: false,
		},
		{
			name:     "slice equality",
			a:        []any{"item1", "item2"},
			b:        []any{"item1", "item2"},
			expected: true,
		},
		{
			name:     "slice inequality - different order",
			a:        []any{"item1", "item2"},
			b:        []any{"item2", "item1"},
			expected: false,
		},
		{
			name:     "properties.UUID equality",
			a:        properties.UUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")),
			b:        properties.UUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valuesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("valuesEqual(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
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
