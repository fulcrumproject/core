package domain

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock store for testing
type mockStore struct {
	serviceTypeRepo  *mockServiceTypeRepo
	serviceRepo      *mockServiceRepo
	serviceGroupRepo *mockServiceGroupRepo
}

func (m *mockStore) ServiceTypeRepo() ServiceTypeRepository {
	return m.serviceTypeRepo
}

func (m *mockStore) ServiceRepo() ServiceRepository {
	return m.serviceRepo
}

func (m *mockStore) ServiceGroupRepo() ServiceGroupRepository {
	return m.serviceGroupRepo
}

func (m *mockStore) Atomic(ctx context.Context, fn func(store Store) error) error {
	return fn(m)
}

// Implement remaining Store interface methods
func (m *mockStore) AgentTypeRepo() AgentTypeRepository                 { return nil }
func (m *mockStore) AgentRepo() AgentRepository                         { return nil }
func (m *mockStore) TokenRepo() TokenRepository                         { return nil }
func (m *mockStore) JobRepo() JobRepository                             { return nil }
func (m *mockStore) EventRepo() EventRepository                         { return nil }
func (m *mockStore) EventSubscriptionRepo() EventSubscriptionRepository { return nil }
func (m *mockStore) MetricTypeRepo() MetricTypeRepository               { return nil }
func (m *mockStore) ParticipantRepo() ParticipantRepository             { return nil }

// Mock service type repo
type mockServiceTypeRepo struct {
	serviceTypes map[properties.UUID]*ServiceType
}

func (m *mockServiceTypeRepo) Get(ctx context.Context, id properties.UUID) (*ServiceType, error) {
	if st, exists := m.serviceTypes[id]; exists {
		return st, nil
	}
	return nil, NewNotFoundErrorf("ServiceType with ID %s not found", id.String())
}

// Implement BaseEntityQuerier methods
func (m *mockServiceTypeRepo) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	_, exists := m.serviceTypes[id]
	return exists, nil
}

func (m *mockServiceTypeRepo) List(ctx context.Context, scope *auth.IdentityScope, req *PageReq) (*PageRes[ServiceType], error) {
	return &PageRes[ServiceType]{}, nil
}

func (m *mockServiceTypeRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(m.serviceTypes)), nil
}

func (m *mockServiceTypeRepo) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return &auth.DefaultObjectScope{}, nil
}

// Implement BaseEntityRepository methods
func (m *mockServiceTypeRepo) Create(ctx context.Context, entity *ServiceType) error {
	m.serviceTypes[entity.ID] = entity
	return nil
}

func (m *mockServiceTypeRepo) Save(ctx context.Context, entity *ServiceType) error {
	m.serviceTypes[entity.ID] = entity
	return nil
}

func (m *mockServiceTypeRepo) Delete(ctx context.Context, id properties.UUID) error {
	delete(m.serviceTypes, id)
	return nil
}

// Mock service repo
type mockServiceRepo struct {
	services map[properties.UUID]*Service
}

func (m *mockServiceRepo) Get(ctx context.Context, id properties.UUID) (*Service, error) {
	if s, exists := m.services[id]; exists {
		return s, nil
	}
	return nil, NewNotFoundErrorf("Service with ID %s not found", id.String())
}

// Implement BaseEntityQuerier methods
func (m *mockServiceRepo) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	_, exists := m.services[id]
	return exists, nil
}

func (m *mockServiceRepo) List(ctx context.Context, scope *auth.IdentityScope, req *PageReq) (*PageRes[Service], error) {
	return &PageRes[Service]{}, nil
}

func (m *mockServiceRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(m.services)), nil
}

func (m *mockServiceRepo) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return &auth.DefaultObjectScope{}, nil
}

// Implement BaseEntityRepository methods
func (m *mockServiceRepo) Create(ctx context.Context, entity *Service) error {
	m.services[entity.ID] = entity
	return nil
}

