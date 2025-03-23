package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProviderState_Validate(t *testing.T) {
	tests := []struct {
		name       string
		state      ProviderState
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Valid Enabled state",
			state:   ProviderEnabled,
			wantErr: false,
		},
		{
			name:    "Valid Disabled state",
			state:   ProviderDisabled,
			wantErr: false,
		},
		{
			name:       "Invalid state",
			state:      "InvalidState",
			wantErr:    true,
			errMessage: "invalid provider state",
		},
		{
			name:       "Empty state",
			state:      "",
			wantErr:    true,
			errMessage: "invalid provider state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
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

func TestParseProviderState(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       ProviderState
		wantErr    bool
		errMessage string
	}{
		{
			name:    "Valid Enabled state",
			input:   "Enabled",
			want:    ProviderEnabled,
			wantErr: false,
		},
		{
			name:    "Valid Disabled state",
			input:   "Disabled",
			want:    ProviderDisabled,
			wantErr: false,
		},
		{
			name:       "Invalid state",
			input:      "InvalidState",
			wantErr:    true,
			errMessage: "invalid provider state",
		},
		{
			name:       "Empty state",
			input:      "",
			wantErr:    true,
			errMessage: "invalid provider state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, err := ParseProviderState(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Equal(t, ProviderState(""), state)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, state)
			}
		})
	}
}

func TestProvider_TableName(t *testing.T) {
	provider := Provider{}
	assert.Equal(t, "providers", provider.TableName())
}

