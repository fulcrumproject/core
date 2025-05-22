package domain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuthRule_String(t *testing.T) {
	rule := AuthRule{
		Subject: SubjectAgent,
		Action:  ActionCreate,
		Roles:   []AuthRole{RoleFulcrumAdmin, RoleParticipant},
	}

	assert.Equal(t, "agent:create", rule.String())
}

func TestNewRuleAuthorizer(t *testing.T) {
	// Create some test rules
	rules := []AuthRule{
		{Subject: SubjectAgent, Action: ActionCreate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
		{Subject: SubjectService, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	}

	// Create a new authorizer with these rules
	authorizer := NewRuleAuthorizer(rules...)

	// Test internal rule structure was properly built
	assert.NotNil(t, authorizer.rules)
	assert.Len(t, authorizer.rules, 2)
	assert.Contains(t, authorizer.rules, SubjectAgent)
	assert.Contains(t, authorizer.rules, SubjectService)
	assert.Contains(t, authorizer.rules[SubjectAgent], ActionCreate)
	assert.Contains(t, authorizer.rules[SubjectService], ActionRead)
	assert.True(t, authorizer.rules[SubjectAgent][ActionCreate][RoleFulcrumAdmin])
	assert.True(t, authorizer.rules[SubjectAgent][ActionCreate][RoleParticipant])
	assert.True(t, authorizer.rules[SubjectService][ActionRead][RoleFulcrumAdmin])
	assert.True(t, authorizer.rules[SubjectService][ActionRead][RoleParticipant])
	assert.True(t, authorizer.rules[SubjectService][ActionRead][RoleParticipant])
}

func TestNewDefaultRuleAuthorizer(t *testing.T) {
	// Create default authorizer
	authorizer := NewDefaultRuleAuthorizer()

	// Spot check some rules
	assert.True(t, authorizer.hasPermission(SubjectParticipant, ActionRead, RoleFulcrumAdmin))
	assert.True(t, authorizer.hasPermission(SubjectAgent, ActionCreate, RoleParticipant))
	assert.True(t, authorizer.hasPermission(SubjectService, ActionRead, RoleParticipant))
	assert.False(t, authorizer.hasPermission(SubjectAgent, ActionCreate, RoleAgent))
}

func TestRuleAuthorizer_hasPermission(t *testing.T) {
	// Create a simple authorizer with a few rules
	authorizer := NewRuleAuthorizer(
		AuthRule{Subject: SubjectAgent, Action: ActionCreate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
		AuthRule{Subject: SubjectService, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	)

	tests := []struct {
		name     string
		subject  AuthSubject
		action   AuthAction
		role     AuthRole
		expected bool
	}{
		{
			name:     "Admin can create agent",
			subject:  SubjectAgent,
			action:   ActionCreate,
			role:     RoleFulcrumAdmin,
			expected: true,
		},
		{
			name:     "Participant can create agent",
			subject:  SubjectAgent,
			action:   ActionCreate,
			role:     RoleParticipant,
			expected: true,
		},
		{
			name:     "Agent cannot create agent",
			subject:  SubjectAgent,
			action:   ActionCreate,
			role:     RoleAgent,
			expected: false,
		},
		{
			name:     "Admin can read service",
			subject:  SubjectService,
			action:   ActionRead,
			role:     RoleFulcrumAdmin,
			expected: true,
		},
		{
			name:     "Participant can read service",
			subject:  SubjectService,
			action:   ActionRead,
			role:     RoleParticipant,
			expected: true,
		},
		{
			name:     "Non-existent subject",
			subject:  "nonexistent",
			action:   ActionRead,
			role:     RoleFulcrumAdmin,
			expected: false,
		},
		{
			name:     "Non-existent action",
			subject:  SubjectAgent,
			action:   "nonexistent",
			role:     RoleFulcrumAdmin,
			expected: false,
		},
		{
			name:     "Non-existent role",
			subject:  SubjectAgent,
			action:   ActionCreate,
			role:     "nonexistent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := authorizer.hasPermission(tt.subject, tt.action, tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRuleAuthorizer_Authorize(t *testing.T) {
	// Create a simple authorizer with a few rules
	authorizer := NewRuleAuthorizer(
		AuthRule{Subject: SubjectAgent, Action: ActionCreate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
		AuthRule{Subject: SubjectService, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	)

	participantID := uuid.New()
	agentID := uuid.New()
	consumerID := uuid.New()

	tests := []struct {
		name        string
		setupMockID func() AuthIdentity
		subject     AuthSubject
		action      AuthAction
		targetScope *AuthScope
		wantErr     bool
		errContains string
	}{
		{
			name: "Admin can create agent - no scope restriction",
			setupMockID: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)
			},
			subject:     SubjectAgent,
			action:      ActionCreate,
			targetScope: &AuthScope{},
			wantErr:     false,
		},
		{
			name: "Participant can create agent in own scope",
			setupMockID: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&participantID)
			},
			subject:     SubjectAgent,
			action:      ActionCreate,
			targetScope: &AuthScope{ProviderID: &participantID},
			wantErr:     false,
		},
		{
			name: "Participant cannot create agent in different scope",
			setupMockID: func() AuthIdentity {
				myProviderID := uuid.New() // Different participant ID
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&myProviderID)
			},
			subject:     SubjectAgent,
			action:      ActionCreate,
			targetScope: &AuthScope{ProviderID: &participantID},
			wantErr:     true,
			errContains: "invalid authorization scope",
		},
		{
			name: "Agent can't create agent (not in role list)",
			setupMockID: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleAgent).
					WithAgentID(&agentID)
			},
			subject:     SubjectAgent,
			action:      ActionCreate,
			targetScope: &AuthScope{},
			wantErr:     true,
			errContains: "does not have permission",
		},
		{
			name: "Participant can read service in own scope",
			setupMockID: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleParticipant).
					WithParticipantID(&consumerID)
			},
			subject:     SubjectService,
			action:      ActionRead,
			targetScope: &AuthScope{ConsumerID: &consumerID},
			wantErr:     false,
		},
		{
			name: "Nil identity",
			setupMockID: func() AuthIdentity {
				return nil
			},
			subject:     SubjectAgent,
			action:      ActionCreate,
			targetScope: &AuthScope{},
			wantErr:     true,
			errContains: "missing identity",
		},
		{
			name: "Nil target scope",
			setupMockID: func() AuthIdentity {
				return NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)
			},
			subject:     SubjectAgent,
			action:      ActionCreate,
			targetScope: nil,
			wantErr:     true,
			errContains: "nil authorization target scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := tt.setupMockID()
			err := authorizer.Authorize(identity, tt.subject, tt.action, tt.targetScope)

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

func TestRuleAuthorizer_AuthorizeCtx(t *testing.T) {
	baseCtx := context.Background()
	authorizer := NewRuleAuthorizer(
		AuthRule{Subject: SubjectAgent, Action: ActionCreate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
		AuthRule{Subject: SubjectService, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	)

	participantID := uuid.New()

	tests := []struct {
		name        string
		setupCtx    func() context.Context
		subject     AuthSubject
		action      AuthAction
		targetScope *AuthScope
		wantErr     bool
	}{
		{
			name: "Valid context with permission",
			setupCtx: func() context.Context {
				identity := NewMockAuthIdentity(uuid.New(), RoleFulcrumAdmin)
				return ContextWithMockAuth(baseCtx, identity)
			},
			subject:     SubjectAgent,
			action:      ActionCreate,
			targetScope: &AuthScope{},
			wantErr:     false,
		},
		{
			name: "Valid context but no permission",
			setupCtx: func() context.Context {
				identity := NewMockAuthIdentity(uuid.New(), RoleAgent)
				return ContextWithMockAuth(baseCtx, identity)
			},
			subject:     SubjectAgent,
			action:      ActionCreate,
			targetScope: &AuthScope{},
			wantErr:     true,
		},
		{
			name: "Provider admin with matching scope",
			setupCtx: func() context.Context {
				identity := NewMockAuthIdentity(uuid.New(), RoleParticipant).WithParticipantID(&participantID)
				return ContextWithMockAuth(baseCtx, identity)
			},
			subject:     SubjectAgent,
			action:      ActionCreate,
			targetScope: &AuthScope{ParticipantID: &participantID},
			wantErr:     false,
		},
		{
			name: "Missing auth identity in context",
			setupCtx: func() context.Context {
				return baseCtx // No identity
			},
			subject:     SubjectAgent,
			action:      ActionCreate,
			targetScope: &AuthScope{},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()

			// For the missing identity test, we need to handle the panic
			if tt.name == "Missing auth identity in context" {
				assert.Panics(t, func() {
					authorizer.AuthorizeCtx(ctx, tt.subject, tt.action, tt.targetScope)
				})
				return
			}

			err := authorizer.AuthorizeCtx(ctx, tt.subject, tt.action, tt.targetScope)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
