package domain

import (
	"context"
	"fmt"
)

// Broker represents a service broker
type Broker struct {
	BaseEntity
	Name string `json:"name" gorm:"not null"`
}

// TableName returns the table name for the broker
func (Broker) TableName() string {
	return "brokers"
}

// Validate ensures all Broker fields are valid
func (b *Broker) Validate() error {
	if b.Name == "" {
		return fmt.Errorf("broker name cannot be empty")
	}
	return nil
}

// BrokerCommander defines the interface for broker command operations
type BrokerCommander interface {
	// Create creates a new broker
	Create(ctx context.Context, name string) (*Broker, error)

	// Update updates a broker
	Update(ctx context.Context, id UUID, name *string) (*Broker, error)

	// Delete removes a broker by ID
	Delete(ctx context.Context, id UUID) error
}

// brokerCommander is the concrete implementation of BrokerCommander
type brokerCommander struct {
	store          Store
	auditCommander AuditEntryCommander
}

// NewBrokerCommander creates a new BrokerCommander
func NewBrokerCommander(
	store Store,
	auditCommander AuditEntryCommander,
) *brokerCommander {
	return &brokerCommander{
		store:          store,
		auditCommander: auditCommander,
	}
}

func (s *brokerCommander) Create(
	ctx context.Context,
	name string,
) (*Broker, error) {
	var broker *Broker
	err := s.store.Atomic(ctx, func(store Store) error {
		broker = &Broker{
			Name: name,
		}
		if err := broker.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.BrokerRepo().Create(ctx, broker); err != nil {
			return err
		}

		_, err := s.auditCommander.CreateCtx(
			ctx,
			EventTypeBrokerCreated,
			JSON{"state": broker},
			&broker.ID, nil, nil, &broker.ID)
		return err
	})

	if err != nil {
		return nil, err
	}
	return broker, nil
}

func (s *brokerCommander) Update(ctx context.Context,
	id UUID,
	name *string,
) (*Broker, error) {
	beforeBroker, err := s.store.BrokerRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store a copy of the broker before modifications for audit diff
	beforeBrokerCopy := *beforeBroker

	if name != nil {
		beforeBroker.Name = *name
	}
	if err := beforeBroker.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = s.store.Atomic(ctx, func(store Store) error {
		err := store.BrokerRepo().Save(ctx, beforeBroker)
		if err != nil {
			return err
		}

		_, err = s.auditCommander.CreateCtxWithDiff(
			ctx,
			EventTypeBrokerUpdated,
			&id, nil, nil, &id,
			&beforeBrokerCopy, beforeBroker)
		return err
	})

	if err != nil {
		return nil, err
	}
	return beforeBroker, nil
}

func (s *brokerCommander) Delete(ctx context.Context, id UUID) error {
	// Get broker before deletion for audit purposes
	broker, err := s.store.BrokerRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}

	return s.store.Atomic(ctx, func(store Store) error {
		// Delete all tokens associated with this broker before deleting the broker
		if err := store.TokenRepo().DeleteByBrokerID(ctx, id); err != nil {
			return err
		}

		if err := store.BrokerRepo().Delete(ctx, id); err != nil {
			return err
		}

		_, err := s.auditCommander.CreateCtx(
			ctx,
			EventTypeBrokerDeleted,
			JSON{"state": broker},
			&id, nil, nil, &id)
		return err
	})
}

type BrokerRepository interface {
	BrokerQuerier

	// Create creates a new entity
	Create(ctx context.Context, entity *Broker) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *Broker) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error
}

type BrokerQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Broker, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id UUID) (bool, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authScope *AuthScope, req *PageRequest) (*PageResponse[Broker], error)

	// Count returns the total number of entities
	Count(ctx context.Context) (int64, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthScope, error)
}
