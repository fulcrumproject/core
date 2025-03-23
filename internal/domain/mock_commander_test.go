package domain

import (
	"context"
)

// Ensure interface compliance
var _ AuditEntryCommander = (*MockAuditEntryCommander)(nil)

// MockAuditEntryCommander implements AuditEntryCommander for testing
type MockAuditEntryCommander struct {
	CreateFunc            func(ctx context.Context, authorityType AuthorityType, authorityID string, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error)
	CreateCtxFunc         func(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error)
	CreateCtxWithDiffFunc func(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error)
	CreateWithDiffFunc    func(ctx context.Context, authorityType AuthorityType, authorityID string, eventType EventType, entityID, providerID, agentID, brokerID *UUID, before, after interface{}) (*AuditEntry, error)
}

// Create creates a new audit entry
func (m *MockAuditEntryCommander) Create(ctx context.Context, authorityType AuthorityType, authorityID string, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, authorityType, authorityID, eventType, properties, entityID, providerID, agentID, brokerID)
	}
	return &AuditEntry{}, nil
}

// CreateWithDiff creates a new audit entry with diff
func (m *MockAuditEntryCommander) CreateWithDiff(ctx context.Context, authorityType AuthorityType, authorityID string, eventType EventType, entityID, providerID, agentID, brokerID *UUID, beforeEntity, afterEntity interface{}) (*AuditEntry, error) {
	if m.CreateWithDiffFunc != nil {
		return m.CreateWithDiffFunc(ctx, authorityType, authorityID, eventType, entityID, providerID, agentID, brokerID, beforeEntity, afterEntity)
	}
	return &AuditEntry{}, nil
}

// CreateCtx creates a new audit entry from context
func (m *MockAuditEntryCommander) CreateCtx(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, brokerID *UUID) (*AuditEntry, error) {
	if m.CreateCtxFunc != nil {
		return m.CreateCtxFunc(ctx, eventType, properties, entityID, providerID, agentID, brokerID)
	}
	return &AuditEntry{}, nil
}

// CreateCtxWithDiff creates a new audit entry with diff
func (m *MockAuditEntryCommander) CreateCtxWithDiff(ctx context.Context, eventType EventType, entityID, providerID, agentID, brokerID *UUID, beforeEntity, afterEntity interface{}) (*AuditEntry, error) {
	if m.CreateCtxWithDiffFunc != nil {
		return m.CreateCtxWithDiffFunc(ctx, eventType, entityID, providerID, agentID, brokerID, beforeEntity, afterEntity)
	}
	return &AuditEntry{}, nil
}
