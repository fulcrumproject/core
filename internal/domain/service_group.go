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

// NewServiceGroup creates a new service group with validation
func NewServiceGroup(name string, brokerID UUID) *ServiceGroup {
	return &ServiceGroup{
		Name:     name,
		BrokerID: brokerID,
	}
}

// Update updates the service group properties and performs validation
func (sg *ServiceGroup) Update(name *string) error {
	if name != nil {
		sg.Name = *name
	}
	return sg.Validate()
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
	// Validate references
	brokerExists, err := s.store.BrokerRepo().Exists(ctx, brokerID)
	if err != nil {
		return nil, err
	}
	if !brokerExists {
		return nil, NewInvalidInputErrorf("broker with ID %s does not exist", brokerID)
	}

	// Create and save
	var sg *ServiceGroup
	err = s.store.Atomic(ctx, func(store Store) error {
		sg = NewServiceGroup(name, brokerID)
		if err := sg.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.ServiceGroupRepo().Create(ctx, sg); err != nil {
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
	// Validate references
	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store a copy for audit diff
	beforeSgCopy := *sg

	// Update and validate
	if err := sg.Update(name); err != nil {
		return nil, InvalidInputError{Err: err}
	}
	if err := sg.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and audit
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceGroupRepo().Save(ctx, sg); err != nil {
			return err
		}

		_, err := s.auditCommander.CreateCtxWithDiff(
			ctx,
			EventTypeServiceGroupUpdated,
			&id, nil, nil, &sg.BrokerID,
			&beforeSgCopy, sg)
		return err
	})
	if err != nil {
		return nil, err
	}

	return sg, nil
}

func (s *serviceGroupCommander) Delete(ctx context.Context, id UUID) error {
	// Validate references
	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Validate delete conditions
	numOfServices, err := s.store.ServiceRepo().CountByGroup(ctx, id)
	if err != nil {
		return err
	}
	if numOfServices > 0 {
		return errors.New("cannot delete service group with associated services")
	}

	// Delete and audit
	return s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceGroupRepo().Delete(ctx, id); err != nil {
			return err
		}

		_, err = s.auditCommander.CreateCtx(
			ctx,
			EventTypeServiceGroupDeleted,
			JSON{"state": sg}, &id, nil, nil, &sg.BrokerID)
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
