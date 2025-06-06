package domain

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParticipantStatus_Validate(t *testing.T) {
	tests := []struct {
		name    string
		status  ParticipantStatus
		wantErr bool
	}{
		{
			name:    "Enabled status",
			status:  ParticipantEnabled,
			wantErr: false,
		},
		{
			name:    "Disabled status",
			status:  ParticipantDisabled,
			wantErr: false,
		},
		{
			name:    "Invalid status",
			status:  "InvalidStatus",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.status.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseParticipantStatus(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    ParticipantStatus
		wantErr bool
	}{
		{
			name:    "Parse Enabled status",
			value:   "Enabled",
			want:    ParticipantEnabled,
			wantErr: false,
		},
		{
			name:    "Parse Disabled status",
			value:   "Disabled",
			want:    ParticipantDisabled,
			wantErr: false,
		},
		{
			name:    "Parse invalid status",
			value:   "InvalidStatus",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseParticipantStatus(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParticipant_TableName(t *testing.T) {
	participant := Participant{}
	assert.Equal(t, "participants", participant.TableName())
}

func TestParticipant_Validate(t *testing.T) {
	validID := uuid.New()
	_ = validID // prevent unused error for now, will be used later

	tests := []struct {
		name        string
		participant *Participant
		wantErr     bool
		errContains string
	}{
		{
			name: "Valid participant",
			participant: &Participant{
				Name:        "test-participant",
				Status:      ParticipantEnabled,
				CountryCode: "US",
				Attributes:  Attributes{"key": []string{"value"}},
			},
			wantErr: false,
		},
		{
			name: "Empty name",
			participant: &Participant{
				Name:        "",
				Status:      ParticipantEnabled,
				CountryCode: "US",
			},
			wantErr:     true,
			errContains: "participant name cannot be empty",
		},
		{
			name: "Invalid status",
			participant: &Participant{
				Name:        "test-participant",
				Status:      "InvalidStatus",
				CountryCode: "US",
			},
			wantErr:     true,
			errContains: "invalid participant status",
		},
		{
			name: "Invalid country code",
			participant: &Participant{
				Name:        "test-participant",
				Status:      ParticipantEnabled,
				CountryCode: "INVALID",
			},
			wantErr:     true,
			errContains: "invalid",
		},
		{
			name: "Valid with empty country code",
			participant: &Participant{
				Name:   "test-participant",
				Status: ParticipantEnabled,
			},
			wantErr: false,
		},
		{
			name: "Invalid attributes",
			participant: &Participant{
				Name:       "test-participant",
				Status:     ParticipantEnabled,
				Attributes: Attributes{"": []string{"value"}}, // Invalid key
			},
			wantErr:     true,
			errContains: "attribute keys cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.participant.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParticipantCommander_Create(t *testing.T) {
	ctx := context.Background()
	validName := "test-participant"
	validStatus := ParticipantEnabled
	validCountryCode := CountryCode("US")
	validAttributes := Attributes{"key": []string{"value"}}

	tests := []struct {
		name          string
		setupMocks    func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository)
		expectedError string
		wantErr       bool
	}{
		{
			name: "Create success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.createFunc = func(ctx context.Context, p *Participant) error {
					assert.Equal(t, validName, p.Name)
					assert.Equal(t, validStatus, p.Status)
					assert.Equal(t, validCountryCode, p.CountryCode)
					assert.Equal(t, validAttributes, p.Attributes)
					return nil
				}
				store.WithParticipantRepo(participantRepo)
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID *UUID, providerID *UUID, agentID *UUID, consumerID *UUID) (*AuditEntry, error) {
					// In this context, for participant creation, entityID and providerID are the participant's ID.
					assert.NotNil(t, entityID)
					assert.NotNil(t, providerID)
					assert.Equal(t, *entityID, *providerID) // Participant ID is used for both EntityID and ProviderID contextually
					assert.Nil(t, agentID)
					assert.Nil(t, consumerID)
					return &AuditEntry{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "Create validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				// No need to mock repo, validation happens before
			},
			expectedError: "participant name cannot be empty", // This will be wrapped in InvalidInputError
			wantErr:       true,
		},
		{
			name: "Create repo error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.createFunc = func(ctx context.Context, p *Participant) error {
					return errors.New("repo create error")
				}
				store.WithParticipantRepo(participantRepo)
			},
			expectedError: "repo create error",
			wantErr:       true,
		},
		{
			name: "Create audit error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.createFunc = func(ctx context.Context, p *Participant) error {
					return nil
				}
				store.WithParticipantRepo(participantRepo)
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID *UUID, providerID *UUID, agentID *UUID, consumerID *UUID) (*AuditEntry, error) {
					return nil, errors.New("audit create error")
				}
			},
			expectedError: "audit create error",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			agentRepo := &MockAgentRepository{} // Needed for Delete, but pass for consistency
			tokenRepo := &MockTokenRepository{} // Needed for Delete, but pass for consistency
			tt.setupMocks(store, audit, agentRepo, tokenRepo)

			commander := NewParticipantCommander(store, audit)
			var participant *Participant
			var err error

			if tt.name == "Create validation error" { // Special case for validation
				participant, err = commander.Create(ctx, "", validStatus, validCountryCode, validAttributes)
			} else {
				participant, err = commander.Create(ctx, validName, validStatus, validCountryCode, validAttributes)
			}

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
					if tt.name == "Create validation error" {
						var invalidInputErr InvalidInputError
						require.True(t, errors.As(err, &invalidInputErr), "error should be InvalidInputError")
					}
				}
				assert.Nil(t, participant)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, participant)
				assert.Equal(t, validName, participant.Name)
				assert.Equal(t, validStatus, participant.Status)
			}
		})
	}
}

