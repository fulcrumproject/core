package domain

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuditEntry_Validate(t *testing.T) {
	entry := AuditEntry{
		AuthorityType: AuthorityTypeAdmin,
		AuthorityID:   uuid.New().String(),
		EventType:     EventTypeAgentCreated,
		Properties:    JSON{"key": "value"},
	}

	// Since Validate() always returns nil, this just confirms it doesn't panic
	err := entry.Validate()
	assert.NoError(t, err)
}

func TestNewEventAuditCtx(t *testing.T) {
	entityID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	consumerID := uuid.New()
	properties := JSON{"key": "value"}

	// Setup context with mock auth identity
	baseCtx := context.Background()
	identityID := uuid.New()
	identity := NewMockAuthIdentity(identityID, RoleFulcrumAdmin)
	ctx := ContextWithMockAuth(baseCtx, identity)

	entry, err := NewEventAuditCtx(
		ctx,
		EventTypeAgentCreated,
		properties,
		&entityID,
		&providerID,
		&agentID,
		&consumerID,
	)

	assert.NoError(t, err)
	assert.Equal(t, AuthorityTypeAdmin, entry.AuthorityType)
	assert.Equal(t, identityID.String(), entry.AuthorityID)
	assert.Equal(t, EventTypeAgentCreated, entry.EventType)
	assert.Equal(t, properties, entry.Properties)
	assert.Equal(t, entityID, *entry.EntityID)
	assert.Equal(t, providerID, *entry.ProviderID)
	assert.Equal(t, agentID, *entry.AgentID)
	assert.Equal(t, consumerID, *entry.ConsumerID)
}

func TestNewEventAuditCtxDiff(t *testing.T) {
	type testEntity struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	beforeEntity := testEntity{
		Name:  "before",
		Value: 10,
	}

	afterEntity := testEntity{
		Name:  "after",
		Value: 20,
	}

	entityID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	consumerID := uuid.New()
	properties := JSON{"key": "value"}

	// Setup context with mock auth identity
	baseCtx := context.Background()
	identityID := uuid.New()
	identity := NewMockAuthIdentity(identityID, RoleParticipant)
	ctx := ContextWithMockAuth(baseCtx, identity)

	entry, err := NewEventAuditCtxDiff(
		ctx,
		EventTypeAgentUpdated,
		properties,
		&entityID,
		&providerID,
		&agentID,
		&consumerID,
		beforeEntity,
		afterEntity,
	)

	assert.NoError(t, err)
	assert.Equal(t, AuthorityTypeParticipant, entry.AuthorityType)
	assert.Equal(t, identityID.String(), entry.AuthorityID)
	assert.Equal(t, EventTypeAgentUpdated, entry.EventType)
	assert.Equal(t, properties, entry.Properties)
	assert.Equal(t, entityID, *entry.EntityID)
	assert.Equal(t, providerID, *entry.ProviderID)
	assert.Equal(t, agentID, *entry.AgentID)
	assert.Equal(t, consumerID, *entry.ConsumerID)

	// Check that diff was generated
	assert.Contains(t, entry.Properties, "diff")
	assert.NotNil(t, entry.Properties["diff"])
}

func TestNewEventAuditCtxDiff_ErrorHandling(t *testing.T) {
	type testEntity struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	beforeEntity := testEntity{
		Name:  "before",
		Value: 10,
	}

	afterEntity := testEntity{
		Name:  "after",
		Value: 20,
	}

	// Unmarshalable entity that will cause json.Marshal to fail
	type badEntity struct {
		BadField func() `json:"bad_field"`
	}

	badEntityInstance := badEntity{
		BadField: func() {},
	}

	// Setup context with mock auth identity
	baseCtx := context.Background()
	identityID := uuid.New()
	identity := NewMockAuthIdentity(identityID, RoleFulcrumAdmin)
	ctx := ContextWithMockAuth(baseCtx, identity)

	entityID := uuid.New()
	properties := JSON{"key": "value"}

	tests := []struct {
		name       string
		before     interface{}
		after      interface{}
		wantErr    bool
		errorCheck func(t *testing.T, err error)
	}{
		{
			name:    "Success",
			before:  beforeEntity,
			after:   afterEntity,
			wantErr: false,
		},
		{
			name:    "Marshal before entity error",
			before:  badEntityInstance,
			after:   afterEntity,
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "failed to marshal 'before' entity")
			},
		},
		{
			name:    "Marshal after entity error",
			before:  beforeEntity,
			after:   badEntityInstance,
			wantErr: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "failed to marshal 'after' entity")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := NewEventAuditCtxDiff(
				ctx,
				EventTypeAgentUpdated,
				properties,
				&entityID,
				nil,
				nil,
				nil,
				tt.before,
				tt.after,
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, entry)
				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, entry)
				assert.NotNil(t, entry.Properties)
				assert.Contains(t, entry.Properties, "diff")
			}
		})
	}
}

func TestAuditEntry_TableName(t *testing.T) {
	auditEntry := AuditEntry{}
	assert.Equal(t, "audit_entries", auditEntry.TableName())
}

func TestExtractAuditAuthority(t *testing.T) {
	baseCtx := context.Background()
	identityID := uuid.New()

	tests := []struct {
		name            string
		setupContext    func() context.Context
		wantAuthority   AuthorityType
		wantAuthorityID string
		wantPanic       bool
	}{
		{
			name: "Admin role",
			setupContext: func() context.Context {
				identity := NewMockAuthIdentity(identityID, RoleFulcrumAdmin)
				return ContextWithMockAuth(baseCtx, identity)
			},
			wantAuthority:   AuthorityTypeAdmin,
			wantAuthorityID: identityID.String(),
		},
		{
			name: "Provider role",
			setupContext: func() context.Context {
				identity := NewMockAuthIdentity(identityID, RoleParticipant)
				return ContextWithMockAuth(baseCtx, identity)
			},
			wantAuthority:   AuthorityTypeParticipant,
			wantAuthorityID: identityID.String(),
		},
		{
			name: "Agent role",
			setupContext: func() context.Context {
				identity := NewMockAuthIdentity(identityID, RoleAgent)
				return ContextWithMockAuth(baseCtx, identity)
			},
			wantAuthority:   AuthorityTypeAgent,
			wantAuthorityID: identityID.String(),
		},
		{
			name: "Consumer role",
			setupContext: func() context.Context {
				identity := NewMockAuthIdentity(identityID, RoleParticipant)
				return ContextWithMockAuth(baseCtx, identity)
			},
			wantAuthority:   AuthorityTypeParticipant,
			wantAuthorityID: identityID.String(),
		},
		{
			name: "Unknown role defaults to internal",
			setupContext: func() context.Context {
				identity := NewMockAuthIdentity(identityID, "unknown-role")
				return ContextWithMockAuth(baseCtx, identity)
			},
			wantAuthority:   AuthorityTypeInternal,
			wantAuthorityID: identityID.String(),
		},
		{
			name: "Missing auth identity",
			setupContext: func() context.Context {
				// No auth identity added to context
				return baseCtx
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()

			if tt.wantPanic {
				assert.Panics(t, func() {
					extractAuditAuthority(ctx)
				})
				return
			}

			authorityType, authorityID := extractAuditAuthority(ctx)
			assert.Equal(t, tt.wantAuthority, authorityType)
			assert.Equal(t, tt.wantAuthorityID, authorityID)
		})
	}
}
