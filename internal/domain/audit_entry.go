package domain

import "context"

// AuditEntry represents an audit log entry
type AuditEntry struct {
	BaseEntity
	AuthorityType string `gorm:"not null"`
	AuthorityID   string `gorm:"not null"`
	Type          string `gorm:"not null"`
	Properties    JSON   `gorm:"column:properties;type:jsonb"`
}

// TableName returns the table name for the audit entry
func (*AuditEntry) TableName() string {
	return "audit_entries"
}

// Validate ensures all AuditEntry fields are valid
func (p *AuditEntry) Validate() error {
	return nil
}

// AuditEntryCommander handles provider operations with validation
type AuditEntryCommander struct {
	repo AuditEntryRepository
}

// NewAuditEntryCommander creates a new AuditEntryService
func NewAuditEntryCommander(
	repo AuditEntryRepository,
) *AuditEntryCommander {
	return &AuditEntryCommander{
		repo: repo,
	}
}

// Create creates a new audit-entry with validation
func (s *AuditEntryCommander) Create(
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
	if err := s.repo.Create(ctx, auditEntry); err != nil {
		return nil, err
	}
	return auditEntry, nil
}

type AuditEntryRepository interface {
	// Create stores a new audit entry
	Create(ctx context.Context, entry *AuditEntry) error

	// List retrieves a list of audit entries based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[AuditEntry], error)
}

type AuditEntryQuerier interface {
	// List retrieves a list of audit entries based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[AuditEntry], error)
}
