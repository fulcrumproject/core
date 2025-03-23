package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBroker_Validate(t *testing.T) {
	tests := []struct {
		name    string
		broker  *Broker
		wantErr bool
	}{
		{
			name: "Valid broker",
			broker: &Broker{
				Name: "test-broker",
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			broker: &Broker{
				Name: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.broker.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "broker name cannot be empty")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBroker_TableName(t *testing.T) {
	broker := Broker{}
	assert.Equal(t, "brokers", broker.TableName())
}

func TestBrokerCommander_Create(t *testing.T) {
	ctx := context.Background()
	validName := "test-broker"

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		brokerName string
		wantErr    bool
		errorCheck func(t *testing.T, err error)
	}{
		{
			name: "Create success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.createFunc = func(ctx context.Context, broker *Broker) error {
					// Verify broker fields
					assert.Equal(t, validName, broker.Name)

					// Set an ID to simulate DB save
					broker.ID = uuid.New()
					return nil
				}

				store.WithBrokerRepo(brokerRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			brokerName: validName,
			wantErr:    false,
		},
		{
			name: "Empty name",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				// We don't need to set up anything specific here since validation happens before DB call

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			brokerName: "",
			wantErr:    true,
			errorCheck: func(t *testing.T, err error) {
				var invalidInputErr InvalidInputError
				assert.True(t, errors.As(err, &invalidInputErr))
				assert.Contains(t, err.Error(), "broker name cannot be empty")
			},
		},
		{
			name: "Repository error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.createFunc = func(ctx context.Context, broker *Broker) error {
					return errors.New("repository error")
				}

				store.WithBrokerRepo(brokerRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			brokerName: validName,
			wantErr:    true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "repository error")
			},
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.createFunc = func(ctx context.Context, broker *Broker) error {
					// Set an ID to simulate DB save
					broker.ID = uuid.New()
					return nil
				}

				store.WithBrokerRepo(brokerRepo)

				// Set up audit error
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, data JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					return nil, errors.New("audit error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			brokerName: validName,
			wantErr:    true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "audit error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewBrokerCommander(store, audit)
			broker, err := commander.Create(ctx, tt.brokerName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, broker)
				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, broker)
				assert.Equal(t, tt.brokerName, broker.Name)
				assert.NotEqual(t, uuid.Nil, broker.ID)
			}
		})
	}
}

func TestBrokerCommander_Update(t *testing.T) {
	ctx := context.Background()
	brokerID := uuid.New()
	existingName := "existing-broker"
	newName := "updated-broker"

	existingBroker := &Broker{
		BaseEntity: BaseEntity{
			ID: brokerID,
		},
		Name: existingName,
	}

	tests := []struct {
		testName   string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		nameParam  *string
		wantErr    bool
		errorCheck func(t *testing.T, err error)
	}{
		{
			testName: "Update name success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					assert.Equal(t, brokerID, id)
					copy := *existingBroker
					return &copy, nil
				}

				brokerRepo.updateFunc = func(ctx context.Context, broker *Broker) error {
					assert.Equal(t, newName, broker.Name)
					return nil
				}

				store.WithBrokerRepo(brokerRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			nameParam: &newName,
			wantErr:   false,
		},
		{
			testName: "Broker not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					return nil, NewNotFoundErrorf("broker not found")
				}

				store.WithBrokerRepo(brokerRepo)
			},
			nameParam: &newName,
			wantErr:   true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			testName: "Empty name validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					copy := *existingBroker
					return &copy, nil
				}

				store.WithBrokerRepo(brokerRepo)
			},
			nameParam: func() *string { s := ""; return &s }(),
			wantErr:   true,
			errorCheck: func(t *testing.T, err error) {
				var invalidInputErr InvalidInputError
				assert.True(t, errors.As(err, &invalidInputErr))
				assert.Contains(t, err.Error(), "broker name cannot be empty")
			},
		},
		{
			testName: "Repository error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					copy := *existingBroker
					return &copy, nil
				}

				brokerRepo.updateFunc = func(ctx context.Context, broker *Broker) error {
					return errors.New("repository error")
				}

				store.WithBrokerRepo(brokerRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			nameParam: &newName,
			wantErr:   true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "repository error")
			},
		},
		{
			testName: "Audit error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					copy := *existingBroker
					return &copy, nil
				}

				brokerRepo.updateFunc = func(ctx context.Context, broker *Broker) error {
					return nil
				}

				store.WithBrokerRepo(brokerRepo)

				// Set up audit error
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					return nil, errors.New("audit error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			nameParam: &newName,
			wantErr:   true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "audit error")
			},
		},
		{
			testName: "No changes (nil name parameter)",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					copy := *existingBroker
					return &copy, nil
				}

				brokerRepo.updateFunc = func(ctx context.Context, broker *Broker) error {
					assert.Equal(t, existingName, broker.Name) // Name should remain unchanged
					return nil
				}

				store.WithBrokerRepo(brokerRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			nameParam: nil,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewBrokerCommander(store, audit)
			broker, err := commander.Update(ctx, brokerID, tt.nameParam)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, broker)
				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, broker)

				// Verify updates
				if tt.nameParam != nil {
					assert.Equal(t, *tt.nameParam, broker.Name)
				} else {
					assert.Equal(t, existingName, broker.Name)
				}
			}
		})
	}
}

