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

func TestServiceReferenceValidator_Validate(t *testing.T) {
	ctx := context.Background()

	currentServiceID := uuid.New()
	referencedServiceID := uuid.New()
	consumerID := uuid.New()
	groupID := uuid.New()
	serviceTypeID := uuid.New()

	tests := []struct {
		name       string
		newValue   any
		config     map[string]any
		setupMocks func(*MockStore, *MockServiceRepository, *MockServiceTypeRepository)
		wantErr    bool
		errSubstr  string
	}{
		{
			name:     "nil value always passes",
			newValue: nil,
			config:   map[string]any{},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				// No mocks needed - nil value should pass without DB calls
			},
			wantErr: false,
		},
		{
			name:     "valid service reference without constraints",
			newValue: referencedServiceID.String(),
			config:   map[string]any{},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(&Service{
					BaseEntity:    BaseEntity{ID: referencedServiceID},
					ServiceTypeID: serviceTypeID,
					ConsumerID:    consumerID,
					GroupID:       groupID,
				}, nil)
			},
			wantErr: false,
		},
		{
			name:     "valid service reference with matching type",
			newValue: referencedServiceID.String(),
			config:   map[string]any{"types": []any{"disk"}},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(&Service{
					BaseEntity:    BaseEntity{ID: referencedServiceID},
					ServiceTypeID: serviceTypeID,
					ConsumerID:    consumerID,
					GroupID:       groupID,
				}, nil)
				store.EXPECT().ServiceTypeRepo().Return(serviceTypeRepo)
				serviceTypeRepo.EXPECT().Get(ctx, serviceTypeID).Return(&ServiceType{
					BaseEntity: BaseEntity{ID: serviceTypeID},
					Name:       "disk",
				}, nil)
			},
			wantErr: false,
		},
		{
			name:     "valid service reference with one of multiple types",
			newValue: referencedServiceID.String(),
			config:   map[string]any{"types": []any{"disk", "block-storage", "nfs"}},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(&Service{
					BaseEntity:    BaseEntity{ID: referencedServiceID},
					ServiceTypeID: serviceTypeID,
					ConsumerID:    consumerID,
					GroupID:       groupID,
				}, nil)
				store.EXPECT().ServiceTypeRepo().Return(serviceTypeRepo)
				serviceTypeRepo.EXPECT().Get(ctx, serviceTypeID).Return(&ServiceType{
					BaseEntity: BaseEntity{ID: serviceTypeID},
					Name:       "block-storage",
				}, nil)
			},
			wantErr: false,
		},
		{
			name:     "valid service reference with same consumer",
			newValue: referencedServiceID.String(),
			config:   map[string]any{"origin": "consumer"},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(&Service{
					BaseEntity: BaseEntity{ID: referencedServiceID},
					ConsumerID: consumerID,
					GroupID:    uuid.New(), // Different group, but same consumer
				}, nil)
			},
			wantErr: false,
		},
		{
			name:     "valid service reference with same group",
			newValue: referencedServiceID.String(),
			config:   map[string]any{"origin": "group"},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(&Service{
					BaseEntity: BaseEntity{ID: referencedServiceID},
					ConsumerID: uuid.New(), // Different consumer, but same group
					GroupID:    groupID,
				}, nil)
			},
			wantErr: false,
		},
		{
			name:     "service not found",
			newValue: referencedServiceID.String(),
			config:   map[string]any{},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(nil, errors.New("not found"))
			},
			wantErr:   true,
			errSubstr: "referenced service not found",
		},
		{
			name:     "invalid UUID format",
			newValue: "not-a-uuid",
			config:   map[string]any{},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				// No mocks needed - should fail before DB calls
			},
			wantErr:   true,
			errSubstr: "invalid service uuid",
		},
		{
			name:     "wrong type - not string",
			newValue: 12345,
			config:   map[string]any{},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				// No mocks needed - should fail before DB calls
			},
			wantErr:   true,
			errSubstr: "expected string uuid",
		},
		{
			name:     "service type does not match",
			newValue: referencedServiceID.String(),
			config:   map[string]any{"types": []any{"disk"}},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(&Service{
					BaseEntity:    BaseEntity{ID: referencedServiceID},
					ServiceTypeID: serviceTypeID,
				}, nil)
				store.EXPECT().ServiceTypeRepo().Return(serviceTypeRepo)
				serviceTypeRepo.EXPECT().Get(ctx, serviceTypeID).Return(&ServiceType{
					BaseEntity: BaseEntity{ID: serviceTypeID},
					Name:       "vm",
				}, nil)
			},
			wantErr:   true,
			errSubstr: "service must be one of types",
		},
		{
			name:     "different consumer when origin=consumer",
			newValue: referencedServiceID.String(),
			config:   map[string]any{"origin": "consumer"},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(&Service{
					BaseEntity: BaseEntity{ID: referencedServiceID},
					ConsumerID: uuid.New(), // Different from context consumerID
				}, nil)
			},
			wantErr:   true,
			errSubstr: "must belong to the same consumer",
		},
		{
			name:     "different group when origin=group",
			newValue: referencedServiceID.String(),
			config:   map[string]any{"origin": "group"},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(&Service{
					BaseEntity: BaseEntity{ID: referencedServiceID},
					GroupID:    uuid.New(), // Different from context groupID
				}, nil)
			},
			wantErr:   true,
			errSubstr: "must belong to the same service group",
		},
		{
			name:     "types config not an array",
			newValue: referencedServiceID.String(),
			config:   map[string]any{"types": "disk"},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(&Service{
					BaseEntity:    BaseEntity{ID: referencedServiceID},
					ServiceTypeID: serviceTypeID,
				}, nil)
				store.EXPECT().ServiceTypeRepo().Return(serviceTypeRepo)
				serviceTypeRepo.EXPECT().Get(ctx, serviceTypeID).Return(&ServiceType{
					BaseEntity: BaseEntity{ID: serviceTypeID},
					Name:       "disk",
				}, nil)
			},
			wantErr:   true,
			errSubstr: "types config must be an array",
		},
		{
			name:     "origin config not a string",
			newValue: referencedServiceID.String(),
			config:   map[string]any{"origin": 123},
			setupMocks: func(store *MockStore, serviceRepo *MockServiceRepository, serviceTypeRepo *MockServiceTypeRepository) {
				store.EXPECT().ServiceRepo().Return(serviceRepo)
				serviceRepo.EXPECT().Get(ctx, properties.UUID(referencedServiceID)).Return(&Service{
					BaseEntity: BaseEntity{ID: referencedServiceID},
				}, nil)
			},
			wantErr:   true,
			errSubstr: "origin config must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := NewMockStore(t)
			mockServiceRepo := NewMockServiceRepository(t)
			mockServiceTypeRepo := NewMockServiceTypeRepository(t)

			tt.setupMocks(mockStore, mockServiceRepo, mockServiceTypeRepo)

			validator := NewServiceReferenceValidator()
			schemaCtx := ServicePropertyContext{
				Actor:      ActorUser,
				Store:      mockStore,
				ConsumerID: consumerID,
				GroupID:    groupID,
				ServiceID:  &currentServiceID,
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

func TestServiceReferenceValidator_ValidateConfig(t *testing.T) {
	validator := &ServiceReferenceValidator{}

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
			name:    "valid config with single type",
			config:  map[string]any{"types": []any{"disk"}},
			wantErr: false,
		},
		{
			name:    "valid config with multiple types",
			config:  map[string]any{"types": []any{"disk", "block-storage", "nfs"}},
			wantErr: false,
		},
		{
			name:    "valid config with origin consumer",
			config:  map[string]any{"origin": "consumer"},
			wantErr: false,
		},
		{
			name:    "valid config with origin group",
			config:  map[string]any{"origin": "group"},
			wantErr: false,
		},
		{
			name:    "valid config with types and origin",
			config:  map[string]any{"types": []any{"disk"}, "origin": "consumer"},
			wantErr: false,
		},
		{
			name:      "types not an array",
			config:    map[string]any{"types": "disk"},
			wantErr:   true,
			errSubstr: "must be an array",
		},
		{
			name:      "types empty array",
			config:    map[string]any{"types": []any{}},
			wantErr:   true,
			errSubstr: "array cannot be empty",
		},
		{
			name:      "types array contains non-string",
			config:    map[string]any{"types": []any{"disk", 123}},
			wantErr:   true,
			errSubstr: "must contain only strings",
		},
		{
			name:      "origin not a string",
			config:    map[string]any{"origin": 123},
			wantErr:   true,
			errSubstr: "must be a string",
		},
		{
			name:      "origin invalid value",
			config:    map[string]any{"origin": "invalid"},
			wantErr:   true,
			errSubstr: "must be 'consumer' or 'group'",
		},
		{
			name:      "origin null",
			config:    map[string]any{"origin": nil},
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
