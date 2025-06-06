package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wI2L/jsondiff"
)

// AuthorityType defines the type of entity being audited
type AuthorityType string

// EventType defines the type of audit event
type EventType string

// Predefined authority types
const (
	AuthorityTypeInternal    AuthorityType = "internal"
	AuthorityTypeAdmin       AuthorityType = "admin"
	AuthorityTypeParticipant AuthorityType = "participant"
	AuthorityTypeAgent       AuthorityType = "agent"
)

// Predefined event types
const (
	// Agent commands
	EventTypeAgentCreated EventType = "agent_created"
	EventTypeAgentUpdated EventType = "agent_updated"
	EventTypeAgentDeleted EventType = "agent_deleted"

	// Particpiant commands
	EventTypeParticipantCreated EventType = "participant_created"
	EventTypeParticipantUpdated EventType = "participant_updated"
	EventTypeParticipantDeleted EventType = "participant_deleted"

	// Service commands
	EventTypeServiceCreated      EventType = "service_created"
	EventTypeServiceUpdated      EventType = "service_updated"
	EventTypeServiceTransitioned EventType = "service_transitioned"
	EventTypeServiceRetried      EventType = "service_retried"

	// ServiceGroup commands
	EventTypeServiceGroupCreated EventType = "service_group_created"
	EventTypeServiceGroupUpdated EventType = "service_group_updated"
	EventTypeServiceGroupDeleted EventType = "service_group_deleted"

	// Token commands
	EventTypeTokenCreated     EventType = "token_created"
	EventTypeTokenUpdated     EventType = "token_updated"
	EventTypeTokenDeleted     EventType = "token_deleted"
	EventTypeTokenRegenerated EventType = "token_regenerate"

	// MetricType commands
	EventTypeMetricTypeCreated EventType = "metric_type_created"
	EventTypeMetricTypeUpdated EventType = "metric_type_updated"
	EventTypeMetricTypeDeleted EventType = "metric_type_deleted"
)

// AuditEntry represents an audit log entry
type AuditEntry struct {
	BaseEntity

	AuthorityType AuthorityType `gorm:"not null"`
	AuthorityID   string        `gorm:"not null"`
	EventType     EventType     `gorm:"not null"`
	Properties    JSON          `gorm:"type:jsonb"`
	EntityID      *UUID         `gorm:"index"`

	ProviderID *UUID `gorm:"type:uuid"`
	AgentID    *UUID `gorm:"type:uuid"`
	ConsumerID *UUID `gorm:"type:uuid"`
}

// NewEventAudit creates a new audit entry for simple event audits
func NewEventAudit(
	authorityType AuthorityType,
	authorityID string,
	eventType EventType,
	properties JSON,
	entityID, providerID, agentID, consumerID *UUID,
) *AuditEntry {
	return &AuditEntry{
		AuthorityType: authorityType,
		AuthorityID:   authorityID,
		EventType:     eventType,
		Properties:    properties,
		EntityID:      entityID,
		ProviderID:    providerID,
		AgentID:       agentID,
		ConsumerID:    consumerID,
	}
}

// GenerateDiff calculates and stores the diff between two entity statuss
func (p *AuditEntry) GenerateDiff(beforeEntity, afterEntity interface{}) error {
	// Convert entities to JSON
	beforeJSON, err := json.Marshal(beforeEntity)
	if err != nil {
		return fmt.Errorf("failed to marshal 'before' entity: %w", err)
	}

	afterJSON, err := json.Marshal(afterEntity)
	if err != nil {
		return fmt.Errorf("failed to marshal 'after' entity: %w", err)
	}

	// Generate RFC 6902 JSON Patch using jsondiff
	patch, err := jsondiff.CompareJSON(beforeJSON, afterJSON)
	if err != nil {
		return fmt.Errorf("failed to generate diff: %w", err)
	}

	// Initialize properties if nil
	if p.Properties == nil {
		p.Properties = JSON{}
	}

	// Store diff in properties
	p.Properties["diff"] = patch

	return nil
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
	Create(ctx context.Context, authorityType AuthorityType, authorityID string, eventType EventType, properties JSON, entityID, providerID, agentID, consumerID *UUID) (*AuditEntry, error)

	// CreateWithDiff creates an audit entry with a diff between two entity statuss
	CreateWithDiff(ctx context.Context, authorityType AuthorityType, authorityID string, eventType EventType,
		entityID, providerID, agentID, consumerID *UUID, beforeEntity, afterEntity interface{}) (*AuditEntry, error)

	// CreateCtx extracts authority info from context and creates an audit entry
	CreateCtx(ctx context.Context, eventType EventType, properties JSON, entityID, providerID, agentID, consumerID *UUID) (*AuditEntry, error)

	// CreateCtxWithDiff extracts authority info from context and creates an audit entry with diff
	CreateCtxWithDiff(ctx context.Context, eventType EventType, entityID, providerID, agentID, consumerID *UUID, beforeEntity, afterEntity interface{}) (*AuditEntry, error)
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
	entityID,
	providerID, agentID, consumerID *UUID,
) (*AuditEntry, error) {
	// Collect data - all data is already provided via parameters

	// Create and validate
	auditEntry := NewEventAudit(
		authorityType,
		authorityID,
		eventType,
		properties,
		entityID,
		providerID,
		agentID,
		consumerID,
	)
	if err := auditEntry.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save
	if err := s.store.AuditEntryRepo().Create(ctx, auditEntry); err != nil {
		return nil, err
	}

	return auditEntry, nil
}

