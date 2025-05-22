package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceCommander_Create(t *testing.T) {
	ctx := context.Background()
	serviceID := uuid.New()
	agentID := uuid.New()
	serviceTypeID := uuid.New()
	groupID := uuid.New()
	providerID := uuid.New()
	consumerID := uuid.New()
	validName := "Web Server"
	validAttributes := Attributes{"tier": {"premium"}}
	validProperties := JSON{"port": 8080}

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Create success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				// Set up repositories
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceGroupRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(serviceGroupRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock agent FindByID
				agent := &Agent{
					BaseEntity: BaseEntity{
						ID: agentID,
					},
					ProviderID: providerID,
				}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					assert.Equal(t, agentID, id)
					return agent, nil
				}

				// Mock service group FindByID
				group := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					ConsumerID: consumerID,
				}
				serviceGroupRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					assert.Equal(t, groupID, id)
					return group, nil
				}

				// Mock service Create
				serviceRepo.createFunc = func(ctx context.Context, service *Service) error {
					service.ID = serviceID
					assert.Equal(t, validName, service.Name)
					assert.Equal(t, ServiceCreating, service.CurrentState)
					assert.Equal(t, validAttributes, service.Attributes)
					assert.Equal(t, &validProperties, service.TargetProperties)
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					assert.Equal(t, serviceID, job.ServiceID)
					assert.Equal(t, ServiceActionCreate, job.Action)
					assert.Equal(t, JobPending, job.State)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, consumerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceCreated, eventType)
					assert.NotNil(t, properties)
					assert.NotNil(t, entityID)
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
			name: "Agent not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				// Mock agent FindByID with error
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return nil, NewNotFoundErrorf("agent not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Service group not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceGroupRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(serviceGroupRepo)

				// Mock agent FindByID
				agent := &Agent{
					BaseEntity: BaseEntity{
						ID: agentID,
					},
					ProviderID: providerID,
				}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return agent, nil
				}

				// Mock service group FindByID with error
				serviceGroupRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return nil, NewNotFoundErrorf("service group not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceGroupRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(serviceGroupRepo)

				// Mock agent FindByID
				agent := &Agent{
					BaseEntity: BaseEntity{
						ID: agentID,
					},
					ProviderID: providerID,
				}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return agent, nil
				}

				// Mock service group FindByID
				group := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					ConsumerID: consumerID,
				}
				serviceGroupRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return group, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "service name cannot be empty",
		},
		{
			name: "Service create error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceGroupRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(serviceGroupRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock agent FindByID
				agent := &Agent{
					BaseEntity: BaseEntity{
						ID: agentID,
					},
					ProviderID: providerID,
				}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return agent, nil
				}

				// Mock service group FindByID
				group := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					ConsumerID: consumerID,
				}
				serviceGroupRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return group, nil
				}

				// Mock service Create with error
				serviceRepo.createFunc = func(ctx context.Context, service *Service) error {
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
			name: "Job create error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceGroupRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(serviceGroupRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock agent FindByID
				agent := &Agent{
					BaseEntity: BaseEntity{
						ID: agentID,
					},
					ProviderID: providerID,
				}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return agent, nil
				}

				// Mock service group FindByID
				group := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					ConsumerID: consumerID,
				}
				serviceGroupRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return group, nil
				}

				// Mock service Create
				serviceRepo.createFunc = func(ctx context.Context, service *Service) error {
					service.ID = serviceID
					return nil
				}

				// Mock job Create with error
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					return errors.New("job create error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "job create error",
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				serviceGroupRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(serviceGroupRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock agent FindByID
				agent := &Agent{
					BaseEntity: BaseEntity{
						ID: agentID,
					},
					ProviderID: providerID,
				}
				agentRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Agent, error) {
					return agent, nil
				}

				// Mock service group FindByID
				group := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					ConsumerID: consumerID,
				}
				serviceGroupRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return group, nil
				}

				// Mock service Create
				serviceRepo.createFunc = func(ctx context.Context, service *Service) error {
					service.ID = serviceID
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, consumerID *UUID) (*AuditEntry, error) {
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

			// Special case for validation error
			if tt.name == "Validation error" {
				tt.setupMocks(store, audit)
				commander := NewServiceCommander(store, audit)
				svc, err := commander.Create(ctx, agentID, serviceTypeID, groupID, "", validAttributes, validProperties)

				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
				assert.Nil(t, svc)
			} else {
				tt.setupMocks(store, audit)
				commander := NewServiceCommander(store, audit)
				svc, err := commander.Create(ctx, agentID, serviceTypeID, groupID, validName, validAttributes, validProperties)

				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMessage != "" {
						assert.Contains(t, err.Error(), tt.errMessage)
					}
					assert.Nil(t, svc)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, svc)
					assert.Equal(t, serviceID, svc.ID)
					assert.Equal(t, validName, svc.Name)
					assert.Equal(t, validAttributes, svc.Attributes)
					assert.Equal(t, ServiceCreating, svc.CurrentState)
					var targetCreated ServiceState = ServiceCreated
					assert.Equal(t, &targetCreated, svc.TargetState)
				}
			}
		})
	}
}

