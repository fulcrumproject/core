package domain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuthRole_Validate(t *testing.T) {
	tests := []struct {
		name    string
		role    AuthRole
		wantErr bool
	}{
		{
			name:    "Valid FulcrumAdmin role",
			role:    RoleFulcrumAdmin,
			wantErr: false,
		},
		{
			name:    "Valid Participant role",
			role:    RoleParticipant,
			wantErr: false,
		},
		{
			name:    "Valid Agent role",
			role:    RoleAgent,
			wantErr: false,
		},
		{
			name:    "Invalid role",
			role:    "invalid_role",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.role.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid auth role")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthSubject_Validate(t *testing.T) {
	tests := []struct {
		name    string
		subject AuthSubject
		wantErr bool
	}{
		{
			name:    "Valid Participant subject",
			subject: SubjectParticipant,
			wantErr: false,
		},
		{
			name:    "Valid Agent subject",
			subject: SubjectAgent,
			wantErr: false,
		},
		{
			name:    "Valid AgentType subject",
			subject: SubjectAgentType,
			wantErr: false,
		},
		{
			name:    "Valid Service subject",
			subject: SubjectService,
			wantErr: false,
		},
		{
			name:    "Valid ServiceType subject",
			subject: SubjectServiceType,
			wantErr: false,
		},
		{
			name:    "Valid ServiceGroup subject",
			subject: SubjectServiceGroup,
			wantErr: false,
		},
		{
			name:    "Valid Job subject",
			subject: SubjectJob,
			wantErr: false,
		},
		{
			name:    "Valid MetricType subject",
			subject: SubjectMetricType,
			wantErr: false,
		},
		{
			name:    "Valid MetricEntry subject",
			subject: SubjectMetricEntry,
			wantErr: false,
		},
		{
			name:    "Valid AuditEntry subject",
			subject: SubjectAuditEntry,
			wantErr: false,
		},
		{
			name:    "Valid Token subject",
			subject: SubjectToken,
			wantErr: false,
		},
		{
			name:    "Invalid subject",
			subject: "invalid_subject",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.subject.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid auth subject")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthAction_Validate(t *testing.T) {
	tests := []struct {
		name    string
		action  AuthAction
		wantErr bool
	}{
		{
			name:    "Valid Create action",
			action:  ActionCreate,
			wantErr: false,
		},
		{
			name:    "Valid Read action",
			action:  ActionRead,
			wantErr: false,
		},
		{
			name:    "Valid Update action",
			action:  ActionUpdate,
			wantErr: false,
		},
		{
			name:    "Valid Delete action",
			action:  ActionDelete,
			wantErr: false,
		},
		{
			name:    "Valid UpdateState action",
			action:  ActionUpdateState,
			wantErr: false,
		},
		{
			name:    "Valid GenerateToken action",
			action:  ActionGenerateToken,
			wantErr: false,
		},
		{
			name:    "Valid Start action",
			action:  ActionStart,
			wantErr: false,
		},
		{
			name:    "Valid Stop action",
			action:  ActionStop,
			wantErr: false,
		},
		{
			name:    "Valid Claim action",
			action:  ActionClaim,
			wantErr: false,
		},
		{
			name:    "Valid Complete action",
			action:  ActionComplete,
			wantErr: false,
		},
		{
			name:    "Valid Fail action",
			action:  ActionFail,
			wantErr: false,
		},
		{
			name:    "Valid ListPending action",
			action:  ActionListPending,
			wantErr: false,
		},
		{
			name:    "Invalid action",
			action:  "invalid_action",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid auth action")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthContextFunctions(t *testing.T) {
	// Setup
	baseCtx := context.Background()
	identityID := uuid.New()
	identity := NewMockAuthIdentity(identityID, RoleFulcrumAdmin)

	// Test WithAuthIdentity
	ctxWithIdentity := WithAuthIdentity(baseCtx, identity)
	assert.NotEqual(t, baseCtx, ctxWithIdentity, "Context should be different after adding identity")

	// Test MustGetAuthIdentity success
	retrievedIdentity := MustGetAuthIdentity(ctxWithIdentity)
	assert.Equal(t, identity.ID(), retrievedIdentity.ID())
	assert.Equal(t, identity.Role(), retrievedIdentity.Role())

	// Test MustGetAuthIdentity panic
	assert.Panics(t, func() {
		MustGetAuthIdentity(baseCtx)
	}, "MustGetAuthIdentity should panic when no identity is in context")
}

func TestValidateAuthScope(t *testing.T) {
	participantID := uuid.New()
	agentID := uuid.New()
	differentAgentID := uuid.New()
	differentParticipantID := uuid.New()

	tests := []struct {
		name          string
		setupIdentity func() AuthIdentity
		targetScope   *AuthScope
		wantErr       bool
		errContains   string
	}{
		{
			name: "Nil identity",
			setupIdentity: func() AuthIdentity {
				return nil
			},
			targetScope: &AuthScope{},
			wantErr:     true,
			errContains: "nil identity",
		},
		{
			name: "Nil target scope",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)
			},
			targetScope: nil,
			wantErr:     true,
			errContains: "nil authorization target scope",
		},
		{
			name: "FulcrumAdmin has access to any scope",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)
			},
			targetScope: &AuthScope{
				ParticipantID: &participantID,
				AgentID:       &agentID,
				ProviderID:    &participantID,
				ConsumerID:    &participantID,
			},
			wantErr: false,
		},
		{
			name: "Participant with matching participant",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&participantID)
			},
			targetScope: &AuthScope{
				ParticipantID: &participantID,
			},
			wantErr: false,
		},
		{
			name: "Participant with non-matching participant",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&participantID)
			},
			targetScope: &AuthScope{
				ParticipantID: &differentParticipantID,
			},
			wantErr:     true,
			errContains: "invalid participant authorization scope",
		},
		{
			name: "Agent with matching agent",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleAgent).
					WithAgentID(&agentID)
			},
			targetScope: &AuthScope{
				AgentID: &agentID,
			},
			wantErr: false,
		},
		{
			name: "Agent with non-matching agent",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleAgent).
					WithAgentID(&agentID)
			},
			targetScope: &AuthScope{
				AgentID: &differentAgentID,
			},
			wantErr:     true,
			errContains: "invalid agent authorization scope",
		},
		{
			name: "Participant with no scope participant ID is valid",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&participantID)
			},
			targetScope: &AuthScope{},
			wantErr:     false,
		},
		{
			name: "Agent with no scope agent ID is valid",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleAgent).
					WithAgentID(&agentID)
			},
			targetScope: &AuthScope{},
			wantErr:     false,
		},
		// New test cases for Provider and Consumer validation
		{
			name: "Participant with matching provider ID",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&participantID)
			},
			targetScope: &AuthScope{
				ProviderID: &participantID,
			},
			wantErr: false,
		},
		{
			name: "Participant with non-matching provider ID",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&participantID)
			},
			targetScope: &AuthScope{
				ProviderID: &differentParticipantID,
			},
			wantErr:     true,
			errContains: "invalid participant authorization scope",
		},
		{
			name: "Participant with matching consumer ID",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&participantID)
			},
			targetScope: &AuthScope{
				ConsumerID: &participantID,
			},
			wantErr: false,
		},
		{
			name: "Participant with non-matching consumer ID",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&participantID)
			},
			targetScope: &AuthScope{
				ConsumerID: &differentParticipantID,
			},
			wantErr:     true,
			errContains: "invalid participant authorization scope",
		},
		{
			name: "Participant with multiple matching scope fields",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&participantID)
			},
			targetScope: &AuthScope{
				ParticipantID: &participantID,
				ProviderID:    &participantID,
				ConsumerID:    &participantID,
			},
			wantErr: false,
		},
		{
			name: "Participant with multiple scope fields and one non-matching",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&participantID)
			},
			targetScope: &AuthScope{
				ParticipantID: &participantID,
				ProviderID:    &participantID,
				ConsumerID:    &differentParticipantID,
			},
			wantErr:     true,
			errContains: "invalid participant authorization scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := tt.setupIdentity()
			err := ValidateAuthScope(identity, tt.targetScope)

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