func TestBrokerCommander_Delete(t *testing.T) {
	ctx := context.Background()
	brokerID := uuid.New()

	existingBroker := &Broker{
		BaseEntity: BaseEntity{
			ID: brokerID,
		},
		Name: "test-broker",
	}

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errorCheck func(t *testing.T, err error)
	}{
		{
			name: "Delete success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					assert.Equal(t, brokerID, id)
					return existingBroker, nil
				}

				brokerRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					assert.Equal(t, brokerID, id)
					return nil
				}

				tokenRepo := &MockTokenRepository{}
				tokenRepo.deleteByBrokerIDFunc = func(ctx context.Context, id UUID) error {
					assert.Equal(t, brokerID, id)
					return nil
				}

				store.WithBrokerRepo(brokerRepo)
				store.WithTokenRepo(tokenRepo)

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
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					return nil, NewNotFoundErrorf("broker not found")
				}

				store.WithBrokerRepo(brokerRepo)
			},
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Token deletion error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					return existingBroker, nil
				}

				tokenRepo := &MockTokenRepository{}
				tokenRepo.deleteByBrokerIDFunc = func(ctx context.Context, id UUID) error {
					return errors.New("token deletion error")
				}

				store.WithBrokerRepo(brokerRepo)
				store.WithTokenRepo(tokenRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "token deletion error")
			},
		},
		{
			name: "Broker deletion error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					return existingBroker, nil
				}

				brokerRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					return errors.New("broker deletion error")
				}

				tokenRepo := &MockTokenRepository{}
				tokenRepo.deleteByBrokerIDFunc = func(ctx context.Context, id UUID) error {
					return nil
				}

				store.WithBrokerRepo(brokerRepo)
				store.WithTokenRepo(tokenRepo)

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "broker deletion error")
			},
		},
		{
			name: "Audit error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				brokerRepo := &MockBrokerRepository{}
				brokerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Broker, error) {
					return existingBroker, nil
				}

				brokerRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					return nil
				}

				tokenRepo := &MockTokenRepository{}
				tokenRepo.deleteByBrokerIDFunc = func(ctx context.Context, id UUID) error {
					return nil
				}

				store.WithBrokerRepo(brokerRepo)
				store.WithTokenRepo(tokenRepo)

				// Set up audit error
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, data JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					return nil, errors.New("audit error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "audit error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewBrokerCommander(store, audit)
			err := commander.Delete(ctx, brokerID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
