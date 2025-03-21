package domain

import "context"

// AuthorityType defines the type of entity being audited
type AuthorityType string

// EventType defines the type of audit event
type EventType string

// Predefined authority types
const (
	AuthorityTypeInternal AuthorityType = "internal"
	AuthorityTypeAdmin    AuthorityType = "admin"
	AuthorityTypeProvider AuthorityType = "provider"
	AuthorityTypeAgent    AuthorityType = "agent"
	AuthorityTypeBroker   AuthorityType = "broker"
)

// Predefined event types
const (
	EventTypeStatusChange      EventType = "status_change"
	EventTypeConfigUpdate      EventType = "config_update"
	EventTypeCreated           EventType = "created"
	EventTypeUpdated           EventType = "updated"
	EventTypeDeleted           EventType = "deleted"
	EventTypeExecutionStart    EventType = "execution_start"
	EventTypeExecutionComplete EventType = "execution_complete"
)

// AuditEntry represents an audit log entry
type AuditEntry struct {
	BaseEntity

	AuthorityType AuthorityType `gorm:"not null"`
	AuthorityID   string        `gorm:"not null"`
	EventType     EventType     `gorm:"not null"`
	Properties    JSON          `gorm:"type:jsonb"`

	ProviderID *UUID
	Provider   *Provider `gorm:"foreignKey:ProviderID"`
	AgentID    *UUID
	Agent      *Agent `gorm:"foreignKey:AgentID"`
	BrokerID   *UUID
	Broker     *Broker `gorm:"foreignKey:BrokerID"`
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
	Create(ctx context.Context, authorityType AuthorityType, authorityID string, eventType EventType, properties JSON, providerID, agentID, brokerID *UUID) (*AuditEntry, error)
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
	authorityType AuthorityType,
	authorityID string,
	eventType EventType,
	properties JSON,
	providerID, agentID, brokerID *UUID,
) (*AuditEntry, error) {
	auditEntry := &AuditEntry{
		AuthorityType: authorityType,
		AuthorityID:   authorityID,
		EventType:     eventType,
		Properties:    properties,
		ProviderID:    providerID,
		AgentID:       agentID,
		BrokerID:      brokerID,
	}
	if err := auditEntry.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
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
	List(ctx context.Context, authScope *AuthScope, req *PageRequest) (*PageResponse[AuditEntry], error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthScope, error)
}
