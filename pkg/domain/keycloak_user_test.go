package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateKeycloakUserParams_Validate(t *testing.T) {
	tests := []struct {
		name       string
		params     *CreateKeycloakUserParams
		wantErr    bool
		errMessage string
	}{
		{
			name: "Valid params",
			params: &CreateKeycloakUserParams{
				Username:  "some-username",
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "secret123",
				Role:      auth.RoleAdmin,
			},
			wantErr: false,
		},
		{
			name: "Missing username",
			params: &CreateKeycloakUserParams{
				Username:  "",
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "secret123",
				Role:      auth.RoleAdmin,
			},
			wantErr:    true,
			errMessage: "username is required",
		},
		{
			name: "Missing email",
			params: &CreateKeycloakUserParams{
				Username:  "some-username",
				Email:     "",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "secret123",
				Role:      auth.RoleAdmin,
			},
			wantErr:    true,
			errMessage: "email is required",
		},
		{
			name: "Missing first name",
			params: &CreateKeycloakUserParams{
				Username:  "some-username",
				Email:     "user@example.com",
				FirstName: "",
				LastName:  "Doe",
				Password:  "secret123",
				Role:      auth.RoleAdmin,
			},
			wantErr:    true,
			errMessage: "first name is required",
		},
		{
			name: "Missing last name",
			params: &CreateKeycloakUserParams{
				Username:  "some-username",
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "",
				Password:  "secret123",
				Role:      auth.RoleAdmin,
			},
			wantErr:    true,
			errMessage: "last name is required",
		},
		{
			name: "Missing password",
			params: &CreateKeycloakUserParams{
				Username:  "some-username",
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "",
				Role:      auth.RoleAdmin,
			},
			wantErr:    true,
			errMessage: "password is required",
		},
		{
			name: "Invalid role",
			params: &CreateKeycloakUserParams{
				Username:  "some-username",
				Email:     "user@example.com",
				FirstName: "John",
				LastName:  "Doe",
				Password:  "secret123",
				Role:      "invalid",
			},
			wantErr:    true,
			errMessage: "invalid role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorAs(t, err, &InvalidInputError{})
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- Commander helpers ---

func validCreateParams() CreateKeycloakUserParams {
	return CreateKeycloakUserParams{
		Username:  "john",
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Password:  "secret123",
		Enabled:   true,
		Role:      auth.RoleAdmin,
	}
}

func sampleKeycloakUser() *KeycloakUser {
	return &KeycloakUser{
		ID: "user-123", Username: "john", Email: "john@example.com",
		FirstName: "John", LastName: "Doe", Enabled: true, Roles: []string{"admin"},
	}
}

// --- Commander Create: validation ---

func TestCommanderCreate_Validation(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*CreateKeycloakUserParams)
		errContain string
	}{
		{"missing username", func(p *CreateKeycloakUserParams) { p.Username = "" }, "username"},
		{"missing email", func(p *CreateKeycloakUserParams) { p.Email = "" }, "email"},
		{"missing first name", func(p *CreateKeycloakUserParams) { p.FirstName = "" }, "first name"},
		{"missing last name", func(p *CreateKeycloakUserParams) { p.LastName = "" }, "last name"},
		{"missing password", func(p *CreateKeycloakUserParams) { p.Password = "" }, "password"},
		{"invalid role", func(p *CreateKeycloakUserParams) { p.Role = "invalid" }, "role"},
		{"participant without participantId", func(p *CreateKeycloakUserParams) {
			p.Role = auth.RoleParticipant
			p.ParticipantID = ""
		}, "participantId"},
		{"agent without agentId", func(p *CreateKeycloakUserParams) {
			p.Role = auth.RoleAgent
			p.AgentID = ""
		}, "agentId"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adminClient := NewMockKeycloakAdminClient(t)
			cmd := NewKeycloakUserCommander(adminClient, nil, nil)

			params := validCreateParams()
			tt.modify(&params)
			_, err := cmd.Create(context.Background(), params)

			require.Error(t, err)
			assert.ErrorAs(t, err, &InvalidInputError{})
			assert.Contains(t, err.Error(), tt.errContain)
		})
	}
}

func TestCommanderCreate_ParticipantValidation(t *testing.T) {
	tests := []struct {
		name          string
		participantID string
		existsReturn  *bool // nil = don't set up Exists mock (invalid UUID case)
		errContain    string
	}{
		{"invalid UUID", "not-a-uuid", nil, "invalid participant id"},
		{"not found", uuid.New().String(), helpers.BoolPtr(false), "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adminClient := NewMockKeycloakAdminClient(t)
			participantQuerier := NewMockParticipantQuerier(t)
			cmd := NewKeycloakUserCommander(adminClient, participantQuerier, nil)

			if tt.existsReturn != nil {
				id, _ := properties.ParseUUID(tt.participantID)
				participantQuerier.EXPECT().Exists(mock.Anything, id).Return(*tt.existsReturn, nil)
			}

			params := validCreateParams()
			params.Role = auth.RoleParticipant
			params.ParticipantID = tt.participantID
			_, err := cmd.Create(context.Background(), params)

			require.Error(t, err)
			assert.ErrorAs(t, err, &InvalidInputError{})
			assert.Contains(t, err.Error(), tt.errContain)
		})
	}
}