func TestParticipantCommander_Update(t *testing.T) {
	ctx := context.Background()
	participantID := uuid.New()
	existingName := "existing-participant"
	existingStatus := ParticipantEnabled
	existingCountryCode := CountryCode("US")
	existingAttributes := Attributes{"old_key": []string{"old_value"}}

	newName := "updated-participant"
	newStatus := ParticipantDisabled
	newCountryCode := CountryCode("CA")
	newAttributes := Attributes{"new_key": []string{"new_value"}}

	tests := []struct {
		name            string
		setupMocks      func(store *MockStore, audit *MockAuditEntryCommander)
		updateName      *string
		updateStatus    *ParticipantStatus
		updateCountry   *CountryCode
		updateAttrs     *Attributes
		expectedName    string
		expectedStatus  ParticipantStatus
		expectedCountry CountryCode
		expectedAttrs   Attributes
		wantErr         bool
		expectedError   string
	}{
		{
			name: "Update success - all fields",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					require.Equal(t, participantID, id)
					return &Participant{
						BaseEntity:  BaseEntity{ID: participantID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
						Name:        existingName,
						Status:      existingStatus,
						CountryCode: existingCountryCode,
						Attributes:  existingAttributes,
					}, nil
				}
				participantRepo.saveFunc = func(ctx context.Context, p *Participant) error {
					assert.Equal(t, participantID, p.ID)
					assert.Equal(t, newName, p.Name)
					assert.Equal(t, newStatus, p.Status)
					assert.Equal(t, newCountryCode, p.CountryCode)
					assert.Equal(t, newAttributes, p.Attributes)
					return nil
				}
				store.WithParticipantRepo(participantRepo)
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID *UUID, providerID *UUID, agentID *UUID, consumerID *UUID, before interface{}, after interface{}) (*AuditEntry, error) {
					// In this context, for participant update, entityID and providerID are the participant's ID.
					assert.NotNil(t, entityID)
					assert.NotNil(t, providerID)
					assert.Equal(t, *entityID, *providerID) // Participant ID is used for both EntityID and ProviderID
					assert.Nil(t, agentID)
					assert.Nil(t, consumerID)
					return &AuditEntry{}, nil
				}
			},
			updateName:      &newName,
			updateStatus:    &newStatus,
			updateCountry:   &newCountryCode,
			updateAttrs:     &newAttributes,
			expectedName:    newName,
			expectedStatus:  newStatus,
			expectedCountry: newCountryCode,
			expectedAttrs:   newAttributes,
			wantErr:         false,
		},
		{
			name: "Update success - only name",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return &Participant{BaseEntity: BaseEntity{ID: participantID}, Name: existingName, Status: existingStatus, CountryCode: existingCountryCode, Attributes: existingAttributes}, nil
				}
				participantRepo.saveFunc = func(ctx context.Context, p *Participant) error {
					assert.Equal(t, newName, p.Name)
					assert.Equal(t, existingStatus, p.Status) // Unchanged
					return nil
				}
				store.WithParticipantRepo(participantRepo)
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID *UUID, providerID *UUID, agentID *UUID, consumerID *UUID, before interface{}, after interface{}) (*AuditEntry, error) {
					return &AuditEntry{}, nil
				}
			},
			updateName:      &newName,
			expectedName:    newName,
			expectedStatus:  existingStatus,
			expectedCountry: existingCountryCode,
			expectedAttrs:   existingAttributes,
			wantErr:         false,
		},
		{
			name: "Update participant not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return nil, NewNotFoundErrorf("participant with ID %s not found", id.String())
				}
				store.WithParticipantRepo(participantRepo)
			},
			updateName:    &newName,
			wantErr:       true,
			expectedError: "participant with ID " + participantID.String() + " not found",
		},
		{
			name: "Update validation error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return &Participant{BaseEntity: BaseEntity{ID: participantID}, Name: existingName, Status: existingStatus}, nil
				}
				store.WithParticipantRepo(participantRepo)
				// No need to mock save, validation happens before
			},
			updateName:    func() *string { s := ""; return &s }(), // Empty name
			wantErr:       true,
			expectedError: "participant name cannot be empty",
		},
		{
			name: "Update repo save error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return &Participant{BaseEntity: BaseEntity{ID: participantID}, Name: existingName, Status: existingStatus}, nil
				}
				participantRepo.saveFunc = func(ctx context.Context, p *Participant) error {
					return errors.New("repo save error")
				}
				store.WithParticipantRepo(participantRepo)
			},
			updateName:    &newName,
			wantErr:       true,
			expectedError: "repo save error",
		},
		{
			name: "Update audit error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return &Participant{BaseEntity: BaseEntity{ID: participantID}, Name: existingName, Status: existingStatus}, nil
				}
				participantRepo.saveFunc = func(ctx context.Context, p *Participant) error {
					return nil
				}
				store.WithParticipantRepo(participantRepo)
				audit.CreateCtxWithDiffFunc = func(ctx context.Context, eventType EventType, entityID *UUID, providerID *UUID, agentID *UUID, consumerID *UUID, before interface{}, after interface{}) (*AuditEntry, error) {
					return nil, errors.New("audit update error")
				}
			},
			updateName:    &newName,
			wantErr:       true,
			expectedError: "audit update error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			tt.setupMocks(store, audit)

			commander := NewParticipantCommander(store, audit)
			participant, err := commander.Update(ctx, participantID, tt.updateName, tt.updateStatus, tt.updateCountry, tt.updateAttrs)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
					if tt.name == "Update validation error" {
						var invalidInputErr InvalidInputError
						require.True(t, errors.As(err, &invalidInputErr), "error should be InvalidInputError")
					}
				}
				assert.Nil(t, participant)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, participant)
				assert.Equal(t, tt.expectedName, participant.Name)
				assert.Equal(t, tt.expectedStatus, participant.Status)
				assert.Equal(t, tt.expectedCountry, participant.CountryCode)
				assert.Equal(t, tt.expectedAttrs, participant.Attributes)
			}
		})
	}
}

