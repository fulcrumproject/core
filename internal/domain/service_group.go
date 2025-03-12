package domain

import (
	"context"
	"errors"
)

// ServiceGroup represents a group of related services
type ServiceGroup struct {
	BaseEntity
	Name     string    `gorm:"not null"`
	Services []Service `gorm:"foreignKey:GroupID"`
}

// Validate checks if the service group is valid
func (sg *ServiceGroup) Validate() error {
	if sg.Name == "" {
		return errors.New("service group name cannot be empty")
	}
	return nil
}

// TableName returns the table name for the service
func (*ServiceGroup) TableName() string {
	return "service_groups"
}

// ServiceGroupCommander handles service group operations with validation
type ServiceGroupCommander struct {
	store Store
}

// NewServiceGroupCommander creates a new ServiceGroupService
func NewServiceGroupCommander(
	store Store,
) *ServiceGroupCommander {
	return &ServiceGroupCommander{
		store: store,
	}
}

// Create creates a new service group with validation
func (s *ServiceGroupCommander) Create(ctx context.Context, name string) (*ServiceGroup, error) {
	sg := &ServiceGroup{
		Name: name,
	}
	if err := sg.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.ServiceGroupRepo().Create(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}

// Save updates an existing service group with validation
func (s *ServiceGroupCommander) Update(ctx context.Context, id UUID, name *string) (*ServiceGroup, error) {
	sg, err := s.store.ServiceGroupRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		sg.Name = *name
	}
	if err := sg.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.ServiceGroupRepo().Save(ctx, sg); err != nil {
		return nil, err
	}
	return sg, nil
}

// Delete removes an entity by ID
func (s *ServiceGroupCommander) Delete(ctx context.Context, id UUID) error {
	_, err := s.store.ServiceGroupRepo().FindByID(ctx, id)
	if err != nil {
		return err
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
	// Create creates a new entity
	Create(ctx context.Context, entity *ServiceGroup) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *ServiceGroup) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*ServiceGroup, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[ServiceGroup], error)
}

// ServiceGroupRepository defines the interface for the ServiceGroup read-only queries
type ServiceGroupQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*ServiceGroup, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[ServiceGroup], error)
}
