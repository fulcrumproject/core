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

// NewEventAuditCtx creates a new audit entry using context to extract authority information
func NewEventAuditCtx(
	ctx context.Context,
	eventType EventType,
	properties JSON,
	entityID, providerID, agentID, consumerID *UUID,
) (*AuditEntry, error) {
	authorityType, authorityID := extractAuditAuthority(ctx)
	ae := &AuditEntry{
		AuthorityType: authorityType,
		AuthorityID:   authorityID,
		EventType:     eventType,
		Properties:    properties,
		EntityID:      entityID,
		ProviderID:    providerID,
		AgentID:       agentID,
		ConsumerID:    consumerID,
	}
	return ae, ae.Validate()
}

// NewEventAuditCtx creates a new audit entry using context to extract authority information
func NewEventAuditCtxDiff(
	ctx context.Context,
	eventType EventType,
	properties JSON,
	entityID, providerID, agentID, consumerID *UUID,
	beforeEntity, afterEntity any,
) (*AuditEntry, error) {
	authorityType, authorityID := extractAuditAuthority(ctx)
	ae := &AuditEntry{
		AuthorityType: authorityType,
		AuthorityID:   authorityID,
		EventType:     eventType,
		Properties:    properties,
		EntityID:      entityID,
		ProviderID:    providerID,
		AgentID:       agentID,
		ConsumerID:    consumerID,
	}
	if err := ae.generateDiff(beforeEntity, afterEntity); err != nil {
		return nil, err
	}
	return ae, ae.Validate()
}

// generateDiff calculates and stores the diff between two entity statuss
func (p *AuditEntry) generateDiff(beforeEntity, afterEntity any) error {
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

// extractAuditAuthority extracts authority information from context
// Returns the authority type and ID for audit entries
func extractAuditAuthority(ctx context.Context) (AuthorityType, string) {
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
