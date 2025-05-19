package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceGroup_Validate(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name       string
		sg         *ServiceGroup
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid service group",
			sg: &ServiceGroup{
				Name:          "Test Group",
				ParticipantID: validID,
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			sg: &ServiceGroup{
				Name:          "",
				ParticipantID: validID,
			},
			wantErr:    true,
			errMessage: "service group name cannot be empty",
		},
		{
			name: "Nil broker ID",
			sg: &ServiceGroup{
				Name:          "Test Group",
				ParticipantID: uuid.Nil,
			},
			wantErr:    true,
			errMessage: "service group broker cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sg.Validate()
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

func TestServiceGroup_TableName(t *testing.T) {
	sg := ServiceGroup{}
	assert.Equal(t, "service_groups", sg.TableName())
}

func TestServiceGroupCommander_Create(t *testing.T) {
	ctx := context.Background()
	groupID := uuid.New()
	brokerID := uuid.New()
	validName := "Test Group"

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Create success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				// Set up broker repo
				participantRepo := &MockParticipantRepository{}
				store.WithParticipantRepo(participantRepo)

				// Mock broker FindByID
				broker := &Participant{
					BaseEntity: BaseEntity{
						ID: brokerID,
					},
					Name: "Test Broker",
				}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					assert.Equal(t, brokerID, id)
					return broker, nil
				}

				// Set up service group repo
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				// Mock service group Create
				sgRepo.createFunc = func(ctx context.Context, sg *ServiceGroup) error {
					sg.ID = groupID
					assert.Equal(t, validName, sg.Name)
					assert.Equal(t, brokerID, sg.ParticipantID)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceGroupCreated, eventType)
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
			name: "Broker not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				store.WithParticipantRepo(participantRepo)

				// Mock broker Exists with error
				participantRepo.existsFunc = func(ctx context.Context, id UUID) (bool, error) {
					return false, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "broker with ID",
		},
		{
			name: "Validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				store.WithParticipantRepo(participantRepo)

				// Mock broker FindByID
				broker := &Participant{
					BaseEntity: BaseEntity{
						ID: brokerID,
					},
					Name: "Test Broker",
				}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return broker, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "service group name cannot be empty",
		},
		{
			name: "Create error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				store.WithParticipantRepo(participantRepo)

				// Mock broker FindByID
				broker := &Participant{
					BaseEntity: BaseEntity{
						ID: brokerID,
					},
					Name: "Test Broker",
				}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return broker, nil
				}

				// Set up service group repo
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				// Mock service group Create with error
				sgRepo.createFunc = func(ctx context.Context, sg *ServiceGroup) error {
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
				participantRepo := &MockParticipantRepository{}
				store.WithParticipantRepo(participantRepo)

				// Mock broker FindByID
				broker := &Participant{
					BaseEntity: BaseEntity{
						ID: brokerID,
					},
					Name: "Test Broker",
				}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return broker, nil
				}

				// Set up service group repo
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				// Mock service group Create
				sgRepo.createFunc = func(ctx context.Context, sg *ServiceGroup) error {
					sg.ID = groupID
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

			// Special case for validation error
			if tt.name == "Validation error" {
				tt.setupMocks(store, audit)
				commander := NewServiceGroupCommander(store, audit)
				sg, err := commander.Create(ctx, "", brokerID) // Empty name should cause validation error

				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
				assert.Nil(t, sg)
			} else {
				tt.setupMocks(store, audit)
				commander := NewServiceGroupCommander(store, audit)
				sg, err := commander.Create(ctx, validName, brokerID)

				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMessage != "" {
						assert.Contains(t, err.Error(), tt.errMessage)
					}
					assert.Nil(t, sg)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, sg)
					assert.Equal(t, groupID, sg.ID)
					assert.Equal(t, validName, sg.Name)
					assert.Equal(t, brokerID, sg.ParticipantID)
				}
			}
		})
	}
}

