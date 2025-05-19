package domain

import (
	"context"
	"errors"
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

func TestNewEventAudit(t *testing.T) {
	entityID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	brokerID := uuid.New()
	properties := JSON{"key": "value"}

	entry := NewEventAudit(
		AuthorityTypeAdmin,
		"admin-1",
		EventTypeAgentCreated,
		properties,
		&entityID,
		&providerID,
		&agentID,
		&brokerID,
	)

	assert.Equal(t, AuthorityTypeAdmin, entry.AuthorityType)
	assert.Equal(t, "admin-1", entry.AuthorityID)
	assert.Equal(t, EventTypeAgentCreated, entry.EventType)
	assert.Equal(t, properties, entry.Properties)
	assert.Equal(t, entityID, *entry.EntityID)
	assert.Equal(t, providerID, *entry.ProviderID)
	assert.Equal(t, agentID, *entry.AgentID)
	assert.Equal(t, brokerID, *entry.ConsumerID)
}

func TestAuditEntry_GenerateDiff(t *testing.T) {
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
			audit := &AuditEntry{}
			err := audit.GenerateDiff(tt.before, tt.after)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, audit.Properties)
				assert.Contains(t, audit.Properties, "diff")
			}
		})
	}
}

func TestAuditEntry_TableName(t *testing.T) {
	auditEntry := AuditEntry{}
	assert.Equal(t, "audit_entries", auditEntry.TableName())
}

func TestAuditEntryCommander_Create(t *testing.T) {
	ctx := context.Background()
	entityID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()
	brokerID := uuid.New()
	properties := JSON{"key": "value"}

	tests := []struct {
		name          string
		setupMocks    func(store *MockStore)
		authorityType AuthorityType
		authorityID   string
		eventType     EventType
		wantErr       bool
		errorCheck    func(t *testing.T, err error)
	}{
		{
			name: "Create success",
			setupMocks: func(store *MockStore) {
				auditRepo := &MockAuditEntryRepository{}
				auditRepo.createFunc = func(ctx context.Context, entry *AuditEntry) error {
					// Verify entry fields
					assert.Equal(t, AuthorityTypeAdmin, entry.AuthorityType)
					assert.Equal(t, "admin-1", entry.AuthorityID)
					assert.Equal(t, EventTypeAgentCreated, entry.EventType)
					assert.Equal(t, properties, entry.Properties)
					assert.Equal(t, entityID, *entry.EntityID)
					assert.Equal(t, providerID, *entry.ProviderID)
					assert.Equal(t, agentID, *entry.AgentID)
					assert.Equal(t, brokerID, *entry.ConsumerID)
					return nil
				}
				store.WithAuditEntryRepo(auditRepo)
			},
			authorityType: AuthorityTypeAdmin,
			authorityID:   "admin-1",
			eventType:     EventTypeAgentCreated,
			wantErr:       false,
		},
		{
			name: "Repository error",
			setupMocks: func(store *MockStore) {
				auditRepo := &MockAuditEntryRepository{}
				auditRepo.createFunc = func(ctx context.Context, entry *AuditEntry) error {
					return errors.New("repository error")
				}
				store.WithAuditEntryRepo(auditRepo)
			},
			authorityType: AuthorityTypeAdmin,
			authorityID:   "admin-1",
			eventType:     EventTypeAgentCreated,
			wantErr:       true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "repository error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			tt.setupMocks(store)

			commander := NewAuditEntryCommander(store)
			entry, err := commander.Create(
				ctx,
				tt.authorityType,
				tt.authorityID,
				tt.eventType,
				properties,
				&entityID,
				&providerID,
				&agentID,
				&brokerID,
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
				assert.Equal(t, tt.authorityType, entry.AuthorityType)
				assert.Equal(t, tt.authorityID, entry.AuthorityID)
				assert.Equal(t, tt.eventType, entry.EventType)
				assert.Equal(t, properties, entry.Properties)
				assert.Equal(t, entityID, *entry.EntityID)
				assert.Equal(t, providerID, *entry.ProviderID)
				assert.Equal(t, agentID, *entry.AgentID)
				assert.Equal(t, brokerID, *entry.ConsumerID)
			}
		})
	}
}