func (m *mockServiceRepo) Save(ctx context.Context, entity *Service) error {
	m.services[entity.ID] = entity
	return nil
}

func (m *mockServiceRepo) Delete(ctx context.Context, id properties.UUID) error {
	delete(m.services, id)
	return nil
}

// Implement ServiceQuerier methods
func (m *mockServiceRepo) GetServicesByGroupID(ctx context.Context, groupID properties.UUID) ([]*Service, error) {
	return nil, nil
}

func (m *mockServiceRepo) ListByGroupID(ctx context.Context, groupID properties.UUID, scope *auth.IdentityScope, req *PageReq) (*PageRes[Service], error) {
	return &PageRes[Service]{}, nil
}

func (m *mockServiceRepo) GetActiveJobsForService(ctx context.Context, serviceID properties.UUID) ([]*Job, error) {
	return nil, nil
}

func (m *mockServiceRepo) CountByAgent(ctx context.Context, agentID properties.UUID) (int64, error) {
	return 0, nil
}

func (m *mockServiceRepo) CountByGroup(ctx context.Context, groupID properties.UUID) (int64, error) {
	return 0, nil
}

func (m *mockServiceRepo) FindByExternalID(ctx context.Context, agentID properties.UUID, externalID string) (*Service, error) {
	return nil, NewNotFoundErrorf("Service with external ID %s not found", externalID)
}

func (m *mockServiceRepo) CountByServiceType(ctx context.Context, serviceTypeID properties.UUID) (int64, error) {
	return 0, nil
}

// Mock service group repo
type mockServiceGroupRepo struct {
	groups map[properties.UUID]*ServiceGroup
}

func (m *mockServiceGroupRepo) Get(ctx context.Context, id properties.UUID) (*ServiceGroup, error) {
	if g, exists := m.groups[id]; exists {
		return g, nil
	}
	return nil, NewNotFoundErrorf("ServiceGroup with ID %s not found", id.String())
}

// Implement BaseEntityQuerier methods
func (m *mockServiceGroupRepo) Exists(ctx context.Context, id properties.UUID) (bool, error) {
	_, exists := m.groups[id]
	return exists, nil
}

func (m *mockServiceGroupRepo) List(ctx context.Context, scope *auth.IdentityScope, req *PageReq) (*PageRes[ServiceGroup], error) {
	return &PageRes[ServiceGroup]{}, nil
}

func (m *mockServiceGroupRepo) Count(ctx context.Context) (int64, error) {
	return int64(len(m.groups)), nil
}

func (m *mockServiceGroupRepo) AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error) {
	return &auth.DefaultObjectScope{}, nil
}

// Implement BaseEntityRepository methods
func (m *mockServiceGroupRepo) Create(ctx context.Context, entity *ServiceGroup) error {
	m.groups[entity.ID] = entity
	return nil
}

func (m *mockServiceGroupRepo) Save(ctx context.Context, entity *ServiceGroup) error {
	m.groups[entity.ID] = entity
	return nil
}

func (m *mockServiceGroupRepo) Delete(ctx context.Context, id properties.UUID) error {
	delete(m.groups, id)
	return nil
}

// Implement ServiceGroupQuerier methods
func (m *mockServiceGroupRepo) ListByConsumerID(ctx context.Context, consumerID properties.UUID, scope *auth.IdentityScope, req *PageReq) (*PageRes[ServiceGroup], error) {
	return &PageRes[ServiceGroup]{}, nil
}

// Test helper functions
func createTestContext() context.Context {
	return context.Background()
}

func createTestStore(serviceType *ServiceType) Store {
	uuid := uuid.New()
	serviceTypeID := properties.UUID(uuid)
	if serviceType != nil {
		serviceType.ID = serviceTypeID
	}

	return &mockStore{
		serviceTypeRepo: &mockServiceTypeRepo{
			serviceTypes: map[properties.UUID]*ServiceType{
				serviceTypeID: serviceType,
			},
		},
		serviceRepo:      &mockServiceRepo{services: make(map[properties.UUID]*Service)},
		serviceGroupRepo: &mockServiceGroupRepo{groups: make(map[properties.UUID]*ServiceGroup)},
	}
}

