package domain

import (
	"context"
	"fmt"

	"github.com/fulcrumproject/core/pkg/properties"
)

// ParticipantStatus represents the possible statuss of a Participant
type ParticipantStatus string

const (
	EventTypeParticipantCreated EventType = "participant.created"
	EventTypeParticipantUpdated EventType = "participant.updated"
	EventTypeParticipantDeleted EventType = "participant.deleted"

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

	Name   string            `json:"name" gorm:"not null"`
	Status ParticipantStatus `json:"status" gorm:"not null"`

	// Relationships
	Agents []Agent `json:"agents,omitempty" gorm:"foreignKey:ProviderID"` // Agent struct will be updated later
}

// NewParticipant creates a new Participant without validation
func NewParticipant(params CreateParticipantParams) *Participant {
	return &Participant{
		Name:   params.Name,
		Status: params.Status,
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
	if err := p.Status.Validate(); err != nil {
		return err
	}
	return nil
}

// Update updates the participant fields if the pointers are non-nil
func (p *Participant) Update(params UpdateParticipantParams) {
	if params.Name != nil {
		p.Name = *params.Name
	}
	if params.Status != nil {
		p.Status = *params.Status
	}
}

// ParticipantCommander defines the interface for participant command operations
type ParticipantCommander interface {
	// Create creates a new participant
	Create(ctx context.Context, params CreateParticipantParams) (*Participant, error)

	// Update updates a participant
	Update(ctx context.Context, params UpdateParticipantParams) (*Participant, error)

	// Delete removes a participant by ID after checking for dependencies
	Delete(ctx context.Context, id properties.UUID) error
}

type CreateParticipantParams struct {
	Name   string            `json:"name"`
	Status ParticipantStatus `json:"status"`
}

type UpdateParticipantParams struct {
	ID     properties.UUID    `json:"id"`
	Name   *string            `json:"name"`
	Status *ParticipantStatus `json:"status"`
}

// participantCommander is the concrete implementation of ParticipantCommander
type participantCommander struct {
	store Store
}

// NewParticipantCommander creates a new default ParticipantCommander
func NewParticipantCommander(
	store Store,
) ParticipantCommander {
	return &participantCommander{
		store: store,
	}
}

func (c *participantCommander) Create(
	ctx context.Context,
	params CreateParticipantParams,
) (*Participant, error) {
	var participant *Participant
	err := c.store.Atomic(ctx, func(store Store) error {
		participant = NewParticipant(params)
		if err := participant.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}
		if err := store.ParticipantRepo().Create(ctx, participant); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeParticipantCreated, WithInitiatorCtx(ctx), WithParticipant(participant))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return participant, nil
}

func (c *participantCommander) Update(
	ctx context.Context,
	params UpdateParticipantParams,
) (*Participant, error) {
	participant, err := c.store.ParticipantRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	beforeParticipant := *participant

	participant.Update(params)
	if err := participant.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = c.store.Atomic(ctx, func(store Store) error {
		if err := store.ParticipantRepo().Save(ctx, participant); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeParticipantUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeParticipant, participant), WithParticipant(participant))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return participant, nil
}

func (c *participantCommander) Delete(ctx context.Context, id properties.UUID) error {
	participant, err := c.store.ParticipantRepo().Get(ctx, id)
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

		eventEntry, err := NewEvent(EventTypeParticipantDeleted, WithInitiatorCtx(ctx), WithParticipant(participant))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		// Delete associated Tokens
		// TokenRepository.DeleteByParticipantID() will be added in a later step as per the plan
		if err := store.TokenRepo().DeleteByParticipantID(ctx, id); err != nil {
			return fmt.Errorf("failed to delete tokens for participant %s: %w", id, err)
		}

		if err := store.ParticipantRepo().Delete(ctx, id); err != nil {
			return err
		}

		return err
	})
}

// ParticipantRepository defines the interface for participant data operations
type ParticipantRepository interface {
	ParticipantQuerier
	BaseEntityRepository[Participant]
}

// ParticipantQuerier defines the interface for participant query operations
type ParticipantQuerier interface {
	BaseEntityQuerier[Participant]
}