func TestCommanderCreate_AgentValidation(t *testing.T) {
	adminClient := NewMockKeycloakAdminClient(t)
	agentQuerier := NewMockAgentQuerier(t)
	cmd := NewKeycloakUserCommander(adminClient, nil, agentQuerier)

	agentID := uuid.New()
	agentQuerier.EXPECT().Exists(mock.Anything, properties.UUID(agentID)).Return(false, nil)

	params := validCreateParams()
	params.Role = auth.RoleAgent
	params.AgentID = agentID.String()
	_, err := cmd.Create(context.Background(), params)

	require.Error(t, err)
	assert.ErrorAs(t, err, &InvalidInputError{})
	assert.Contains(t, err.Error(), "not found")
}

// --- Commander Create: compensating deletes ---

func TestCommanderCreate_AdminClientFails(t *testing.T) {
	adminClient := NewMockKeycloakAdminClient(t)
	adminClient.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, errors.New("create error"))

	cmd := NewKeycloakUserCommander(adminClient, nil, nil)
	_, err := cmd.Create(context.Background(), validCreateParams())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "create error")
}

// --- Commander Create: happy path ---

func TestCommanderCreate_Success(t *testing.T) {
	adminClient := NewMockKeycloakAdminClient(t)
	cmd := NewKeycloakUserCommander(adminClient, nil, nil)

	adminClient.EXPECT().Create(mock.Anything, mock.Anything).Return(sampleKeycloakUser(), nil)

	result, err := cmd.Create(context.Background(), validCreateParams())

	require.NoError(t, err)
	assert.Equal(t, "user-123", result.ID)
	assert.Equal(t, "john", result.Username)
}

// --- Commander Update ---

