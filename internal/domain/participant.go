package domain

import (
	"context"
	"fmt"
)

// ParticipantStatus represents the possible statuss of a Participant
type ParticipantStatus string

const (
	ParticipantEnabled  ParticipantStatus = "Enabled"
	ParticipantDisabled ParticipantStatus = "Disabled"
)

// Validate checks if the participant status is valid
func (s ParticipantStatus) Validate() error {
	switch s {
	case ParticipantEnabled, ParticipantDisabled:
		return nil
	default:
		return fmt.Errorf("invalid participant status: %s", s)
	}
}

// ParseParticipantStatus parses a string into a ParticipantStatus
func ParseParticipantStatus(value string) (ParticipantStatus, error) {
	status := ParticipantStatus(value)
	if err := status.Validate(); err != nil {
		return "", err
	}
	return status, nil
}

// Participant represents a unified entity for providers and consumers
type Participant struct {
	BaseEntity

	Name        string            `json:"name" gorm:"not null"`
	CountryCode CountryCode       `json:"countryCode,omitempty" gorm:"size:2"`
	Attributes  Attributes        `json:"attributes,omitempty" gorm:"type:jsonb"`
	Status      ParticipantStatus `json:"status" gorm:"not null"`

	// Relationships
	Agents []Agent `json:"agents,omitempty" gorm:"foreignKey:ProviderID"` // Agent struct will be updated later
}

// NewParticipant creates a new Participant without validation
func NewParticipant(name string, status ParticipantStatus, countryCode CountryCode, attributes Attributes) *Participant {
	return &Participant{
		Name:        name,
		Status:      status,
		CountryCode: countryCode,
		Attributes:  attributes,
	}
}

// TableName returns the table name for the participant
func (Participant) TableName() string {
	return "participants"
}

// Validate ensures all Participant fields are valid
func (p *Participant) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("participant name cannot be empty")
	}
	if err := p.CountryCode.Validate(); err != nil {
		// Allow empty country code
		if string(p.CountryCode) != "" {
			return err
		}
	}
	if p.Attributes != nil {
		if err := p.Attributes.Validate(); err != nil {
			return err
		}
	}
	if err := p.Status.Validate(); err != nil {
		return err
	}
	return nil
}

// Update updates the participant fields if the pointers are non-nil
func (p *Participant) Update(name *string, status *ParticipantStatus, countryCode *CountryCode, attributes *Attributes) {
	if name != nil {
		p.Name = *name
	}
	if status != nil {
		p.Status = *status
	}
	if countryCode != nil {
		p.CountryCode = *countryCode
	}
	if attributes != nil {
		p.Attributes = *attributes
	}
}

// ParticipantCommander defines the interface for participant command operations
type ParticipantCommander interface {
	// Create creates a new participant
	Create(ctx context.Context, name string, status ParticipantStatus, countryCode CountryCode, attributes Attributes) (*Participant, error)

	// Update updates a participant
	Update(ctx context.Context, id UUID, name *string, status *ParticipantStatus, countryCode *CountryCode, attributes *Attributes) (*Participant, error)

	// Delete removes a participant by ID after checking for dependencies
	Delete(ctx context.Context, id UUID) error
}

// participantCommander is the concrete implementation of ParticipantCommander
type participantCommander struct {
	store          Store
	auditCommander AuditEntryCommander
}

// NewParticipantCommander creates a new default ParticipantCommander
func NewParticipantCommander(
	store Store,
	auditCommander AuditEntryCommander,
) ParticipantCommander {
	return &participantCommander{
		store:          store,
		auditCommander: auditCommander,
	}
}

func (c *participantCommander) Create(
	ctx context.Context,
	name string,
	status ParticipantStatus,
	countryCode CountryCode,
	attributes Attributes,
) (*Participant, error) {
	var participant *Participant
	err := c.store.Atomic(ctx, func(store Store) error {
		participant = NewParticipant(name, status, countryCode, attributes)
		if err := participant.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.ParticipantRepo().Create(ctx, participant); err != nil {
			return err
		}
		// EventTypeParticipantCreated will be defined in audit_entry.go as per plan
		_, err := c.auditCommander.CreateCtx(
			ctx, "EventTypeParticipantCreated", JSON{"status": participant}, // Placeholder
			&participant.ID, &participant.ID, nil, nil) // ParticipantID is the authority
		return err
	})
	if err != nil {
		return nil, err
	}
	return participant, nil
}

func (c *participantCommander) Update(
	ctx context.Context,
	id UUID,
	name *string,
	status *ParticipantStatus,
	countryCode *CountryCode,
	attributes *Attributes,
) (*Participant, error) {
	participant, err := c.store.ParticipantRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	beforeParticipant := *participant

	participant.Update(name, status, countryCode, attributes)
	if err := participant.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = c.store.Atomic(ctx, func(store Store) error {
		if err := store.ParticipantRepo().Save(ctx, participant); err != nil {
			return err
		}
		// EventTypeParticipantUpdated will be defined in audit_entry.go as per plan
		_, err = c.auditCommander.CreateCtxWithDiff(ctx, "EventTypeParticipantUpdated", // Placeholder
			&id, &id, nil, nil, &beforeParticipant, participant) // ParticipantID is the authority
		return err
	})
	if err != nil {
		return nil, err
	}
	return participant, nil
}

func (c *participantCommander) Delete(ctx context.Context, id UUID) error {
	participant, err := c.store.ParticipantRepo().FindByID(ctx, id)
	if err != nil {
		return err // Handles NotFoundError as well
	}

	return c.store.Atomic(ctx, func(store Store) error {
		// Check for dependent Agents
		// AgentRepo().CountByParticipant() will need to be added to AgentQuerier/AgentRepository
		agentCount, err := store.AgentRepo().CountByProvider(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to count agents for participant %s: %w", id, err)
		}
		if agentCount > 0 {
			return NewInvalidInputErrorf("cannot delete participant %s: %d dependent agent(s) exist", id, agentCount)
		}

		// Delete associated Tokens
		// TokenRepository.DeleteByParticipantID() will be added in a later step as per the plan
		if err := store.TokenRepo().DeleteByParticipantID(ctx, id); err != nil {
			return fmt.Errorf("failed to delete tokens for participant %s: %w", id, err)
		}

		if err := store.ParticipantRepo().Delete(ctx, id); err != nil {
			return err
		}

		// EventTypeParticipantDeleted will be defined in audit_entry.go as per plan
		_, err = c.auditCommander.CreateCtx(ctx, "EventTypeParticipantDeleted", // Placeholder
			JSON{"status": participant}, &id, &id, nil, nil) // ParticipantID is the authority
		return err
	})
}

// ParticipantRepository defines the interface for participant data operations
type ParticipantRepository interface {
	ParticipantQuerier

	// Create creates a new entity
	Create(ctx context.Context, entity *Participant) error

	// Save updates an existing entity
	Save(ctx context.Context, entity *Participant) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error
}

// ParticipantQuerier defines the interface for participant query operations
type ParticipantQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Participant, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id UUID) (bool, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[Participant], error)

	// AuthScope retrieves the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error)
}
