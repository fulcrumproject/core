package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMetricEntry_TableName(t *testing.T) {
	entry := MetricEntry{}
	assert.Equal(t, "metric_entries", entry.TableName())
}

func TestMetricEntry_Validate(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name       string
		entry      *MetricEntry
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid entry",
			entry: &MetricEntry{
				ResourceID: "cpu",
				Value:      42.5,
				TypeID:     validID,
				AgentID:    validID,
				ServiceID:  validID,
				ProviderID: validID,
				BrokerID:   validID,
			},
			wantErr: false,
		},
		{
			name: "Empty ResourceID",
			entry: &MetricEntry{
				ResourceID: "",
				Value:      42.5,
				TypeID:     validID,
				AgentID:    validID,
				ServiceID:  validID,
				ProviderID: validID,
				BrokerID:   validID,
			},
			wantErr:    true,
			errMessage: "resource ID cannot be empty",
		},
		{
			name: "Empty TypeID",
			entry: &MetricEntry{
				ResourceID: "cpu",
				Value:      42.5,
				TypeID:     uuid.Nil,
				AgentID:    validID,
				ServiceID:  validID,
				ProviderID: validID,
				BrokerID:   validID,
			},
			wantErr:    true,
			errMessage: "metric type ID cannot be empty",
		},
		{
			name: "Empty AgentID",
			entry: &MetricEntry{
				ResourceID: "cpu",
				Value:      42.5,
				TypeID:     validID,
				AgentID:    uuid.Nil,
				ServiceID:  validID,
				ProviderID: validID,
				BrokerID:   validID,
			},
			wantErr:    true,
			errMessage: "agent ID cannot be empty",
		},
		{
			name: "Empty ServiceID",
			entry: &MetricEntry{
				ResourceID: "cpu",
				Value:      42.5,
				TypeID:     validID,
				AgentID:    validID,
				ServiceID:  uuid.Nil,
				ProviderID: validID,
				BrokerID:   validID,
			},
			wantErr:    true,
			errMessage: "service ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
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

func TestMetricEntryCommander_Create(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	serviceID := uuid.New()
	providerID := uuid.New()
	brokerID := uuid.New()
	metricTypeID := uuid.New()
	resourceID := "cpu-usage"
	typeName := "cpu"
	value := 75.5

	tests := []struct {
		name       string
		setupMocks func(store *MockStore)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Create success",
			setupMocks: func(store *MockStore) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				metricEntryRepo := &MockMetricEntryRepository{}
				store.WithMetricEntryRepo(metricEntryRepo)

				// Agent exists check
				agentRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					assert.Equal(t, agentID, id)
					return true, nil
				}

				// Service findByID
				service := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					ProviderID: providerID,
					BrokerID:   brokerID,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return service, nil
				}

				// Metric type findByName
				metricType := &MetricType{
					BaseEntity: BaseEntity{
						ID: metricTypeID,
					},
					Name:       typeName,
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByNameFunc = func(ctx context.Context, name string) (*MetricType, error) {
					assert.Equal(t, typeName, name)
					return metricType, nil
				}

				// Metric type exists check
				metricTypeRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					assert.Equal(t, metricTypeID, id)
					return true, nil
				}

				// Create metric entry
				metricEntryRepo.createFunc = func(ctx context.Context, entry *MetricEntry) error {
					assert.Equal(t, resourceID, entry.ResourceID)
					assert.Equal(t, value, entry.Value)
					assert.Equal(t, metricTypeID, entry.TypeID)
					assert.Equal(t, agentID, entry.AgentID)
					assert.Equal(t, serviceID, entry.ServiceID)
					assert.Equal(t, providerID, entry.ProviderID)
					assert.Equal(t, brokerID, entry.BrokerID)
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "Agent does not exist",
			setupMocks: func(store *MockStore) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				agentRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return false, nil
				}
			},
			wantErr:    true,
			errMessage: "invalid agent ID",
		},
		{
			name: "Agent exists check error",
			setupMocks: func(store *MockStore) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				agentRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return false, errors.New("database error")
				}
			},
			wantErr:    true,
			errMessage: "database error",
		},
		{
			name: "Service not found",
			setupMocks: func(store *MockStore) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				agentRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return nil, NewNotFoundErrorf("service not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Metric type not found",
			setupMocks: func(store *MockStore) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				agentRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				service := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					ProviderID: providerID,
					BrokerID:   brokerID,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				metricTypeRepo.findByNameFunc = func(ctx context.Context, name string) (*MetricType, error) {
					return nil, NewNotFoundErrorf("metric type not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Metric type does not exist",
			setupMocks: func(store *MockStore) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				agentRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				service := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					ProviderID: providerID,
					BrokerID:   brokerID,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				metricType := &MetricType{
					BaseEntity: BaseEntity{
						ID: metricTypeID,
					},
					Name:       typeName,
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByNameFunc = func(ctx context.Context, name string) (*MetricType, error) {
					return metricType, nil
				}

				metricTypeRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return false, nil
				}
			},
			wantErr:    true,
			errMessage: "metric type with ID",
		},
		{
			name: "Failed to create metric entry",
			setupMocks: func(store *MockStore) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)

				metricEntryRepo := &MockMetricEntryRepository{}
				store.WithMetricEntryRepo(metricEntryRepo)

				agentRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				service := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					ProviderID: providerID,
					BrokerID:   brokerID,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				metricType := &MetricType{
					BaseEntity: BaseEntity{
						ID: metricTypeID,
					},
					Name:       typeName,
					EntityType: MetricEntityTypeResource,
				}
				metricTypeRepo.findByNameFunc = func(ctx context.Context, name string) (*MetricType, error) {
					return metricType, nil
				}

				metricTypeRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				metricEntryRepo.createFunc = func(ctx context.Context, entry *MetricEntry) error {
					return errors.New("database error")
				}
			},
			wantErr:    true,
			errMessage: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			tt.setupMocks(store)

			commander := NewMetricEntryCommander(store)
			entry, err := commander.Create(ctx, typeName, agentID, serviceID, resourceID, value)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, entry)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, entry)
				assert.Equal(t, resourceID, entry.ResourceID)
				assert.Equal(t, value, entry.Value)
				assert.Equal(t, metricTypeID, entry.TypeID)
				assert.Equal(t, agentID, entry.AgentID)
				assert.Equal(t, serviceID, entry.ServiceID)
				assert.Equal(t, providerID, entry.ProviderID)
				assert.Equal(t, brokerID, entry.BrokerID)
			}
		})
	}
}

