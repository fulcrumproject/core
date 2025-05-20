package domain

import (
	"context"
	"fmt"
)

// ParticipantState represents the possible states of a Participant
type ParticipantState string

const (
	ParticipantEnabled  ParticipantState = "Enabled"
	ParticipantDisabled ParticipantState = "Disabled"
)

// Validate checks if the participant state is valid
func (s ParticipantState) Validate() error {
	switch s {
	case ParticipantEnabled, ParticipantDisabled:
		return nil
	default:
		return fmt.Errorf("invalid participant state: %s", s)
	}
}

// ParseParticipantState parses a string into a ParticipantState
func ParseParticipantState(value string) (ParticipantState, error) {
	state := ParticipantState(value)
	if err := state.Validate(); err != nil {
		return "", err
	}
	return state, nil
}

// Participant represents a unified entity for providers and brokers
type Participant struct {
	BaseEntity

	Name        string           `json:"name" gorm:"not null"`
	CountryCode CountryCode      `json:"countryCode,omitempty" gorm:"size:2"`
	Attributes  Attributes       `json:"attributes,omitempty" gorm:"type:jsonb"`
	State       ParticipantState `json:"state" gorm:"not null"`

	// Relationships
	Agents []Agent `json:"agents,omitempty" gorm:"foreignKey:ProviderID"` // Agent struct will be updated later
}

// NewParticipant creates a new Participant without validation
func NewParticipant(name string, state ParticipantState, countryCode CountryCode, attributes Attributes) *Participant {
	return &Participant{
		Name:        name,
		State:       state,
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
	if err := p.State.Validate(); err != nil {
		return err
	}
	return nil
}

// Update updates the participant fields if the pointers are non-nil
func (p *Participant) Update(name *string, state *ParticipantState, countryCode *CountryCode, attributes *Attributes) {
	if name != nil {
		p.Name = *name
	}
	if state != nil {
		p.State = *state
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
	Create(ctx context.Context, name string, state ParticipantState, countryCode CountryCode, attributes Attributes) (*Participant, error)

	// Update updates a participant
	Update(ctx context.Context, id UUID, name *string, state *ParticipantState, countryCode *CountryCode, attributes *Attributes) (*Participant, error)

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
	state ParticipantState,
	countryCode CountryCode,
	attributes Attributes,
) (*Participant, error) {
	var participant *Participant
	err := c.store.Atomic(ctx, func(store Store) error {
		participant = NewParticipant(name, state, countryCode, attributes)
		if err := participant.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.ParticipantRepo().Create(ctx, participant); err != nil {
			return err
		}
		// EventTypeParticipantCreated will be defined in audit_entry.go as per plan
		_, err := c.auditCommander.CreateCtx(
			ctx, "EventTypeParticipantCreated", JSON{"state": participant}, // Placeholder
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
	state *ParticipantState,
	countryCode *CountryCode,
	attributes *Attributes,
) (*Participant, error) {
	participant, err := c.store.ParticipantRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	beforeParticipant := *participant

	participant.Update(name, state, countryCode, attributes)
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
			JSON{"state": participant}, &id, &id, nil, nil) // ParticipantID is the authority
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
	AuthScope(ctx context.Context, id UUID) (*AuthScope, error)
}