func svcActionPtr(a ServiceAction) *ServiceAction {
	return &a
}

func TestServiceCommander_Update(t *testing.T) {
	ctx := context.Background()
	serviceID := uuid.New()
	agentID := uuid.New()
	typeID := uuid.New()
	groupID := uuid.New()
	providerID := uuid.New()
	consumerID := uuid.New()
	validName := "Web Server"
	newName := "API Server"
	validProperties := JSON{"port": 8080}
	newProperties := JSON{"port": 9000}
	newPropertiesPtr := &newProperties

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		inputName  *string
		inputProps *JSON
		wantErr    bool
		errMessage string
		checkState func(t *testing.T, svc *Service)
	}{
		{
			name: "Service not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return nil, NewNotFoundErrorf("service not found")
				}
			},
			inputName:  &newName,
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Update name only",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:           agentID,
					ProviderID:        providerID,
					ServiceTypeID:     typeID,
					GroupID:           groupID,
					ConsumerID:        consumerID,
					Name:              validName,
					CurrentState:      ServiceStopped,
					CurrentProperties: &validProperties,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					assert.Equal(t, newName, service.Name)
					assert.Equal(t, ServiceStopped, service.CurrentState)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, original, current interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceUpdated, eventType)
					assert.Equal(t, &serviceID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			inputName:  &newName,
			inputProps: nil,
			wantErr:    false,
			checkState: func(t *testing.T, svc *Service) {
				assert.Equal(t, newName, svc.Name)
				assert.Equal(t, ServiceStopped, svc.CurrentState)
				assert.Equal(t, validProperties, *svc.CurrentProperties)
			},
		},
		{
			name: "Update properties - cold update",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:           agentID,
					ProviderID:        providerID,
					ServiceTypeID:     typeID,
					GroupID:           groupID,
					ConsumerID:        consumerID,
					Name:              validName,
					CurrentState:      ServiceStopped,
					CurrentProperties: &validProperties,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					assert.Equal(t, validName, service.Name)
					assert.Equal(t, ServiceColdUpdating, service.CurrentState)
					assert.Equal(t, newPropertiesPtr, service.TargetProperties)
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					assert.Equal(t, serviceID, job.ServiceID)
					assert.Equal(t, ServiceActionColdUpdate, job.Action)
					assert.Equal(t, JobPending, job.State)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, original, current interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceUpdated, eventType)
					assert.Equal(t, &serviceID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			inputName:  nil,
			inputProps: newPropertiesPtr,
			wantErr:    false,
			checkState: func(t *testing.T, svc *Service) {
				assert.Equal(t, validName, svc.Name)
				assert.Equal(t, ServiceColdUpdating, svc.CurrentState)
				var expectedState ServiceState = ServiceStopped
				assert.Equal(t, &expectedState, svc.TargetState)
				assert.Equal(t, newPropertiesPtr, svc.TargetProperties)
			},
		},
		{
			name: "Update properties - hot update",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:           agentID,
					ProviderID:        providerID,
					ServiceTypeID:     typeID,
					GroupID:           groupID,
					ConsumerID:        consumerID,
					Name:              validName,
					CurrentState:      ServiceStarted,
					CurrentProperties: &validProperties,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					assert.Equal(t, validName, service.Name)
					assert.Equal(t, ServiceHotUpdating, service.CurrentState)
					assert.Equal(t, newPropertiesPtr, service.TargetProperties)
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					assert.Equal(t, serviceID, job.ServiceID)
					assert.Equal(t, ServiceActionHotUpdate, job.Action)
					assert.Equal(t, JobPending, job.State)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, original, current interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceUpdated, eventType)
					assert.Equal(t, &serviceID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			inputName:  nil,
			inputProps: newPropertiesPtr,
			wantErr:    false,
			checkState: func(t *testing.T, svc *Service) {
				assert.Equal(t, validName, svc.Name)
				assert.Equal(t, ServiceHotUpdating, svc.CurrentState)
				var expectedState ServiceState = ServiceStarted
				assert.Equal(t, &expectedState, svc.TargetState)
				assert.Equal(t, newPropertiesPtr, svc.TargetProperties)
			},
		},
		{
			name: "Invalid state for update",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:           agentID,
					ProviderID:        providerID,
					ServiceTypeID:     typeID,
					GroupID:           groupID,
					ConsumerID:        consumerID,
					Name:              validName,
					CurrentState:      ServiceCreating,
					CurrentProperties: &validProperties,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}
			},
			inputName:  nil,
			inputProps: newPropertiesPtr,
			wantErr:    true,
			errMessage: "cannot update attributes on a service with state",
		},
		{
			name: "Service save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:           agentID,
					ProviderID:        providerID,
					ServiceTypeID:     typeID,
					GroupID:           groupID,
					ConsumerID:        consumerID,
					Name:              validName,
					CurrentState:      ServiceStopped,
					CurrentProperties: &validProperties,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save with error
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					return errors.New("database error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			inputName:  &newName,
			wantErr:    true,
			errMessage: "database error",
		},
		{
			name: "Job create error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:           agentID,
					ServiceTypeID:     typeID,
					GroupID:           groupID,
					ProviderID:        providerID,
					ConsumerID:        consumerID,
					Name:              validName,
					CurrentState:      ServiceStopped,
					CurrentProperties: &validProperties,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					return nil
				}

				// Mock job Create with error
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					return errors.New("job create error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			inputName:  nil,
			inputProps: newPropertiesPtr,
			wantErr:    true,
			errMessage: "job create error",
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:           agentID,
					ProviderID:        providerID,
					ServiceTypeID:     typeID,
					GroupID:           groupID,
					ConsumerID:        consumerID,
					Name:              validName,
					CurrentState:      ServiceStopped,
					CurrentProperties: &validProperties,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, original, current interface{}) (*AuditEntry, error) {
					return nil, errors.New("audit entry error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			inputName:  nil,
			inputProps: newPropertiesPtr,
			wantErr:    true,
			errMessage: "audit entry error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}

			tt.setupMocks(store, audit)
			commander := NewServiceCommander(store, audit)

			svc, err := commander.Update(ctx, serviceID, tt.inputName, tt.inputProps)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, svc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)
				tt.checkState(t, svc)
			}
		})
	}
}