func TestMetricEntryCommander_CreateWithExternalID(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.New()
	serviceID := uuid.New()
	externalID := "external-svc-123"
	resourceID := "memory-usage"
	typeName := "memory"
	value := 85.2

	tests := []struct {
		name       string
		setupMocks func(store *MockStore)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Create with external ID success",
			setupMocks: func(store *MockStore) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock finding service by external ID
				service := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
				}
				serviceRepo.findByExternalIDFunc = func(ctx context.Context, agentId UUID, extID string) (*Service, error) {
					assert.Equal(t, agentID, agentId)
					assert.Equal(t, externalID, extID)
					return service, nil
				}

				// We'll set up a simple mock for the regular create method
				// The full testing of the Create method is done in TestMetricEntryCommander_Create
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)
				metricTypeRepo := &MockMetricTypeRepository{}
				store.WithMetricTypeRepo(metricTypeRepo)
				metricEntryRepo := &MockMetricEntryRepository{}
				store.WithMetricEntryRepo(metricEntryRepo)

				// Set up enough mocks for the Create method to succeed
				agentRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return service, nil
				}

				metricTypeRepo.findByNameFunc = func(ctx context.Context, name string) (*MetricType, error) {
					return &MetricType{BaseEntity: BaseEntity{ID: uuid.New()}}, nil
				}

				metricTypeRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return true, nil
				}

				metricEntryRepo.createFunc = func(ctx context.Context, entry *MetricEntry) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "Service not found by external ID",
			setupMocks: func(store *MockStore) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				serviceRepo.findByExternalIDFunc = func(ctx context.Context, agentId UUID, extID string) (*Service, error) {
					return nil, NewNotFoundErrorf("service not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			tt.setupMocks(store)

			commander := NewMetricEntryCommander(store)
			entry, err := commander.CreateWithExternalID(ctx, typeName, agentID, externalID, resourceID, value)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, entry)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, entry)
			}
		})
	}
}