func TestAuditEntryCommander_CreateWithDiff(t *testing.T) {
	ctx := context.Background()
	entityID := uuid.New()
	providerID := uuid.New()

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

	tests := []struct {
		name          string
		setupMocks    func(store *MockStore)
		authorityType AuthorityType
		authorityID   string
		eventType     EventType
		before        interface{}
		after         interface{}
		wantErr       bool
		errorCheck    func(t *testing.T, err error)
	}{
		{
			name: "CreateWithDiff success",
			setupMocks: func(store *MockStore) {
				auditRepo := &MockAuditEntryRepository{}
				auditRepo.createFunc = func(ctx context.Context, entry *AuditEntry) error {
					// Verify the diff was created
					assert.NotNil(t, entry.Properties)
					assert.Contains(t, entry.Properties, "diff")

					return nil
				}
				store.WithAuditEntryRepo(auditRepo)
			},
			authorityType: AuthorityTypeAdmin,
			authorityID:   "admin-1",
			eventType:     EventTypeAgentUpdated,
			before:        beforeEntity,
			after:         afterEntity,
			wantErr:       false,
		},
		{
			name: "Marshal before entity error",
			setupMocks: func(store *MockStore) {
				// No need to set up audit repo as it shouldn't be called
			},
			authorityType: AuthorityTypeAdmin,
			authorityID:   "admin-1",
			eventType:     EventTypeAgentUpdated,
			before:        badEntityInstance,
			after:         afterEntity,
			wantErr:       true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "failed to marshal 'before' entity")
			},
		},
		{
			name: "Marshal after entity error",
			setupMocks: func(store *MockStore) {
				// No need to set up audit repo as it shouldn't be called
			},
			authorityType: AuthorityTypeAdmin,
			authorityID:   "admin-1",
			eventType:     EventTypeAgentUpdated,
			before:        beforeEntity,
			after:         badEntityInstance,
			wantErr:       true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "failed to marshal 'after' entity")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			tt.setupMocks(store)

			commander := NewAuditEntryCommander(store)
			entry, err := commander.CreateWithDiff(
				ctx,
				tt.authorityType,
				tt.authorityID,
				tt.eventType,
				&entityID,
				&providerID,
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
				assert.Equal(t, tt.authorityType, entry.AuthorityType)
				assert.Equal(t, tt.authorityID, entry.AuthorityID)
				assert.Equal(t, tt.eventType, entry.EventType)
				assert.Contains(t, entry.Properties, "diff")
			}
		})
	}
}

func TestAuditEntryCommander_CreateCtx(t *testing.T) {
	baseCtx := context.Background()
	identityID := uuid.New()
	entityID := uuid.New()
	providerID := uuid.New()

	properties := JSON{"key": "value"}

	tests := []struct {
		name          string
		setupContext  func() context.Context
		setupMocks    func(store *MockStore)
		eventType     EventType
		wantErr       bool
		wantAuthority AuthorityType
		errorCheck    func(t *testing.T, err error)
	}{
		{
			name: "Admin role",
			setupContext: func() context.Context {
				identity := NewMockAuthIdentity(identityID, RoleFulcrumAdmin)
				return ContextWithMockAuth(baseCtx, identity)
			},
			setupMocks: func(store *MockStore) {
				auditRepo := &MockAuditEntryRepository{}
				auditRepo.createFunc = func(ctx context.Context, entry *AuditEntry) error {
					assert.Equal(t, AuthorityTypeAdmin, entry.AuthorityType)
					assert.Equal(t, identityID.String(), entry.AuthorityID)
					return nil
				}
				store.WithAuditEntryRepo(auditRepo)
			},
			eventType:     EventTypeParticipantCreated,
			wantErr:       false,
			wantAuthority: AuthorityTypeAdmin,
		},
		{
			name: "Provider role",
			setupContext: func() context.Context {
				identity := NewMockAuthIdentity(identityID, RoleParticipant)
				return ContextWithMockAuth(baseCtx, identity)
			},
			setupMocks: func(store *MockStore) {
				auditRepo := &MockAuditEntryRepository{}
				auditRepo.createFunc = func(ctx context.Context, entry *AuditEntry) error {
					assert.Equal(t, AuthorityTypeParticipant, entry.AuthorityType)
					assert.Equal(t, identityID.String(), entry.AuthorityID)
					return nil
				}
				store.WithAuditEntryRepo(auditRepo)
			},
			eventType:     EventTypeAgentCreated,
			wantErr:       false,
			wantAuthority: AuthorityTypeParticipant,
		},
		{
			name: "Agent role",
			setupContext: func() context.Context {
				identity := NewMockAuthIdentity(identityID, RoleAgent)
				return ContextWithMockAuth(baseCtx, identity)
			},
			setupMocks: func(store *MockStore) {
				auditRepo := &MockAuditEntryRepository{}
				auditRepo.createFunc = func(ctx context.Context, entry *AuditEntry) error {
					assert.Equal(t, AuthorityTypeAgent, entry.AuthorityType)
					assert.Equal(t, identityID.String(), entry.AuthorityID)
					return nil
				}
				store.WithAuditEntryRepo(auditRepo)
			},
			eventType:     EventTypeServiceUpdated,
			wantErr:       false,
			wantAuthority: AuthorityTypeAgent,
		},
		{
			name: "Broker role",
			setupContext: func() context.Context {
				identity := NewMockAuthIdentity(identityID, RoleParticipant)
				return ContextWithMockAuth(baseCtx, identity)
			},
			setupMocks: func(store *MockStore) {
				auditRepo := &MockAuditEntryRepository{}
				auditRepo.createFunc = func(ctx context.Context, entry *AuditEntry) error {
					assert.Equal(t, AuthorityTypeParticipant, entry.AuthorityType)
					assert.Equal(t, identityID.String(), entry.AuthorityID)
					return nil
				}
				store.WithAuditEntryRepo(auditRepo)
			},
			eventType:     EventTypeServiceUpdated,
			wantErr:       false,
			wantAuthority: AuthorityTypeParticipant,
		},
		{
			name: "Unknown role defaults to internal",
			setupContext: func() context.Context {
				// Using a string directly here to simulate an unknown role
				identity := NewMockAuthIdentity(identityID, "unknown-role")
				return ContextWithMockAuth(baseCtx, identity)
			},
			setupMocks: func(store *MockStore) {
				auditRepo := &MockAuditEntryRepository{}
				auditRepo.createFunc = func(ctx context.Context, entry *AuditEntry) error {
					assert.Equal(t, AuthorityTypeInternal, entry.AuthorityType)
					assert.Equal(t, identityID.String(), entry.AuthorityID)
					return nil
				}
				store.WithAuditEntryRepo(auditRepo)
			},
			eventType:     EventTypeServiceUpdated,
			wantErr:       false,
			wantAuthority: AuthorityTypeInternal,
		},
		{
			name: "Missing auth identity",
			setupContext: func() context.Context {
				// No auth identity added to context
				return baseCtx
			},
			setupMocks: func(store *MockStore) {
				// No need to set up audit repo as it shouldn't be called
			},
			eventType: EventTypeServiceUpdated,
			wantErr:   true,
			errorCheck: func(t *testing.T, err error) {
				// This checks for panic
				assert.Contains(t, err.Error(), "auth identity")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			tt.setupMocks(store)
			ctx := tt.setupContext()

			commander := NewAuditEntryCommander(store)

			// Handle panic from MustGetAuthIdentity if no identity in context
			if tt.name == "Missing auth identity" {
				assert.Panics(t, func() {
					commander.CreateCtx(
						ctx,
						tt.eventType,
						properties,
						&entityID,
						&providerID,
						nil,
						nil,
					)
				})
				return
			}

			entry, err := commander.CreateCtx(
				ctx,
				tt.eventType,
				properties,
				&entityID,
				&providerID,
				nil,
				nil,
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
				assert.Equal(t, tt.wantAuthority, entry.AuthorityType)
				assert.Equal(t, identityID.String(), entry.AuthorityID)
				assert.Equal(t, tt.eventType, entry.EventType)
				assert.Equal(t, properties, entry.Properties)
			}
		})
	}
}