func createTestServiceType(schema ServiceSchema) *ServiceType {
	uuid := uuid.New()
	id := properties.UUID(uuid)
	return &ServiceType{
		BaseEntity: BaseEntity{
			ID:        id,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:           "Test Service Type",
		PropertySchema: &schema,
	}
}

func validateServicePropertiesHelper(data map[string]any, schema ServiceSchema) (map[string]any, []ValidationErrorDetail) {
	ctx := createTestContext()
	serviceType := createTestServiceType(schema)
	store := createTestStore(serviceType)
	uuid := uuid.New()
	groupID := properties.UUID(uuid)

	params := &ServicePropertyValidationParams{
		ServiceTypeID: serviceType.ID,
		GroupID:       groupID,
		Properties:    data,
	}

	result, err := ValidateServiceProperties(ctx, store, params)
	if err != nil {
		if validationError, ok := err.(ValidationError); ok {
			return nil, validationError.Errors
		}
		// For other errors, return them as a single validation error
		return nil, []ValidationErrorDetail{{Message: err.Error()}}
	}
	return result, nil
}

func TestValidate_RequiredFields(t *testing.T) {
	schema := ServiceSchema{
		"name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Required: true,
		},
		"age": ServicePropertyDefinition{
			Type:     SchemaTypeInteger,
			Required: false,
		},
	}

	// Test missing required field
	data := map[string]any{
		"age": 25,
	}

	_, errors := validateServicePropertiesHelper(data, schema)
	require.Len(t, errors, 1)
	assert.Equal(t, "name", errors[0].Path)
	assert.Equal(t, ErrSchemaRequiredFieldMissing, errors[0].Message)

	// Test with required field present
	data["name"] = "John"
	result, errors := validateServicePropertiesHelper(data, schema)
	require.Len(t, errors, 0)
	assert.Equal(t, "John", result["name"])
	assert.Equal(t, 25, result["age"])
}

func TestValidate_TypeValidation(t *testing.T) {
	schema := ServiceSchema{
		"name":   ServicePropertyDefinition{Type: SchemaTypeString},
		"age":    ServicePropertyDefinition{Type: SchemaTypeInteger},
		"score":  ServicePropertyDefinition{Type: SchemaTypeNumber},
		"active": ServicePropertyDefinition{Type: SchemaTypeBoolean},
		"config": ServicePropertyDefinition{Type: SchemaTypeObject},
		"tags":   ServicePropertyDefinition{Type: SchemaTypeArray},
	}

	// Test valid types
	validData := map[string]any{
		"name":   "John",
		"age":    25,
		"score":  95.5,
		"active": true,
		"config": map[string]any{"key": "value"},
		"tags":   []any{"tag1", "tag2"},
	}

	result, errors := validateServicePropertiesHelper(validData, schema)
	require.Len(t, errors, 0)
	assert.Equal(t, "John", result["name"])

	// Test invalid types
	invalidData := map[string]any{
		"name":   123,         // should be string
		"age":    "twenty",    // should be integer
		"score":  "high",      // should be number
		"active": "yes",       // should be boolean
		"config": "invalid",   // should be object
		"tags":   "tag1,tag2", // should be array
	}

	_, errors = validateServicePropertiesHelper(invalidData, schema)
	assert.Len(t, errors, 6)
}