func TestProvider_Validate(t *testing.T) {
	validID := uuid.New()

	tests := []struct {
		name       string
		provider   *Provider
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid provider",
			provider: &Provider{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name:        "AWS",
				CountryCode: "US",
				State:       ProviderEnabled,
				Attributes: Attributes{
					"region": {"us-west-2"},
					"tier":   {"premium"},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			provider: &Provider{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name:        "",
				CountryCode: "US",
				State:       ProviderEnabled,
			},
			wantErr:    true,
			errMessage: "provider name cannot be empty",
		},
		{
			name: "Invalid country code",
			provider: &Provider{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name:        "AWS",
				CountryCode: "USA", // Invalid: not two uppercase letters
				State:       ProviderEnabled,
			},
			wantErr:    true,
			errMessage: "invalid lentgh for ISO 3166-1 alpha-2 country code",
		},
		{
			name: "Invalid state",
			provider: &Provider{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name:        "AWS",
				CountryCode: "US",
				State:       "InvalidState",
			},
			wantErr:    true,
			errMessage: "invalid provider state",
		},
		{
			name: "Invalid attributes",
			provider: &Provider{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name:        "AWS",
				CountryCode: "US",
				State:       ProviderEnabled,
				Attributes: Attributes{
					"region": {""}, // Empty value
				},
			},
			wantErr:    true,
			errMessage: "has an empty value",
		},
		{
			name: "Nil attributes",
			provider: &Provider{
				BaseEntity: BaseEntity{
					ID: validID,
				},
				Name:        "AWS",
				CountryCode: "US",
				State:       ProviderEnabled,
				Attributes:  nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.provider.Validate()
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

func TestProviderCommander_Create(t *testing.T) {
	ctx := context.Background()
	providerID := uuid.New()
	validName := "AWS"
	validState := ProviderEnabled
	validCountryCode := CountryCode("US")
	validAttributes := Attributes{"region": {"us-west-2"}}

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Create success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				// Mock Create to set the ID
				providerRepo.createFunc = func(ctx context.Context, provider *Provider) error {
					provider.ID = providerID
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeProviderCreated, eventType)
					assert.NotNil(t, properties)
					// Don't compare exact pointers, just verify they're not nil and point to the expected UUID
					assert.NotNil(t, entityID)
					assert.NotNil(t, providerID)
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
			errMessage: "invalid input",
		},
		{
			name: "Create repository error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				// Mock Create to return error
				providerRepo.createFunc = func(ctx context.Context, provider *Provider) error {
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
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				// Mock Create
				providerRepo.createFunc = func(ctx context.Context, provider *Provider) error {
					provider.ID = providerID
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
				commander := NewProviderCommander(store, audit)
				provider, err := commander.Create(ctx, "", validState, "INVALID", validAttributes)

				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
				assert.Nil(t, provider)
			} else {
				tt.setupMocks(store, audit)
				commander := NewProviderCommander(store, audit)
				provider, err := commander.Create(ctx, validName, validState, validCountryCode, validAttributes)

				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMessage != "" {
						assert.Contains(t, err.Error(), tt.errMessage)
					}
					assert.Nil(t, provider)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, provider)
					assert.Equal(t, providerID, provider.ID)
					assert.Equal(t, validName, provider.Name)
					assert.Equal(t, validState, provider.State)
					assert.Equal(t, validCountryCode, provider.CountryCode)
					assert.Equal(t, validAttributes, provider.Attributes)
				}
			}
		})
	}
}

func TestProviderCommander_Update(t *testing.T) {
	ctx := context.Background()
	providerID := uuid.New()
	existingName := "AWS"
	newName := "Azure"
	validState := ProviderEnabled
	newState := ProviderDisabled
	validCountryCode := CountryCode("US")
	newCountryCode := CountryCode("CA")
	existingAttributes := Attributes{"region": {"us-west-2"}}
	newAttributes := Attributes{"region": {"canada-central"}}

	tests := []struct {
		name              string
		setupMocks        func(store *MockStore, audit *MockAuditEntryCommander)
		updateName        *string
		updateState       *ProviderState
		updateCountryCode *CountryCode
		updateAttributes  *Attributes
		wantErr           bool
		errMessage        string
	}{
		{
			name: "Update all fields",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				// Existing provider
				existingProvider := &Provider{
					BaseEntity: BaseEntity{
						ID: providerID,
					},
					Name:        existingName,
					State:       validState,
					CountryCode: validCountryCode,
					Attributes:  existingAttributes,
				}

				// Mock FindByID
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					assert.Equal(t, providerID, id)
					return existingProvider, nil
				}

				// Mock Save
				providerRepo.updateFunc = func(ctx context.Context, provider *Provider) error {
					assert.Equal(t, providerID, provider.ID)
					assert.Equal(t, newName, provider.Name)
					assert.Equal(t, newState, provider.State)
					assert.Equal(t, newCountryCode, provider.CountryCode)
					assert.Equal(t, newAttributes, provider.Attributes)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					assert.Equal(t, EventTypeProviderUpdated, eventType)
					assert.NotNil(t, entityID)

					// Verify before object
					beforeProvider, ok := before.(*Provider)
					assert.True(t, ok)
					assert.Equal(t, existingName, beforeProvider.Name)

					// Verify after object
					afterProvider, ok := after.(*Provider)
					assert.True(t, ok)
					assert.Equal(t, newName, afterProvider.Name)

					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			updateName:        &newName,
			updateState:       &newState,
			updateCountryCode: &newCountryCode,
			updateAttributes:  &newAttributes,
			wantErr:           false,
		},
		{
			name: "Update only name",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				// Existing provider
				existingProvider := &Provider{
					BaseEntity: BaseEntity{
						ID: providerID,
					},
					Name:        existingName,
					State:       validState,
					CountryCode: validCountryCode,
					Attributes:  existingAttributes,
				}

				// Mock FindByID
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					return existingProvider, nil
				}

				// Mock Save
				providerRepo.updateFunc = func(ctx context.Context, provider *Provider) error {
					assert.Equal(t, newName, provider.Name)
					assert.Equal(t, validState, provider.State)              // unchanged
					assert.Equal(t, validCountryCode, provider.CountryCode)  // unchanged
					assert.Equal(t, existingAttributes, provider.Attributes) // unchanged
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error) {
					return &AuditEntry{}, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			updateName:        &newName,
			updateState:       nil,
			updateCountryCode: nil,
			updateAttributes:  nil,
			wantErr:           false,
		},
		{
			name: "Provider not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				// Mock FindByID to return not found error
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					return nil, NewNotFoundErrorf("provider not found")
				}
			},
			updateName: &newName,
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				// Existing provider
				existingProvider := &Provider{
					BaseEntity: BaseEntity{
						ID: providerID,
					},
					Name:        existingName,
					State:       validState,
					CountryCode: validCountryCode,
					Attributes:  existingAttributes,
				}

				// Mock FindByID
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					return existingProvider, nil
				}
			},
			updateName: strPtr(""), // Invalid empty name
			wantErr:    true,
			errMessage: "provider name cannot be empty",
		},
		{
			name: "Save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				// Existing provider
				existingProvider := &Provider{
					BaseEntity: BaseEntity{
						ID: providerID,
					},
					Name:        existingName,
					State:       validState,
					CountryCode: validCountryCode,
					Attributes:  existingAttributes,
				}

				// Mock FindByID
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					return existingProvider, nil
				}

				// Mock Save with error
				providerRepo.updateFunc = func(ctx context.Context, provider *Provider) error {
					return errors.New("database error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			updateName: &newName,
			wantErr:    true,
			errMessage: "database error",
		},
		{
			name: "Audit entry error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				// Existing provider
				existingProvider := &Provider{
					BaseEntity: BaseEntity{
						ID: providerID,
					},
					Name:        existingName,
					State:       validState,
					CountryCode: validCountryCode,
					Attributes:  existingAttributes,
				}

				// Mock FindByID
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					return existingProvider, nil
				}

				// Mock Save
				providerRepo.updateFunc = func(ctx context.Context, provider *Provider) error {
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
			updateName: &newName,
			wantErr:    true,
			errMessage: "audit entry error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewProviderCommander(store, audit)
			provider, err := commander.Update(ctx, providerID, tt.updateName, tt.updateState, tt.updateCountryCode, tt.updateAttributes)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)

				// Check that the fields were updated
				if tt.updateName != nil {
					assert.Equal(t, *tt.updateName, provider.Name)
				}
				if tt.updateState != nil {
					assert.Equal(t, *tt.updateState, provider.State)
				}
				if tt.updateCountryCode != nil {
					assert.Equal(t, *tt.updateCountryCode, provider.CountryCode)
				}
				if tt.updateAttributes != nil {
					assert.Equal(t, *tt.updateAttributes, provider.Attributes)
				}
			}
		})
	}
}