func TestServiceCommander_Transition(t *testing.T) {
	ctx := context.Background()
	serviceID := uuid.New()
	agentID := uuid.New()
	providerID := uuid.New()
	consumerID := uuid.New()
	typeID := uuid.New()
	groupID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		current    ServiceState
		target     ServiceState
		wantErr    bool
		errMessage string
		checkState func(t *testing.T, svc *Service)
	}{
		{
			name: "Service not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return nil, NewNotFoundErrorf("service not found")
				}
			},
			current:    ServiceCreated,
			target:     ServiceStarted,
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Transition from Created to Started",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceCreated,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					assert.Equal(t, ServiceStarting, service.CurrentState)
					var target ServiceState = ServiceStarted
					assert.Equal(t, &target, service.TargetState)
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					assert.Equal(t, serviceID, job.ServiceID)
					assert.Equal(t, ServiceActionStart, job.Action)
					assert.Equal(t, JobPending, job.State)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, original, current interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceTransitioned, eventType)
					assert.Equal(t, &serviceID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			current: ServiceCreated,
			target:  ServiceStarted,
			wantErr: false,
			checkState: func(t *testing.T, svc *Service) {
				assert.Equal(t, ServiceStarting, svc.CurrentState)
				var target ServiceState = ServiceStarted
				assert.Equal(t, &target, svc.TargetState)
			},
		},
		{
			name: "Transition from Started to Stopped",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceStarted,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					assert.Equal(t, ServiceStopping, service.CurrentState)
					var target ServiceState = ServiceStopped
					assert.Equal(t, &target, service.TargetState)
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					assert.Equal(t, serviceID, job.ServiceID)
					assert.Equal(t, ServiceActionStop, job.Action)
					assert.Equal(t, JobPending, job.State)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, original, current interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceTransitioned, eventType)
					assert.Equal(t, &serviceID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			current: ServiceStarted,
			target:  ServiceStopped,
			wantErr: false,
			checkState: func(t *testing.T, svc *Service) {
				assert.Equal(t, ServiceStopping, svc.CurrentState)
				var target ServiceState = ServiceStopped
				assert.Equal(t, &target, svc.TargetState)
			},
		},
		{
			name: "Transition from Stopped to Deleted",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceStopped,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					assert.Equal(t, ServiceDeleting, service.CurrentState)
					var target ServiceState = ServiceDeleted
					assert.Equal(t, &target, service.TargetState)
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					assert.Equal(t, serviceID, job.ServiceID)
					assert.Equal(t, ServiceActionDelete, job.Action)
					assert.Equal(t, JobPending, job.State)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, original, current interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceTransitioned, eventType)
					assert.Equal(t, &serviceID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			current: ServiceStopped,
			target:  ServiceDeleted,
			wantErr: false,
			checkState: func(t *testing.T, svc *Service) {
				assert.Equal(t, ServiceDeleting, svc.CurrentState)
				var target ServiceState = ServiceDeleted
				assert.Equal(t, &target, svc.TargetState)
			},
		},
		{
			name: "Invalid transition",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceStarted,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}
			},
			current:    ServiceStarted,
			target:     ServiceDeleted,
			wantErr:    true,
			errMessage: "invalid transition from",
		},
		{
			name: "Service save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceCreated,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save with error
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					return errors.New("database error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			current:    ServiceCreated,
			target:     ServiceStarted,
			wantErr:    true,
			errMessage: "database error",
		},
		{
			name: "Job create error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceCreated,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					return nil
				}

				// Mock job Create with error
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					return errors.New("job create error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			current:    ServiceCreated,
			target:     ServiceStarted,
			wantErr:    true,
			errMessage: "job create error",
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceCreated,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, original, current interface{}) (*AuditEntry, error) {
					return nil, errors.New("audit entry error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			current:    ServiceCreated,
			target:     ServiceStarted,
			wantErr:    true,
			errMessage: "audit entry error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}

			tt.setupMocks(store, audit)
			commander := NewServiceCommander(store, audit)

			svc, err := commander.Transition(ctx, serviceID, tt.target)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, svc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)
				tt.checkState(t, svc)
			}
		})
	}
}

