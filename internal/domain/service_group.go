package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ServiceGroup represents a group of related services
type ServiceGroup struct {
	BaseEntity

	Name string `json:"name" gorm:"not null"`

	// Relationships
	Services []Service `json:"-" gorm:"foreignKey:GroupID"`
	BrokerID UUID      `json:"brokerId" gorm:"not null"`
	Broker   *Broker   `json:"-" gorm:"foreignKey:BrokerID"`
}

// Validate checks if the service group is valid
func (sg *ServiceGroup) Validate() error {
	if sg.Name == "" {
		return errors.New("service group name cannot be empty")
	}
	if sg.BrokerID == uuid.Nil {
		return errors.New("service group broker cannot be nil")
	}
	return nil
}

// TableName returns the table name for the service
func (ServiceGroup) TableName() string {
	return "service_groups"
}

// ServiceGroupCommander defines the interface for service group command operations
type ServiceGroupCommander interface {
	// Create creates a new service group
	Create(ctx context.Context, name string, brokerID UUID) (*ServiceGroup, error)

	// Update updates an existing service group
	Update(ctx context.Context, id UUID, name *string) (*ServiceGroup, error)

	// Delete removes a service group by ID after checking for dependencies
	Delete(ctx context.Context, id UUID) error
}

// serviceGroupCommander is the concrete implementation of ServiceGroupCommander
type serviceGroupCommander struct {
	store          Store
	auditCommander AuditEntryCommander
}

// NewServiceGroupCommander creates a new ServiceGroupService
func NewServiceGroupCommander(
	store Store,
	auditCommander AuditEntryCommander,
) *serviceGroupCommander {
	return &serviceGroupCommander{
		store:          store,
		auditCommander: auditCommander,
	}
}

func (s *serviceGroupCommander) Create(ctx context.Context, name string, brokerID UUID) (*ServiceGroup, error) {
	var sg *ServiceGroup
	err := s.store.Atomic(ctx, func(store Store) error {
		broker, err := store.BrokerRepo().FindByID(ctx, brokerID)
		if err != nil {
			return NewInvalidInputErrorf("invalid broker: %s", brokerID)
		}

		sg = &ServiceGroup{
			Name:     name,
			BrokerID: brokerID,
			Broker:   broker,
		}
		if err := sg.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err = store.ServiceGroupRepo().Create(ctx, sg); err != nil {
			return err
		}

		_, err = s.auditCommander.CreateCtx(
			ctx,
			EventTypeServiceGroupCreated,
			JSON{"state": sg},
			&sg.ID, nil, nil, &brokerID)
		return err
	})

	if err != nil {
		return nil, err
	}
	return sg, nil
}

func (s *serviceGroupCommander) Update(ctx context.Context, id UUID, name *string) (*ServiceGroup, error) {
	beforeSg, err := s.store.ServiceGroupRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store a copy of the serviceGroup before modifications for audit diff
	beforeSgCopy := *beforeSg

	// Apply updates
	if name != nil {
		beforeSg.Name = *name
	}
	if err := beforeSg.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceGroupRepo().Save(ctx, beforeSg); err != nil {
			return err
		}

		_, err := s.auditCommander.CreateCtxWithDiff(
			ctx,
			EventTypeServiceGroupUpdated,
			&id, nil, nil, &beforeSg.BrokerID,
			&beforeSgCopy, beforeSg)
		return err
	})

	if err != nil {
		return nil, err
	}
	return beforeSg, nil
}

func (s *serviceGroupCommander) Delete(ctx context.Context, id UUID) error {
	// Get service group before deletion for audit purposes
	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Store broker ID for audit entry
	brokerID := sg.BrokerID

	return s.store.Atomic(ctx, func(store Store) error {
		numOfServices, err := store.ServiceRepo().CountByGroup(ctx, id)
		if err != nil {
			return err
		}
		if numOfServices > 0 {
			return errors.New("cannot delete service group with associated services")
		}

		if err := store.ServiceGroupRepo().Delete(ctx, id); err != nil {
			return err
		}

		_, err = s.auditCommander.CreateCtx(
			ctx,
			EventTypeServiceGroupDeleted,
			JSON{"state": sg}, &id, nil, nil, &brokerID)
		return err
	})
}

// ServiceGroupRepository defines the interface for the ServiceGroup repository
type ServiceGroupRepository interface {
	ServiceGroupQuerier

	// Create creates a new entity
	Create(ctx context.Context, entity *ServiceGroup) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *ServiceGroup) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error
}

// ServiceGroupRepository defines the interface for the ServiceGroup read-only queries
type ServiceGroupQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*ServiceGroup, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authScope *AuthScope, req *PageRequest) (*PageResponse[ServiceGroup], error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthScope, error)
}