func TestServiceGroupCommander_Update(t *testing.T) {
	ctx := context.Background()
	groupID := uuid.New()
	brokerID := uuid.New()
	existingName := "Existing Group"
	newName := "Updated Group"

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Update success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				// Mock FindByID
				existingSg := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					Name:          existingName,
					ParticipantID: brokerID,
				}
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					assert.Equal(t, groupID, id)
					return existingSg, nil
				}

				// Mock Save
				sgRepo.updateFunc = func(ctx context.Context, sg *ServiceGroup) error {
					assert.Equal(t, groupID, sg.ID)
					assert.Equal(t, newName, sg.Name)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceGroupUpdated, eventType)
					assert.NotNil(t, entityID)

					// Verify before object
					beforeSg, ok := before.(*ServiceGroup)
					assert.True(t, ok)
					assert.Equal(t, existingName, beforeSg.Name)

					// Verify after object
					afterSg, ok := after.(*ServiceGroup)
					assert.True(t, ok)
					assert.Equal(t, newName, afterSg.Name)

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
			name: "Service group not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				// Mock FindByID with error
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return nil, NewNotFoundErrorf("service group not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				// Mock FindByID
				existingSg := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					Name:          existingName,
					ParticipantID: brokerID,
				}
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return existingSg, nil
				}
			},
			wantErr:    true,
			errMessage: "service group name cannot be empty",
		},
		{
			name: "Save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				// Mock FindByID
				existingSg := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					Name:          existingName,
					ParticipantID: brokerID,
				}
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return existingSg, nil
				}

				// Mock Save with error
				sgRepo.updateFunc = func(ctx context.Context, sg *ServiceGroup) error {
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
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				// Mock FindByID
				existingSg := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					Name:          existingName,
					ParticipantID: brokerID,
				}
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return existingSg, nil
				}

				// Mock Save
				sgRepo.updateFunc = func(ctx context.Context, sg *ServiceGroup) error {
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

			// Special case for validation error
			if tt.name == "Validation error" {
				tt.setupMocks(store, audit)
				commander := NewServiceGroupCommander(store, audit)
				emptyName := ""
				sg, err := commander.Update(ctx, groupID, &emptyName)

				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
				assert.Nil(t, sg)
			} else {
				tt.setupMocks(store, audit)
				commander := NewServiceGroupCommander(store, audit)
				sg, err := commander.Update(ctx, groupID, &newName)

				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMessage != "" {
						assert.Contains(t, err.Error(), tt.errMessage)
					}
					assert.Nil(t, sg)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, sg)
					assert.Equal(t, groupID, sg.ID)
					assert.Equal(t, newName, sg.Name)
				}
			}
		})
	}
}

func TestServiceGroupCommander_Delete(t *testing.T) {
	ctx := context.Background()
	groupID := uuid.New()
	brokerID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Delete success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock FindByID
				existingSg := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					Name:          "Test Group",
					ParticipantID: brokerID,
				}
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					assert.Equal(t, groupID, id)
					return existingSg, nil
				}

				// Mock CountByGroup to return 0 (no associated services)
				serviceRepo.countByGroupFunc = func(ctx context.Context, id UUID) (int64, error) {
					assert.Equal(t, groupID, id)
					return 0, nil
				}

				// Mock Delete
				sgRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					assert.Equal(t, groupID, id)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeServiceGroupDeleted, eventType)
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
			name: "Service group not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				// Mock FindByID with error
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return nil, NewNotFoundErrorf("service group not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Has associated services",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock FindByID
				existingSg := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					Name:          "Test Group",
					ParticipantID: brokerID,
				}
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return existingSg, nil
				}

				// Mock CountByGroup to return non-zero (has associated services)
				serviceRepo.countByGroupFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 5, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "cannot delete service group with associated services",
		},
		{
			name: "Count error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock FindByID
				existingSg := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					Name:          "Test Group",
					ParticipantID: brokerID,
				}
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return existingSg, nil
				}

				// Mock CountByGroup with error
				serviceRepo.countByGroupFunc = func(ctx context.Context, id UUID) (int64, error) {
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
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock FindByID
				existingSg := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					Name:          "Test Group",
					ParticipantID: brokerID,
				}
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return existingSg, nil
				}

				// Mock CountByGroup to return 0 (no associated services)
				serviceRepo.countByGroupFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 0, nil
				}

				// Mock Delete with error
				sgRepo.deleteFunc = func(ctx context.Context, id UUID) error {
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
				sgRepo := &MockServiceGroupRepository{}
				store.WithServiceGroupRepo(sgRepo)

				serviceRepo := &MockServiceRepository{}
				store.WithServiceRepo(serviceRepo)

				// Mock FindByID
				existingSg := &ServiceGroup{
					BaseEntity: BaseEntity{
						ID: groupID,
					},
					Name:          "Test Group",
					ParticipantID: brokerID,
				}
				sgRepo.findByIDFunc = func(ctx context.Context, id UUID) (*ServiceGroup, error) {
					return existingSg, nil
				}

				// Mock CountByGroup to return 0 (no associated services)
				serviceRepo.countByGroupFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 0, nil
				}

				// Mock Delete
				sgRepo.deleteFunc = func(ctx context.Context, id UUID) error {
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

			commander := NewServiceGroupCommander(store, audit)
			err := commander.Delete(ctx, groupID)

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