func TestCommanderUpdate(t *testing.T) {
	newEmail := "new@example.com"
	newPass := "newpass123"

	tests := []struct {
		name        string
		id          string
		params      UpdateKeycloakUserParams
		setup       func(*MockKeycloakAdminClient, *MockParticipantQuerier, *MockAgentQuerier)
		wantErr     bool
		errContain  string
		checkResult func(*testing.T, *KeycloakUser)
	}{
		{
			name:   "empty ID",
			id:     "",
			params: UpdateKeycloakUserParams{},
			setup:  func(m *MockKeycloakAdminClient, pq *MockParticipantQuerier, aq *MockAgentQuerier) {},
			wantErr:    true,
			errContain: "id is required",
		},
		{
			name:   "success without password",
			id:     "user-123",
			params: UpdateKeycloakUserParams{Email: &newEmail},
			setup: func(m *MockKeycloakAdminClient, pq *MockParticipantQuerier, aq *MockAgentQuerier) {
				updated := sampleKeycloakUser()
				updated.Email = newEmail
				m.EXPECT().Update(mock.Anything, "user-123", mock.Anything).Return(updated, nil)
			},
			checkResult: func(t *testing.T, u *KeycloakUser) {
				assert.Equal(t, "new@example.com", u.Email)
			},
		},
		{
			name:   "success with password",
			id:     "user-123",
			params: UpdateKeycloakUserParams{Password: &newPass},
			setup: func(m *MockKeycloakAdminClient, pq *MockParticipantQuerier, aq *MockAgentQuerier) {
				m.EXPECT().Update(mock.Anything, "user-123", mock.Anything).Return(sampleKeycloakUser(), nil)
			},
			checkResult: func(t *testing.T, u *KeycloakUser) {
				assert.Equal(t, "user-123", u.ID)
			},
		},
		{
			name:   "UpdateUser fails",
			id:     "user-123",
			params: UpdateKeycloakUserParams{},
			setup: func(m *MockKeycloakAdminClient, pq *MockParticipantQuerier, aq *MockAgentQuerier) {
				m.EXPECT().Update(mock.Anything, "user-123", mock.Anything).Return(nil, errors.New("update failed"))
			},
			wantErr:    true,
			errContain: "update failed",
		},
		{
			name: "role change to participant with participantId",
			id:   "user-123",
			params: UpdateKeycloakUserParams{
				Role:          rolePtr(auth.RoleParticipant),
				ParticipantID: helpers.StringPtr(sampleParticipantUUID.String()),
			},
			setup: func(m *MockKeycloakAdminClient, pq *MockParticipantQuerier, aq *MockAgentQuerier) {
				pq.EXPECT().Exists(mock.Anything, properties.UUID(sampleParticipantUUID)).Return(true, nil)
				m.EXPECT().Update(mock.Anything, "user-123", mock.Anything).Return(&KeycloakUser{
					ID: "user-123", Roles: []string{"participant"}, ParticipantID: sampleParticipantUUID.String(),
				}, nil)
			},
			checkResult: func(t *testing.T, u *KeycloakUser) {
				assert.Equal(t, []string{"participant"}, u.Roles)
				assert.Equal(t, sampleParticipantUUID.String(), u.ParticipantID)
			},
		},
		{
			name: "role change to participant without participantId",
			id:   "user-123",
			params: UpdateKeycloakUserParams{
				Role: rolePtr(auth.RoleParticipant),
			},
			setup: func(m *MockKeycloakAdminClient, pq *MockParticipantQuerier, aq *MockAgentQuerier) {},
			wantErr:    true,
			errContain: "participantId is required",
		},
		{
			name: "role change to admin clears attributes",
			id:   "user-123",
			params: UpdateKeycloakUserParams{
				Role: rolePtr(auth.RoleAdmin),
			},
			setup: func(m *MockKeycloakAdminClient, pq *MockParticipantQuerier, aq *MockAgentQuerier) {
				m.EXPECT().Update(mock.Anything, "user-123", mock.MatchedBy(func(p UpdateKeycloakUserParams) bool {
					return p.ParticipantID != nil && *p.ParticipantID == "" &&
						p.AgentID != nil && *p.AgentID == ""
				})).Return(sampleKeycloakUser(), nil)
			},
			checkResult: func(t *testing.T, u *KeycloakUser) {
				assert.Equal(t, []string{"admin"}, u.Roles)
			},
		},
		{
			name: "attribute-only update: participantId on participant user",
			id:   "user-123",
			params: UpdateKeycloakUserParams{
				ParticipantID: helpers.StringPtr(sampleParticipantUUID.String()),
			},
			setup: func(m *MockKeycloakAdminClient, pq *MockParticipantQuerier, aq *MockAgentQuerier) {
				m.EXPECT().Get(mock.Anything, "user-123").Return(&KeycloakUser{
					ID: "user-123", Roles: []string{"participant"}, ParticipantID: "old-id",
				}, nil)
				pq.EXPECT().Exists(mock.Anything, properties.UUID(sampleParticipantUUID)).Return(true, nil)
				m.EXPECT().Update(mock.Anything, "user-123", mock.Anything).Return(&KeycloakUser{
					ID: "user-123", Roles: []string{"participant"}, ParticipantID: sampleParticipantUUID.String(),
				}, nil)
			},
			checkResult: func(t *testing.T, u *KeycloakUser) {
				assert.Equal(t, sampleParticipantUUID.String(), u.ParticipantID)
			},
		},
		{
			name: "attribute-only update: participantId on admin user rejected",
			id:   "user-123",
			params: UpdateKeycloakUserParams{
				ParticipantID: helpers.StringPtr(sampleParticipantUUID.String()),
			},
			setup: func(m *MockKeycloakAdminClient, pq *MockParticipantQuerier, aq *MockAgentQuerier) {
				m.EXPECT().Get(mock.Anything, "user-123").Return(sampleKeycloakUser(), nil)
			},
			wantErr:    true,
			errContain: "participantId can only be set on users with role participant",
		},
		{
			name: "invalid role",
			id:   "user-123",
			params: UpdateKeycloakUserParams{
				Role: rolePtr("invalid"),
			},
			setup: func(m *MockKeycloakAdminClient, pq *MockParticipantQuerier, aq *MockAgentQuerier) {},
			wantErr:    true,
			errContain: "invalid role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adminClient := NewMockKeycloakAdminClient(t)
			participantQuerier := NewMockParticipantQuerier(t)
			agentQuerier := NewMockAgentQuerier(t)
			tt.setup(adminClient, participantQuerier, agentQuerier)
			cmd := NewKeycloakUserCommander(adminClient, participantQuerier, agentQuerier)

			result, err := cmd.Update(context.Background(), tt.id, tt.params)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				require.NoError(t, err)
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

var sampleParticipantUUID = uuid.New()

func rolePtr(r auth.Role) *auth.Role { return &r }

// --- Commander Delete ---

func TestCommanderDelete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		setup      func(*MockKeycloakAdminClient)
		wantErr    bool
		errContain string
	}{
		{
			name:       "empty ID",
			id:         "",
			setup:      func(m *MockKeycloakAdminClient) {},
			wantErr:    true,
			errContain: "id is required",
		},
		{
			name: "success",
			id:   "user-123",
			setup: func(m *MockKeycloakAdminClient) {
				m.EXPECT().Delete(mock.Anything, "user-123").Return(nil)
			},
		},
		{
			name: "failure",
			id:   "user-123",
			setup: func(m *MockKeycloakAdminClient) {
				m.EXPECT().Delete(mock.Anything, "user-123").Return(errors.New("delete failed"))
			},
			wantErr:    true,
			errContain: "delete failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adminClient := NewMockKeycloakAdminClient(t)
			tt.setup(adminClient)
			cmd := NewKeycloakUserCommander(adminClient, nil, nil)

			err := cmd.Delete(context.Background(), tt.id)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
