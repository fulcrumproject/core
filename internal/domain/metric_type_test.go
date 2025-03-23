package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMetricEntityType_Validate(t *testing.T) {
	tests := []struct {
		name       string
		entityType MetricEntityType
		wantErr    bool
		errMessage string
	}{
		{
			name:       "Valid MetricEntityTypeAgent",
			entityType: MetricEntityTypeAgent,
			wantErr:    false,
		},
		{
			name:       "Valid MetricEntityTypeService",
			entityType: MetricEntityTypeService,
			wantErr:    false,
		},
		{
			name:       "Valid MetricEntityTypeResource",
			entityType: MetricEntityTypeResource,
			wantErr:    false,
		},
		{
			name:       "Invalid entity type",
			entityType: "InvalidEntityType",
			wantErr:    true,
			errMessage: "invalid InvalidEntityType metric entity type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entityType.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricType_TableName(t *testing.T) {
	metricType := MetricType{}
	assert.Equal(t, "metric_types", metricType.TableName())
}

func TestMetricType_Validate(t *testing.T) {
	tests := []struct {
		name       string
		metricType *MetricType
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid metric type",
			metricType: &MetricType{
				Name:       "cpu-usage",
				EntityType: MetricEntityTypeResource,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			metricType: &MetricType{
				Name:       "",
				EntityType: MetricEntityTypeResource,
			},
			wantErr:    true,
			errMessage: "metric type name cannot be empty",
		},
		{
			name: "Invalid entity type",
			metricType: &MetricType{
				Name:       "cpu-usage",
				EntityType: "InvalidEntityType",
			},
			wantErr:    true,
			errMessage: "invalid entity type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metricType.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricTypeCommander_Create(t *testing.T) {
	ctx := context.Background()
	typeName := "memory-usage"
	entityType := MetricEntityTypeResource
	typeID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Create success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				// Mock Create method
				metricTypeRepo.createFunc = func(ctx context.Context, metricType *MetricType) error {
					// Set the ID to simulate DB create
					metricType.ID = typeID
					assert.Equal(t, typeName, metricType.Name)
					assert.Equal(t, entityType, metricType.EntityType)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeMetricTypeCreated, eventType)
					assert.NotNil(t, properties)
					assert.Equal(t, &typeID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name: "Validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "invalid entity type",
		},
		{
			name: "Create metric type error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				// Mock Create method to return error
				metricTypeRepo.createFunc = func(ctx context.Context, metricType *MetricType) error {
					return errors.New("database error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "database error",
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				// Mock Create method
				metricTypeRepo.createFunc = func(ctx context.Context, metricType *MetricType) error {
					metricType.ID = typeID
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					return nil, errors.New("audit entry error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "audit entry error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}

			// Special case for validation error which requires invalid inputs
			if tt.name == "Validation error" {
				tt.setupMocks(store, audit)
				commander := NewMetricTypeCommander(store, audit)
				metricType, err := commander.Create(ctx, typeName, "InvalidEntityType")

				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
				assert.Nil(t, metricType)
			} else {
				tt.setupMocks(store, audit)
				commander := NewMetricTypeCommander(store, audit)
				metricType, err := commander.Create(ctx, typeName, entityType)

				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMessage != "" {
						assert.Contains(t, err.Error(), tt.errMessage)
					}
					assert.Nil(t, metricType)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, metricType)
					assert.Equal(t, typeName, metricType.Name)
					assert.Equal(t, entityType, metricType.EntityType)
					assert.Equal(t, typeID, metricType.ID)
				}
			}
		})
	}
}

func TestMetricTypeCommander_Update(t *testing.T) {
	ctx := context.Background()
	typeID := uuid.New()
	existingName := "cpu-usage"
	newName := "cpu-utilization"

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Update success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				// Mock FindByID method
				existingType := &MetricType{
					BaseEntity: BaseEntity{
						ID: typeID,
					},
					Name:       existingName,
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					assert.Equal(t, typeID, id)
					return existingType, nil
				}

				// Mock Save method
				metricTypeRepo.updateFunc = func(ctx context.Context, metricType *MetricType) error {
					assert.Equal(t, typeID, metricType.ID)
					assert.Equal(t, newName, metricType.Name)
					assert.Equal(t, MetricEntityTypeResource, metricType.EntityType)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeMetricTypeUpdated, eventType)
					assert.Equal(t, &typeID, entityID)

					// Verify before and after objects
					beforeType, ok := before.(*MetricType)
					assert.True(t, ok)
					assert.Equal(t, existingName, beforeType.Name)

					afterType, ok := after.(*MetricType)
					assert.True(t, ok)
					assert.Equal(t, newName, afterType.Name)

					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name: "Metric type not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				// Mock FindByID method to return not found error
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					return nil, NewNotFoundErrorf("metric type not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				// Mock FindByID method
				existingType := &MetricType{
					BaseEntity: BaseEntity{
						ID: typeID,
					},
					Name:       existingName,
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					// Change entity type to invalid for validation error
					existingType.EntityType = "InvalidEntityType"
					return existingType, nil
				}
			},
			wantErr:    true,
			errMessage: "invalid entity type",
		},
		{
			name: "Save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				// Mock FindByID method
				existingType := &MetricType{
					BaseEntity: BaseEntity{
						ID: typeID,
					},
					Name:       existingName,
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					return existingType, nil
				}

				// Mock Save method with error
				metricTypeRepo.updateFunc = func(ctx context.Context, metricType *MetricType) error {
					return errors.New("database error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "database error",
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				// Mock FindByID method
				existingType := &MetricType{
					BaseEntity: BaseEntity{
						ID: typeID,
					},
					Name:       existingName,
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					return existingType, nil
				}

				// Mock Save method
				metricTypeRepo.updateFunc = func(ctx context.Context, metricType *MetricType) error {
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					return nil, errors.New("audit entry error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "audit entry error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewMetricTypeCommander(store, audit)
			updatedType, err := commander.Update(ctx, typeID, &newName)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, updatedType)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updatedType)
				assert.Equal(t, typeID, updatedType.ID)
				assert.Equal(t, newName, updatedType.Name)
			}
		})
	}
}

func TestMetricTypeCommander_Delete(t *testing.T) {
	ctx := context.Background()
	typeID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Delete success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				metricEntryRepo := &MockMetricEntryRepository{}
				store.WithMetricEntryRepo(metricEntryRepo)

				// Mock FindByID method
				existingType := &MetricType{
					BaseEntity: BaseEntity{
						ID: typeID,
					},
					Name:       "cpu-usage",
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					assert.Equal(t, typeID, id)
					return existingType, nil
				}

				// Mock CountByMetricType to return 0 (no associated entries)
				metricEntryRepo.countByMetricTypeFunc = func(ctx context.Context, id UUID) (int64, error) {
					assert.Equal(t, typeID, id)
					return 0, nil
				}

				// Mock Delete method
				metricTypeRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					assert.Equal(t, typeID, id)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeMetricTypeDeleted, eventType)
					assert.NotNil(t, properties)
					assert.Equal(t, &typeID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
		},
		{
			name: "Metric type not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				// Mock FindByID method to return not found error
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					return nil, NewNotFoundErrorf("metric type not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Has associated entries",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				metricEntryRepo := &MockMetricEntryRepository{}
				store.WithMetricEntryRepo(metricEntryRepo)

				// Mock FindByID method
				existingType := &MetricType{
					BaseEntity: BaseEntity{
						ID: typeID,
					},
					Name:       "cpu-usage",
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					return existingType, nil
				}

				// Mock CountByMetricType to return non-zero (has associated entries)
				metricEntryRepo.countByMetricTypeFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 5, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "cannot delete metric-type with associated entries",
		},
		{
			name: "Count error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				metricEntryRepo := &MockMetricEntryRepository{}
				store.WithMetricEntryRepo(metricEntryRepo)

				// Mock FindByID method
				existingType := &MetricType{
					BaseEntity: BaseEntity{
						ID: typeID,
					},
					Name:       "cpu-usage",
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					return existingType, nil
				}

				// Mock CountByMetricType to return error
				metricEntryRepo.countByMetricTypeFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 0, errors.New("database error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "database error",
		},
		{
			name: "Delete error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				metricEntryRepo := &MockMetricEntryRepository{}
				store.WithMetricEntryRepo(metricEntryRepo)

				// Mock FindByID method
				existingType := &MetricType{
					BaseEntity: BaseEntity{
						ID: typeID,
					},
					Name:       "cpu-usage",
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					return existingType, nil
				}

				// Mock CountByMetricType to return 0 (no associated entries)
				metricEntryRepo.countByMetricTypeFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 0, nil
				}

				// Mock Delete method with error
				metricTypeRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					return errors.New("delete error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "delete error",
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				metricEntryRepo := &MockMetricEntryRepository{}
				store.WithMetricEntryRepo(metricEntryRepo)

				// Mock FindByID method
				existingType := &MetricType{
					BaseEntity: BaseEntity{
						ID: typeID,
					},
					Name:       "cpu-usage",
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByIDFunc = func(ctx context.Context, id UUID) (*MetricType, error) {
					return existingType, nil
				}

				// Mock CountByMetricType to return 0 (no associated entries)
				metricEntryRepo.countByMetricTypeFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 0, nil
				}

				// Mock Delete method
				metricTypeRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					return nil, errors.New("audit entry error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "audit entry error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewMetricTypeCommander(store, audit)
			err := commander.Delete(ctx, typeID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