func TestServiceCommander_Retry(t *testing.T) {
	ctx := context.Background()
	serviceID := uuid.New()
	agentID := uuid.New()
	providerID := uuid.New()
	consumerID := uuid.New()
	typeID := uuid.New()
	groupID := uuid.New()
	failedAction := ServiceActionStart

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
		checkState func(t *testing.T, svc *Service)
	}{
		{
			name: "Service not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return nil, NewNotFoundErrorf("service not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "No failed action to retry",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock service FindByID with no failed action
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceStarted,
					FailedAction:  nil, // No failed action
					RetryCount:    0,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}
			},
			wantErr: false,
			checkState: func(t *testing.T, svc *Service) {
				assert.Equal(t, ServiceStarted, svc.CurrentState)
				assert.Nil(t, svc.FailedAction)
				assert.Equal(t, 0, svc.RetryCount)
			},
		},
		{
			name: "Successful retry",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID with failed action
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceStarted,
					FailedAction:  &failedAction,
					RetryCount:    1,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					assert.Equal(t, 2, service.RetryCount) // Should be incremented
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					assert.Equal(t, serviceID, job.ServiceID)
					assert.Equal(t, failedAction, job.Action)
					assert.Equal(t, JobPending, job.State)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, original, current interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceRetried, eventType)
					assert.Equal(t, &serviceID, entityID)
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: false,
			checkState: func(t *testing.T, svc *Service) {
				assert.Equal(t, ServiceStarted, svc.CurrentState)
				assert.NotNil(t, svc.FailedAction)
				assert.Equal(t, failedAction, *svc.FailedAction)
				assert.Equal(t, 2, svc.RetryCount)
			},
		},
		{
			name: "Service save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock service FindByID with failed action
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceStarted,
					FailedAction:  &failedAction,
					RetryCount:    1,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save with error
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
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
			name: "Job create error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID with failed action
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceStarted,
					FailedAction:  &failedAction,
					RetryCount:    1,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					return nil
				}

				// Mock job Create with error
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					return errors.New("job create error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "job create error",
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock service FindByID with failed action
				svc := &Service{
					BaseEntity: BaseEntity{
						ID: serviceID,
					},
					AgentID:       agentID,
					ProviderID:    providerID,
					ConsumerID:    consumerID,
					ServiceTypeID: typeID,
					GroupID:       groupID,
					Name:          "Test Service",
					CurrentState:  ServiceStarted,
					FailedAction:  &failedAction,
					RetryCount:    1,
				}
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					assert.Equal(t, serviceID, id)
					return svc, nil
				}

				// Mock service Save
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					return nil
				}

				// Mock job Create
				jobRepo.createFunc = func(ctx context.Context, job *Job) error {
					return nil
				}

				// Mock audit entry creation with error
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, original, current interface{}) (*AuditEntry, error) {
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
			commander := NewServiceCommander(store, audit)

			svc, err := commander.Retry(ctx, serviceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, svc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)
				tt.checkState(t, svc)
			}
		})
	}
}

