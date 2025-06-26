package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/wI2L/jsondiff"
)

// InitiatorType defines the type of actor that initiated the event
type InitiatorType string

// EventType defines the type of event
type EventType string

// Predefined initiator types
const (
	InitiatorTypeSystem InitiatorType = "system"
	InitiatorTypeUser   InitiatorType = "user"
)

// Event represents an event in the system
type Event struct {
	BaseEntity

	// For strict ordering of events
	SequenceNumber int64 `json:"sequenceNumber" gorm:"autoIncrement;uniqueIndex;not null"`

	InitiatorType InitiatorType `gorm:"not null"`
	InitiatorID   string        `gorm:"not null"`

	Type    EventType       `gorm:"not null"`
	Payload properties.JSON `gorm:"type:jsonb"`

	// Target entity ID for the event
	EntityID *properties.UUID `gorm:"index"`

	// Optional IDs for related entities and filtering
	ParticipantID *properties.UUID `gorm:"type:uuid"`
	ProviderID    *properties.UUID `gorm:"type:uuid"`
	AgentID       *properties.UUID `gorm:"type:uuid"`
	ConsumerID    *properties.UUID `gorm:"type:uuid"`
}

// EventOption defines a function that configures an EventEntry
type EventOption func(*Event) error

// WithAgent sets the entity ID for the event
func WithAgent(t *Agent) EventOption {
	return func(e *Event) error {
		e.EntityID = &t.ID
		e.AgentID = &t.ID
		e.ProviderID = &t.ProviderID
		return nil
	}
}

// WithToken sets the entity ID for the event
func WithToken(t *Token) EventOption {
	return func(e *Event) error {
		e.EntityID = &t.ID
		e.ParticipantID = t.ParticipantID
		e.AgentID = t.AgentID
		return nil
	}
}

// WithMetricType sets the entity ID for the event
func WithMetricType(t *MetricType) EventOption {
	return func(e *Event) error {
		e.EntityID = &t.ID
		return nil
	}
}

// WithParticipant sets the entity ID for the event
func WithParticipant(t *Participant) EventOption {
	return func(e *Event) error {
		e.EntityID = &t.ID
		e.ParticipantID = &t.ID
		return nil
	}
}

// WithJob sets the entity ID for the event
func WithJob(t *Job) EventOption {
	return func(e *Event) error {
		e.EntityID = &t.ID
		e.AgentID = &t.AgentID
		e.ProviderID = &t.ProviderID
		e.ConsumerID = &t.ConsumerID
		return nil
	}
}

// WithService sets the entity ID for the event
func WithService(t *Service) EventOption {
	return func(e *Event) error {
		e.EntityID = &t.ID
		e.AgentID = &t.AgentID
		e.ProviderID = &t.ProviderID
		e.ConsumerID = &t.ConsumerID
		return nil
	}
}

// WithServiceGroup sets the entity ID for the event
func WithServiceGroup(t *ServiceGroup) EventOption {
	return func(e *Event) error {
		e.EntityID = &t.ID
		e.ConsumerID = &t.ConsumerID
		return nil
	}
}

// WithInitiatorCtx sets the event from a context
func WithInitiatorCtx(ctx context.Context) EventOption {
	return func(e *Event) error {
		identity := auth.MustGetIdentity(ctx)
		e.InitiatorType = InitiatorTypeUser
		e.InitiatorID = identity.ID.String()
		return nil
	}
}

// WithDiff
func WithDiff(beforeEntity, afterEntity any) EventOption {
	return func(e *Event) error {
		// Convert entities to properties.JSON
		beforeJSON, err := json.Marshal(beforeEntity)
		if err != nil {
			return fmt.Errorf("failed to marshal 'before' entity: %w", err)
		}

		afterJSON, err := json.Marshal(afterEntity)
		if err != nil {
			return fmt.Errorf("failed to marshal 'after' entity: %w", err)
		}

		// Generate RFC 6902 properties.JSON Patch using jsondiff
		patch, err := jsondiff.CompareJSON(beforeJSON, afterJSON, jsondiff.Invertible())
		if err != nil {
			return fmt.Errorf("failed to generate diff: %w", err)
		}

		e.Payload = properties.JSON{
			"diff": patch,
		}

		return nil
	}
}

// NewEvent creates a new event
func NewEvent(
	eventType EventType,
	opts ...EventOption,
) (*Event, error) {
	ae := &Event{
		InitiatorType: InitiatorTypeSystem,
		Type:          eventType,
	}

	for _, opt := range opts {
		err := opt(ae)
		if err != nil {
			return nil, fmt.Errorf("failed to apply event option: %w", err)
		}
	}

	return ae, ae.Validate()
}

// TableName returns the table name for the event
func (Event) TableName() string {
	return "events"
}

// Validate ensures all Event fields are valid
func (p *Event) Validate() error {
	return nil
}

type EventRepository interface {
	BaseEntityRepository[Event]
	EventQuerier

	// Create stores a new event
	Create(ctx context.Context, entry *Event) error
}

type EventQuerier interface {
	BaseEntityQuerier[Event]

	// ListFromSequence retrieves events starting from a specific sequence number
	ListFromSequence(ctx context.Context, fromSequenceNumber int64, limit int) ([]*Event, error)

	// Uptime returns the uptime in percentage of a service in a time range
	Uptime(ctx context.Context, serviceID properties.UUID, start time.Time, end time.Time) (float64, error)
}
