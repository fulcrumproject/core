package domain

import "context"

// AuditEntry represents an audit log entry
type AuditEntry struct {
	BaseEntity
	// TODO Add Provider, Broker, Agent
	AuthorityType string `gorm:"not null"`
	AuthorityID   string `gorm:"not null"`
	Type          string `gorm:"not null"`
	Properties    JSON   `gorm:"type:jsonb"`
}

// TableName returns the table name for the audit entry
func (AuditEntry) TableName() string {
	return "audit_entries"
}

// Validate ensures all AuditEntry fields are valid
func (p *AuditEntry) Validate() error {
	return nil
}

// AuditEntryCommander defines the interface for audit entry command operations
type AuditEntryCommander interface {
	// Create creates a new audit entry
	Create(ctx context.Context, authorityType, authorityID, auditType string, properties JSON) (*AuditEntry, error)
}

// auditEntryCommander is the concrete implementation of AuditEntryCommander
type auditEntryCommander struct {
	store Store
}

// NewAuditEntryCommander creates a new AuditEntryService
func NewAuditEntryCommander(
	store Store,
) *auditEntryCommander {
	return &auditEntryCommander{
		store: store,
	}
}

func (s *auditEntryCommander) Create(
	ctx context.Context,
	authorityType,
	authorityID,
	auditType string,
	properties JSON,
) (*AuditEntry, error) {
	auditEntry := &AuditEntry{
		AuthorityType: authorityType,
		AuthorityID:   authorityID,
		Type:          auditType,
		Properties:    properties,
	}
	if err := auditEntry.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.AuditEntryRepo().Create(ctx, auditEntry); err != nil {
		return nil, err
	}
	return auditEntry, nil
}

type AuditEntryRepository interface {
	AuditEntryQuerier

	// Create stores a new audit entry
	Create(ctx context.Context, entry *AuditEntry) error
}

type AuditEntryQuerier interface {
	// List retrieves a list of audit entries based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[AuditEntry], error)
}