func TestParticipantCommander_Delete(t *testing.T) {
	ctx := context.Background()
	participantID := uuid.New()

	tests := []struct {
		name          string
		setupMocks    func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository)
		wantErr       bool
		expectedError string
	}{
		{
			name: "Delete success",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					require.Equal(t, participantID, id)
					return &Participant{BaseEntity: BaseEntity{ID: participantID}, Name: "test"}, nil
				}
				agentRepo.countByProviderFunc = func(ctx context.Context, pid UUID) (int64, error) {
					require.Equal(t, participantID, pid)
					return 0, nil
				}
				tokenRepo.deleteByParticipantIDFunc = func(ctx context.Context, pid UUID) error {
					require.Equal(t, participantID, pid)
					return nil
				}
				participantRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					require.Equal(t, participantID, id)
					return nil
				}
				store.WithParticipantRepo(participantRepo)
				store.WithAgentRepo(agentRepo) // Ensure agentRepo is set on the mock store
				store.WithTokenRepo(tokenRepo) // Ensure tokenRepo is set on the mock store

				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID *UUID, providerID *UUID, agentID *UUID, consumerID *UUID) (*AuditEntry, error) {
					// For participant deletion, entityID and providerID are the participant's ID.
					assert.NotNil(t, entityID)
					assert.NotNil(t, providerID)
					assert.Equal(t, *entityID, *providerID)
					assert.Nil(t, agentID)
					assert.Nil(t, consumerID)
					return &AuditEntry{}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "Delete participant not found",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return nil, NewNotFoundErrorf("participant with ID %s not found", id.String())
				}
				store.WithParticipantRepo(participantRepo)
			},
			wantErr:       true,
			expectedError: "participant with ID " + participantID.String() + " not found",
		},
		{
			name: "Delete with dependent agents",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return &Participant{BaseEntity: BaseEntity{ID: participantID}}, nil
				}
				agentRepo.countByProviderFunc = func(ctx context.Context, pid UUID) (int64, error) {
					return 1, nil
				}
				store.WithParticipantRepo(participantRepo)
				store.WithAgentRepo(agentRepo)
			},
			wantErr:       true,
			expectedError: fmt.Sprintf("cannot delete participant %s: 1 dependent agent(s) exist", participantID),
		},
		{
			name: "Delete agent count error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return &Participant{BaseEntity: BaseEntity{ID: participantID}}, nil
				}
				agentRepo.countByProviderFunc = func(ctx context.Context, pid UUID) (int64, error) {
					return 0, errors.New("agent count error")
				}
				store.WithParticipantRepo(participantRepo)
				store.WithAgentRepo(agentRepo)
			},
			wantErr:       true,
			expectedError: "failed to count agents for participant",
		},
		{
			name: "Delete token deletion error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return &Participant{BaseEntity: BaseEntity{ID: participantID}}, nil
				}
				agentRepo.countByProviderFunc = func(ctx context.Context, pid UUID) (int64, error) {
					return 0, nil
				}
				tokenRepo.deleteByParticipantIDFunc = func(ctx context.Context, pid UUID) error {
					return errors.New("token delete error")
				}
				store.WithParticipantRepo(participantRepo)
				store.WithAgentRepo(agentRepo)
				store.WithTokenRepo(tokenRepo)
			},
			wantErr:       true,
			expectedError: "failed to delete tokens for participant",
		},
		{
			name: "Delete participant repo error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return &Participant{BaseEntity: BaseEntity{ID: participantID}}, nil
				}
				agentRepo.countByProviderFunc = func(ctx context.Context, pid UUID) (int64, error) {
					return 0, nil
				}
				tokenRepo.deleteByParticipantIDFunc = func(ctx context.Context, pid UUID) error {
					return nil
				}
				participantRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					return errors.New("repo delete error")
				}
				store.WithParticipantRepo(participantRepo)
				store.WithAgentRepo(agentRepo)
				store.WithTokenRepo(tokenRepo)
			},
			wantErr:       true,
			expectedError: "repo delete error",
		},
		{
			name: "Delete audit error",
			setupMocks: func(store *MockStore, audit *MockAuditEntryCommander, agentRepo *MockAgentRepository, tokenRepo *MockTokenRepository) {
				participantRepo := &MockParticipantRepository{}
				participantRepo.findByIDFunc = func(ctx context.Context, id UUID) (*Participant, error) {
					return &Participant{BaseEntity: BaseEntity{ID: participantID}}, nil
				}
				agentRepo.countByProviderFunc = func(ctx context.Context, pid UUID) (int64, error) {
					return 0, nil
				}
				tokenRepo.deleteByParticipantIDFunc = func(ctx context.Context, pid UUID) error {
					return nil
				}
				participantRepo.deleteFunc = func(ctx context.Context, id UUID) error {
					return nil
				}
				store.WithParticipantRepo(participantRepo)
				store.WithAgentRepo(agentRepo)
				store.WithTokenRepo(tokenRepo)
				audit.CreateCtxFunc = func(ctx context.Context, eventType EventType, properties JSON, entityID *UUID, providerID *UUID, agentID *UUID, consumerID *UUID) (*AuditEntry, error) {
					return nil, errors.New("audit delete error")
				}
			},
			wantErr:       true,
			expectedError: "audit delete error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			audit := &MockAuditEntryCommander{}
			// Need to initialize mock agent and token repos for delete tests
			agentRepo := &MockAgentRepository{}
			tokenRepo := &MockTokenRepository{}

			tt.setupMocks(store, audit, agentRepo, tokenRepo)

			commander := NewParticipantCommander(store, audit)
			err := commander.Delete(ctx, participantID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
					if tt.name == "Delete with dependent agents" {
						var invalidInputErr InvalidInputError
						require.True(t, errors.As(err, &invalidInputErr), "error should be InvalidInputError")
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
