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
			name:    "Valid ProviderAdmin role",
			role:    RoleProviderAdmin,
			wantErr: false,
		},
		{
			name:    "Valid Broker role",
			role:    RoleBroker,
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
			name:    "Valid Provider subject",
			subject: SubjectProvider,
			wantErr: false,
		},
		{
			name:    "Valid Broker subject",
			subject: SubjectBroker,
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
	providerID := uuid.New()
	agentID := uuid.New()
	brokerID := uuid.New()
	differentProviderID := uuid.New()
	differentAgentID := uuid.New()
	differentBrokerID := uuid.New()

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
				ProviderID: &providerID,
				AgentID:    &agentID,
				BrokerID:   &brokerID,
			},
			wantErr: false,
		},
		{
			name: "ProviderAdmin with matching provider",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleProviderAdmin).
					WithProviderID(&providerID)
			},
			targetScope: &AuthScope{
				ProviderID: &providerID,
			},
			wantErr: false,
		},
		{
			name: "ProviderAdmin with non-matching provider",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleProviderAdmin).
					WithProviderID(&providerID)
			},
			targetScope: &AuthScope{
				ProviderID: &differentProviderID,
			},
			wantErr:     true,
			errContains: "invalid provider authorization scope",
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
			name: "Broker with matching broker",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleBroker).
					WithBrokerID(&brokerID)
			},
			targetScope: &AuthScope{
				BrokerID: &brokerID,
			},
			wantErr: false,
		},
		{
			name: "Broker with non-matching broker",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleBroker).
					WithBrokerID(&brokerID)
			},
			targetScope: &AuthScope{
				BrokerID: &differentBrokerID,
			},
			wantErr:     true,
			errContains: "invalid broker authorization scope",
		},
		{
			name: "ProviderAdmin with no scope provider ID is valid",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleProviderAdmin).
					WithProviderID(&providerID)
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
		{
			name: "Broker with no scope broker ID is valid",
			setupIdentity: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleBroker).
					WithBrokerID(&brokerID)
			},
			targetScope: &AuthScope{},
			wantErr:     false,
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
