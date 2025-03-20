package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ServiceGroup represents a group of related services
type ServiceGroup struct {
	BaseEntity

	Name string `gorm:"not null"`

	// Relationships
	Services []Service `gorm:"foreignKey:GroupID"`
	BrokerID UUID      `gorm:"not null"`
	Broker   *Broker   `gorm:"foreignKey:BrokerID"`
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
	store Store
}

// NewServiceGroupCommander creates a new ServiceGroupService
func NewServiceGroupCommander(
	store Store,
) *serviceGroupCommander {
	return &serviceGroupCommander{
		store: store,
	}
}

func (s *serviceGroupCommander) Create(ctx context.Context, name string, brokerID UUID) (*ServiceGroup, error) {
	if err := ValidateAuthScope(ctx, &AuthScope{BrokerID: &brokerID}); err != nil {
		return nil, UnauthorizedError{Err: err}
	}

	// Check if broker exists
	broker, err := s.store.BrokerRepo().FindByID(ctx, brokerID)
	if err != nil {
		return nil, NewInvalidInputErrorf("invalid broker: %s", brokerID)
	}
	sg := &ServiceGroup{
		Name:     name,
		BrokerID: brokerID,
		Broker:   broker,
	}
	if err := sg.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}
	if err = s.store.ServiceGroupRepo().Create(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}

func (s *serviceGroupCommander) Update(ctx context.Context, id UUID, name *string) (*ServiceGroup, error) {
	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := ValidateAuthScope(ctx, &AuthScope{BrokerID: &sg.BrokerID}); err != nil {
		return nil, UnauthorizedError{Err: err}
	}

	if name != nil {
		sg.Name = *name
	}
	if err := sg.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}
	if err := s.store.ServiceGroupRepo().Save(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}

func (s *serviceGroupCommander) Delete(ctx context.Context, id UUID) error {
	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := ValidateAuthScope(ctx, &AuthScope{BrokerID: &sg.BrokerID}); err != nil {
		return UnauthorizedError{Err: err}
	}

	return s.store.Atomic(ctx, func(store Store) error {
		numOfServices, err := store.ServiceRepo().CountByGroup(ctx, id)
		if err != nil {
			return err
		}
		if numOfServices > 0 {
			return errors.New("cannot delete service group with associated services")
		}
		return store.ServiceGroupRepo().Delete(ctx, id)
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
	List(ctx context.Context, req *PageRequest) (*PageResponse[ServiceGroup], error)
}