func TestProviderCommander_Delete(t *testing.T) {
	ctx := context.Background()
	providerID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(store *MockStore, audit *MockAuditEntryCommander)
		wantErr    bool
		errMessage string
	}{
		{
			name: "Delete success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID
				existingProvider := &Provider{
					BaseEntity: BaseEntity{
						ID: providerID,
					},
					Name:        "AWS",
					State:       ProviderEnabled,
					CountryCode: "US",
				}
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					assert.Equal(t, providerID, id)
					return existingProvider, nil
				}

				// Mock CountByProvider to return 0 (no associated agents)
				agentRepo.countByProviderFunc = func(ctx context.Context, id UUID) (int64, error) {
					assert.Equal(t, providerID, id)
					return 0, nil
				}

				// Mock DeleteByProviderID
				tokenRepo.deleteByProviderIDFunc = func(ctx context.Context, id UUID) error {
					assert.Equal(t, providerID, id)
					return nil
				}

				// Mock Delete
				providerRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					assert.Equal(t, providerID, id)
					return nil
				}

				// Mock audit entry creation
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
					assert.Equal(t, EventTypeProviderDeleted, eventType)
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
			name: "Provider not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				// Mock FindByID to return not found error
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					return nil, NewNotFoundErrorf("provider not found")
				}
			},
			wantErr:    true,
			errMessage: "not found",
		},
		{
			name: "Has associated agents",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				// Mock FindByID
				existingProvider := &Provider{
					BaseEntity: BaseEntity{
						ID: providerID,
					},
					Name:        "AWS",
					State:       ProviderEnabled,
					CountryCode: "US",
				}
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					return existingProvider, nil
				}

				// Mock CountByProvider to return non-zero (has associated agents)
				agentRepo.countByProviderFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 5, nil
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "cannot delete provider with associated agents",
		},
		{
			name: "Error deleting tokens",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID
				existingProvider := &Provider{
					BaseEntity: BaseEntity{
						ID: providerID,
					},
					Name:        "AWS",
					State:       ProviderEnabled,
					CountryCode: "US",
				}
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					return existingProvider, nil
				}

				// Mock CountByProvider to return 0 (no associated agents)
				agentRepo.countByProviderFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 0, nil
				}

				// Mock DeleteByProviderID with error
				tokenRepo.deleteByProviderIDFunc = func(ctx context.Context, id UUID) error {
					return errors.New("token delete error")
				}

				// Make atomic work correctly
				store.atomicFunc = func(ctx context.Context, fn func(Store) error) error {
					return fn(store)
				}
			},
			wantErr:    true,
			errMessage: "token delete error",
		},
		{
			name: "Provider delete error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID
				existingProvider := &Provider{
					BaseEntity: BaseEntity{
						ID: providerID,
					},
					Name:        "AWS",
					State:       ProviderEnabled,
					CountryCode: "US",
				}
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					return existingProvider, nil
				}

				// Mock CountByProvider to return 0 (no associated agents)
				agentRepo.countByProviderFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 0, nil
				}

				// Mock DeleteByProviderID
				tokenRepo.deleteByProviderIDFunc = func(ctx context.Context, id UUID) error {
					return nil
				}

				// Mock Delete with error
				providerRepo.deleteFunc = func(ctx context.Context, id UUID) error {
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
				providerRepo := &MockProviderRepository{}
				store.WithProviderRepo(providerRepo)

				agentRepo := &MockAgentRepository{}
				store.WithAgentRepo(agentRepo)

				tokenRepo := &MockTokenRepository{}
				store.WithTokenRepo(tokenRepo)

				// Mock FindByID
				existingProvider := &Provider{
					BaseEntity: BaseEntity{
						ID: providerID,
					},
					Name:        "AWS",
					State:       ProviderEnabled,
					CountryCode: "US",
				}
				providerRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Provider, error) {
					return existingProvider, nil
				}

				// Mock CountByProvider to return 0 (no associated agents)
				agentRepo.countByProviderFunc = func(ctx context.Context, id UUID) (int64, error) {
					return 0, nil
				}

				// Mock DeleteByProviderID
				tokenRepo.deleteByProviderIDFunc = func(ctx context.Context, id UUID) error {
					return nil
				}

				// Mock Delete
				providerRepo.deleteFunc = func(ctx context.Context, id UUID) error {
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

			commander := NewProviderCommander(store, audit)
			err := commander.Delete(ctx, providerID)

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

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}