func TestAuditEntryCommander_CreateCtxWithDiff(t *testing.T) {
	baseCtx := context.Background()
	identityID := uuid.New()
	entityID := uuid.New()
	providerID := uuid.New()

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

	tests := []struct {
		name          string
		setupContext  func() context.Context
		setupMocks    func(store *MockStore)
		eventType     EventType
		wantErr       bool
		wantAuthority AuthorityType
		errorCheck    func(t *testing.T, err error)
	}{
		{
			name: "Admin role with diff",
			setupContext: func() context.Context {
				identity := NewMockAuthIdentity(identityID, RoleFulcrumAdmin)
				return ContextWithMockAuth(baseCtx, identity)
			},
			setupMocks: func(store *MockStore) {
				auditRepo := &MockAuditEntryRepository{}
				auditRepo.createFunc = func(ctx context.Context, entry *AuditEntry) error {
					assert.Equal(t, AuthorityTypeAdmin, entry.AuthorityType)
					assert.Equal(t, identityID.String(), entry.AuthorityID)

					// Verify the diff was created
					assert.NotNil(t, entry.Properties)
					assert.Contains(t, entry.Properties, "diff")

					return nil
				}
				store.WithAuditEntryRepo(auditRepo)
			},
			eventType:     EventTypeParticipantUpdated,
			wantErr:       false,
			wantAuthority: AuthorityTypeAdmin,
		},
		{
			name: "Missing auth identity",
			setupContext: func() context.Context {
				// No auth identity added to context
				return baseCtx
			},
			setupMocks: func(store *MockStore) {
				// No need to set up audit repo as it shouldn't be called
			},
			eventType: EventTypeParticipantUpdated,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMockStore()
			tt.setupMocks(store)
			ctx := tt.setupContext()

			commander := NewAuditEntryCommander(store)

			// Handle panic from MustGetAuthIdentity if no identity in context
			if tt.name == "Missing auth identity" {
				assert.Panics(t, func() {
					commander.CreateCtxWithDiff(
						ctx,
						tt.eventType,
						&entityID,
						&providerID,
						nil,
						nil,
						beforeEntity,
						afterEntity,
					)
				})
				return
			}

			entry, err := commander.CreateCtxWithDiff(
				ctx,
				tt.eventType,
				&entityID,
				&providerID,
				nil,
				nil,
				beforeEntity,
				afterEntity,
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
				assert.Equal(t, tt.wantAuthority, entry.AuthorityType)
				assert.Equal(t, identityID.String(), entry.AuthorityID)
				assert.Equal(t, tt.eventType, entry.EventType)
				assert.Contains(t, entry.Properties, "diff")
			}
		})
	}
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
			name: "Broker role",
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
					ExtractAuditAuthority(ctx)
				})
				return
			}

			authorityType, authorityID := ExtractAuditAuthority(ctx)
			assert.Equal(t, tt.wantAuthority, authorityType)
			assert.Equal(t, tt.wantAuthorityID, authorityID)
		})
	}
}