// CreateWithDiff creates an audit entry with a JSON diff between two entity statuss using jsondiff
func (s *auditEntryCommander) CreateWithDiff(
	ctx context.Context,
	authorityType AuthorityType,
	authorityID string,
	eventType EventType,
	entityID, providerID, agentID, consumerID *UUID,
	beforeEntity, afterEntity interface{},
) (*AuditEntry, error) {
	// Collect data - all data is already provided via parameters

	// Create and validate
	auditEntry := NewEventAudit(
		authorityType,
		authorityID,
		eventType,
		nil, // We'll generate the properties with the diff
		entityID,
		providerID,
		agentID,
		consumerID,
	)

	// Generate diff
	if err := auditEntry.GenerateDiff(beforeEntity, afterEntity); err != nil {
		return nil, err
	}

	if err := auditEntry.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save
	if err := s.store.AuditEntryRepo().Create(ctx, auditEntry); err != nil {
		return nil, err
	}

	return auditEntry, nil
}

// CreateCtx extracts authority info from context and creates an audit entry
func (s *auditEntryCommander) CreateCtx(
	ctx context.Context,
	eventType EventType,
	properties JSON,
	entityID, providerID, agentID, consumerID *UUID,
) (*AuditEntry, error) {
	// Collect data - extract authority information from context
	authorityType, authorityID := ExtractAuditAuthority(ctx)

	// Create and validate, save (delegated to Create method)
	return s.Create(ctx, authorityType, authorityID, eventType, properties, entityID, providerID, agentID, consumerID)
}

// CreateCtxWithDiff extracts authority info from context and creates an audit entry with diff
func (s *auditEntryCommander) CreateCtxWithDiff(
	ctx context.Context,
	eventType EventType,
	entityID, providerID, agentID, consumerID *UUID,
	beforeEntity, afterEntity interface{},
) (*AuditEntry, error) {
	// Collect data - extract authority information from context
	authorityType, authorityID := ExtractAuditAuthority(ctx)

	// Create and validate, save (delegated to CreateWithDiff method)
	return s.CreateWithDiff(ctx, authorityType, authorityID, eventType, entityID, providerID, agentID, consumerID, beforeEntity, afterEntity)
}

// ExtractAuditAuthority extracts authority information from context
// Returns the authority type and ID for audit entries
func ExtractAuditAuthority(ctx context.Context) (AuthorityType, string) {
	identity := MustGetAuthIdentity(ctx)

	// Map role to authority type
	var authorityType AuthorityType
	switch identity.Role() {
	case RoleFulcrumAdmin:
		authorityType = AuthorityTypeAdmin
	case RoleParticipant:
		authorityType = AuthorityTypeParticipant
	case RoleAgent:
		authorityType = AuthorityTypeAgent
	default:
		authorityType = AuthorityTypeInternal
	}

	return authorityType, identity.ID().String()
}

type AuditEntryRepository interface {
	AuditEntryQuerier

	// Create stores a new audit entry
	Create(ctx context.Context, entry *AuditEntry) error
}

type AuditEntryQuerier interface {
	// List retrieves a list of audit entries based on the provided filters
	List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[AuditEntry], error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error)
}