func TestValidate_StringValidators(t *testing.T) {
	schema := ServiceSchema{
		"username": ServicePropertyDefinition{
			Type: SchemaTypeString,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMinLength, Value: 3},
				{Type: SchemaValidatorMaxLength, Value: 20},
				{Type: SchemaValidatorPattern, Value: "^[a-zA-Z0-9_]+$"},
			},
		},
		"status": ServicePropertyDefinition{
			Type: SchemaTypeString,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorEnum, Value: []any{"active", "inactive", "pending"}},
			},
		},
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectError bool
		errorCount  int
	}{
		{
			name: "valid string data",
			data: map[string]any{
				"username": "john_doe123",
				"status":   "active",
			},
			expectError: false,
		},
		{
			name: "username too short",
			data: map[string]any{
				"username": "jo",
				"status":   "active",
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "username too long",
			data: map[string]any{
				"username": "this_username_is_way_too_long_for_validation",
				"status":   "active",
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "username invalid pattern",
			data: map[string]any{
				"username": "john-doe!",
				"status":   "active",
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "invalid enum value",
			data: map[string]any{
				"username": "john_doe",
				"status":   "unknown",
			},
			expectError: true,
			errorCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errors := validateServicePropertiesHelper(tt.data, schema)
			if tt.expectError {
				assert.Len(t, errors, tt.errorCount)
			} else {
				assert.Len(t, errors, 0)
			}
		})
	}
}

func TestValidate_NumericValidators(t *testing.T) {
	schema := ServiceSchema{
		"cpu": ServicePropertyDefinition{
			Type: SchemaTypeInteger,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 1},
				{Type: SchemaValidatorMax, Value: 32},
			},
		},
		"memory": ServicePropertyDefinition{
			Type: SchemaTypeNumber,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 0.5},
				{Type: SchemaValidatorMax, Value: 64.0},
			},
		},
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectError bool
		errorCount  int
	}{
		{
			name: "valid numeric data",
			data: map[string]any{
				"cpu":    4,
				"memory": 8.5,
			},
			expectError: false,
		},
		{
			name: "cpu below minimum",
			data: map[string]any{
				"cpu":    0,
				"memory": 8.0,
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "cpu above maximum",
			data: map[string]any{
				"cpu":    64,
				"memory": 8.0,
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "memory below minimum",
			data: map[string]any{
				"cpu":    4,
				"memory": 0.1,
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "memory above maximum",
			data: map[string]any{
				"cpu":    4,
				"memory": 128.0,
			},
			expectError: true,
			errorCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errors := validateServicePropertiesHelper(tt.data, schema)
			if tt.expectError {
				assert.Len(t, errors, tt.errorCount)
			} else {
				assert.Len(t, errors, 0)
			}
		})
	}
}

func TestValidate_ArrayValidators(t *testing.T) {
	schema := ServiceSchema{
		"ports": ServicePropertyDefinition{
			Type: SchemaTypeArray,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMinItems, Value: 1},
				{Type: SchemaValidatorMaxItems, Value: 5},
				{Type: SchemaValidatorUniqueItems, Value: true},
			},
			Items: &ServicePropertyDefinition{
				Type: SchemaTypeInteger,
				Validators: []ServicePropertyValidatorDefinition{
					{Type: SchemaValidatorMin, Value: 1},
					{Type: SchemaValidatorMax, Value: 65535},
				},
			},
		},
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectError bool
		errorCount  int
	}{
		{
			name: "valid array",
			data: map[string]any{
				"ports": []any{80, 443, 8080},
			},
			expectError: false,
		},
		{
			name: "empty array (below minimum)",
			data: map[string]any{
				"ports": []any{},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "too many items",
			data: map[string]any{
				"ports": []any{80, 443, 8080, 9000, 3000, 5000},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "duplicate items",
			data: map[string]any{
				"ports": []any{80, 443, 80},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "invalid item values",
			data: map[string]any{
				"ports": []any{0, 80000},
			},
			expectError: true,
			errorCount:  2, // Two invalid port numbers
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errors := validateServicePropertiesHelper(tt.data, schema)
			if tt.expectError {
				assert.Len(t, errors, tt.errorCount)
			} else {
				assert.Len(t, errors, 0)
			}
		})
	}
}

func TestValidate_NestedObjects(t *testing.T) {
	schema := ServiceSchema{
		"metadata": ServicePropertyDefinition{
			Type: SchemaTypeObject,
			Properties: map[string]ServicePropertyDefinition{
				"owner": {
					Type:     SchemaTypeString,
					Required: true,
				},
				"version": {
					Type: SchemaTypeNumber,
					Validators: []ServicePropertyValidatorDefinition{
						{Type: SchemaValidatorMin, Value: 1.0},
					},
				},
				"tags": {
					Type: SchemaTypeArray,
					Items: &ServicePropertyDefinition{
						Type: SchemaTypeString,
						Validators: []ServicePropertyValidatorDefinition{
							{Type: SchemaValidatorMinLength, Value: 1},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		data        map[string]any
		expectError bool
		errorCount  int
	}{
		{
			name: "valid nested object",
			data: map[string]any{
				"metadata": map[string]any{
					"owner":   "john",
					"version": 2.1,
					"tags":    []any{"prod", "web"},
				},
			},
			expectError: false,
		},
		{
			name: "missing required nested field",
			data: map[string]any{
				"metadata": map[string]any{
					"version": 2.1,
				},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "invalid nested array item",
			data: map[string]any{
				"metadata": map[string]any{
					"owner": "john",
					"tags":  []any{"prod", ""},
				},
			},
			expectError: true,
			errorCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errors := validateServicePropertiesHelper(tt.data, schema)
			if tt.expectError {
				assert.Len(t, errors, tt.errorCount)
			} else {
				assert.Len(t, errors, 0)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	schema := ServiceSchema{
		"name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Required: true,
		},
		"port": ServicePropertyDefinition{
			Type:    SchemaTypeInteger,
			Default: 8080,
		},
		"enabled": ServicePropertyDefinition{
			Type:    SchemaTypeBoolean,
			Default: true,
		},
		"config": ServicePropertyDefinition{
			Type: SchemaTypeObject,
			Properties: map[string]ServicePropertyDefinition{
				"timeout": {
					Type:    SchemaTypeInteger,
					Default: 30,
				},
				"retries": {
					Type:    SchemaTypeInteger,
					Default: 3,
				},
			},
		},
	}

	tests := []struct {
		name     string
		input    map[string]any
		expected map[string]any
	}{
		{
			name: "apply missing defaults",
			input: map[string]any{
				"name": "test-service",
			},
			expected: map[string]any{
				"name":    "test-service",
				"port":    8080,
				"enabled": true,
			},
		},
		{
			name: "preserve existing values",
			input: map[string]any{
				"name":    "test-service",
				"port":    9000,
				"enabled": false,
			},
			expected: map[string]any{
				"name":    "test-service",
				"port":    9000,
				"enabled": false,
			},
		},
		{
			name: "apply nested defaults",
			input: map[string]any{
				"name": "test-service",
				"config": map[string]any{
					"timeout": 60,
				},
			},
			expected: map[string]any{
				"name":    "test-service",
				"port":    8080,
				"enabled": true,
				"config": map[string]any{
					"timeout": 60,
					"retries": 3,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyServicePropertiesDefaults(tt.input, schema)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestValidateWithDefaults(t *testing.T) {
	schema := ServiceSchema{
		"name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Required: true,
		},
		"port": ServicePropertyDefinition{
			Type:    SchemaTypeInteger,
			Default: 8080,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 1},
				{Type: SchemaValidatorMax, Value: 65535},
			},
		},
	}

	// Test with valid data and defaults
	input := map[string]any{
		"name": "test-service",
	}

	result, errors := validateServicePropertiesHelper(input, schema)
	require.Len(t, errors, 0)

	expected := map[string]any{
		"name": "test-service",
		"port": 8080,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}

	// Test with invalid default (this shouldn't happen in practice, but tests edge case)
	schemaWithInvalidDefault := ServiceSchema{
		"name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Required: true,
		},
		"port": ServicePropertyDefinition{
			Type:    SchemaTypeInteger,
			Default: 0, // Invalid according to validator
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 1},
			},
		},
	}

	_, errors = validateServicePropertiesHelper(input, schemaWithInvalidDefault)
	require.Greater(t, len(errors), 0, "Expected validation error for invalid default value")
}

func TestValidate_UnknownProperties(t *testing.T) {
	schema := ServiceSchema{
		"name": ServicePropertyDefinition{
			Type: SchemaTypeString,
		},
	}

	data := map[string]any{
		"name":         "test",
		"unknown_prop": "value",
	}

	_, errors := validateServicePropertiesHelper(data, schema)
	require.Len(t, errors, 1)
	assert.Equal(t, "unknown_prop", errors[0].Path)
	assert.Equal(t, ErrSchemaUnknownProperty, errors[0].Message)
}

func TestValidate_ComplexExample(t *testing.T) {
	// This test uses the example schema from the feature specification
	schema := ServiceSchema{
		"cpu": ServicePropertyDefinition{
			Type:     SchemaTypeInteger,
			Label:    "CPU Cores",
			Required: true,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMin, Value: 1},
			},
		},
		"image_name": ServicePropertyDefinition{
			Type:     SchemaTypeString,
			Label:    "Container Image",
			Required: true,
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMinLength, Value: 5},
				{Type: SchemaValidatorPattern, Value: "^[a-z0-9-]+$"},
			},
		},
		"environment": ServicePropertyDefinition{
			Type:  SchemaTypeString,
			Label: "Environment",
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorEnum, Value: []any{"development", "staging", "production"}},
			},
		},
		"enable_feature_x": ServicePropertyDefinition{
			Type:  SchemaTypeBoolean,
			Label: "Enable Feature X",
		},
		"metadata": ServicePropertyDefinition{
			Type:  SchemaTypeObject,
			Label: "Service Metadata",
			Properties: map[string]ServicePropertyDefinition{
				"owner": {
					Type:     SchemaTypeString,
					Label:    "Owner",
					Required: true,
				},
				"version": {
					Type:  SchemaTypeNumber,
					Label: "Version",
				},
			},
		},
		"ports": ServicePropertyDefinition{
			Type:  SchemaTypeArray,
			Label: "Port Configuration",
			Items: &ServicePropertyDefinition{
				Type: SchemaTypeInteger,
				Validators: []ServicePropertyValidatorDefinition{
					{Type: SchemaValidatorMin, Value: 1},
					{Type: SchemaValidatorMax, Value: 65535},
				},
			},
			Validators: []ServicePropertyValidatorDefinition{
				{Type: SchemaValidatorMinItems, Value: 1},
			},
		},
	}

	// Valid data
	validData := map[string]any{
		"cpu":              4,
		"image_name":       "nginx-alpine",
		"environment":      "production",
		"enable_feature_x": true,
		"metadata": map[string]any{
			"owner":   "devops-team",
			"version": 1.2,
		},
		"ports": []any{80, 443},
	}

	_, errors := validateServicePropertiesHelper(validData, schema)
	require.Len(t, errors, 0, "Expected no errors for valid complex data, got: %v", errors)

	// Invalid data with multiple errors
	invalidData := map[string]any{
		"cpu":              0,         // Below minimum
		"image_name":       "NGINX",   // Invalid pattern (uppercase)
		"environment":      "testing", // Not in enum
		"enable_feature_x": "yes",     // Wrong type
		"metadata": map[string]any{
			"version": 1.2,
			// Missing required "owner"
		},
		"ports": []any{}, // Below minimum items
	}

	_, errors = validateServicePropertiesHelper(invalidData, schema)
	require.GreaterOrEqual(t, len(errors), 5, "Expected at least 5 errors for invalid complex data, got: %v", errors)
}