func TestServiceCommander_FailTimeoutServicesAndJobs(t *testing.T) {
	ctx := context.Background()
	serviceID1 := uuid.New()
	serviceID2 := uuid.New()
	timeout := 5 * time.Minute

	tests := []struct {
		name       string
		setupMocks func(store *MockStore)
		wantErr    bool
		errMessage string
		wantCount  int
	}{
		{
			name: "Error getting timed out jobs",
			setupMocks: func(store *MockStore) {
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock getting timed out jobs with error
				jobRepo.getTimeOutJobsFunc = func(ctx context.Context, timeout time.Duration) ([]*Job, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr:    true,
			errMessage: "failed to retrive timeout jobs",
			wantCount:  0,
		},
		{
			name: "No timed out jobs",
			setupMocks: func(store *MockStore) {
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)

				// Mock getting timed out jobs - empty result
				jobRepo.getTimeOutJobsFunc = func(ctx context.Context, timeout time.Duration) ([]*Job, error) {
					return []*Job{}, nil
				}
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "Error finding service",
			setupMocks: func(store *MockStore) {
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock getting timed out jobs
				jobRepo.getTimeOutJobsFunc = func(ctx context.Context, timeout time.Duration) ([]*Job, error) {
					return []*Job{
						{
							BaseEntity: BaseEntity{
								ID: uuid.New(),
							},
							ServiceID: serviceID1,
							Action:    ServiceActionStart,
							State:     JobProcessing,
						},
					}, nil
				}

				// Mock finding service with error
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return nil, errors.New("service not found")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "failed to find service",
			wantCount:  0,
		},
		{
			name: "Error saving job",
			setupMocks: func(store *MockStore) {
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock getting timed out jobs
				jobRepo.getTimeOutJobsFunc = func(ctx context.Context, timeout time.Duration) ([]*Job, error) {
					return []*Job{
						{
							BaseEntity: BaseEntity{
								ID: uuid.New(),
							},
							ServiceID: serviceID1,
							Action:    ServiceActionStart,
							State:     JobProcessing,
						},
					}, nil
				}

				// Mock finding service
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return &Service{
						BaseEntity: BaseEntity{
							ID: id,
						},
						CurrentState: ServiceStarting,
					}, nil
				}

				// Mock saving job with error
				jobRepo.saveFunc = func(ctx context.Context, job *Job) error {
					return errors.New("error saving job")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "Error marking timed out job",
			wantCount:  0,
		},
		{
			name: "Error saving service",
			setupMocks: func(store *MockStore) {
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock getting timed out jobs
				jobRepo.getTimeOutJobsFunc = func(ctx context.Context, timeout time.Duration) ([]*Job, error) {
					return []*Job{
						{
							BaseEntity: BaseEntity{
								ID: uuid.New(),
							},
							ServiceID: serviceID1,
							Action:    ServiceActionStart,
							State:     JobProcessing,
						},
					}, nil
				}

				// Mock finding service
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					return &Service{
						BaseEntity: BaseEntity{
							ID: id,
						},
						CurrentState: ServiceStarting,
					}, nil
				}

				// Mock saving job
				jobRepo.saveFunc = func(ctx context.Context, job *Job) error {
					return nil
				}

				// Mock saving service with error
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					return errors.New("error saving service")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "Error marking timed out job",
			wantCount:  0,
		},
		{
			name: "Successfully process timed out jobs",
			setupMocks: func(store *MockStore) {
				jobRepo := &MockJobRepository{}
				store.WithJobRepo(jobRepo)
				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock getting timed out jobs
				jobRepo.getTimeOutJobsFunc = func(ctx context.Context, timeout time.Duration) ([]*Job, error) {
					return []*Job{
						{
							BaseEntity: BaseEntity{
								ID: uuid.New(),
							},
							ServiceID: serviceID1,
							Action:    ServiceActionStart,
							State:     JobProcessing,
						},
						{
							BaseEntity: BaseEntity{
								ID: uuid.New(),
							},
							ServiceID: serviceID2,
							Action:    ServiceActionStop,
							State:     JobProcessing,
						},
					}, nil
				}

				// Mock finding service
				serviceRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Service, error) {
					var state ServiceState
					if id == serviceID1 {
						state = ServiceStarting
					} else {
						state = ServiceStopping
					}
					return &Service{
						BaseEntity: BaseEntity{
							ID: id,
						},
						CurrentState: state,
					}, nil
				}

				// Mock saving job
				jobRepo.saveFunc = func(ctx context.Context, job *Job) error {
					assert.Equal(t, JobFailed, job.State)
					assert.NotNil(t, job.CompletedAt)
					assert.Contains(t, job.ErrorMessage, "exceeding maximum processing time")
					return nil
				}

				// Mock saving service
				serviceRepo.saveFunc = func(ctx context.Context, service *Service) error {
					assert.NotNil(t, service.ErrorMessage)
					assert.NotNil(t, service.FailedAction)
					return nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:   false,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}

			tt.setupMocks(store)
			commander := NewServiceCommander(store, audit)

			count, err := commander.FailTimeoutServicesAndJobs(ctx, timeout)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}
		})
	}
}
