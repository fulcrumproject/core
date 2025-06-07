package domain

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

const (
	EventTypeServiceActivationCreated      EventType = "service_activation_created"
	EventTypeServiceActivationUpdated      EventType = "service_activation_updated"
	EventTypeServiceActivationDeleted      EventType = "service_activation_deleted"
	EventTypeServiceActivationAgentAdded   EventType = "service_activation_agent_added"
	EventTypeServiceActivationAgentRemoved EventType = "service_activation_agent_removed"
)

// ServiceActivation represents a standardized service activation with specific tags
// that can be provisioned via a set of agents
type ServiceActivation struct {
	BaseEntity

	// Tags representing certifications or capabilities of this service activation
	Tags []string `json:"tags" gorm:"type:text[]"`

	// Relationships
	ProviderID    UUID         `json:"providerId" gorm:"not null"`
	Provider      *Participant `json:"-" gorm:"foreignKey:ProviderID"`
	ServiceTypeID UUID         `json:"serviceTypeId" gorm:"not null"`
	ServiceType   *ServiceType `json:"-" gorm:"foreignKey:ServiceTypeID"`
	Agents        []Agent      `gorm:"many2many:service_activations_agents;"`
}

// NewServiceActivation creates a new service activation with proper validation
func NewServiceActivation(providerID UUID, serviceTypeID UUID, tags []string) *ServiceActivation {
	return &ServiceActivation{
		ProviderID:    providerID,
		ServiceTypeID: serviceTypeID,
		Tags:          tags,
	}
}

// TableName returns the table name for the service activation
func (ServiceActivation) TableName() string {
	return "service_activations"
}

// Validate ensures all service activation fields are valid
func (sa *ServiceActivation) Validate() error {
	if sa.ProviderID == uuid.Nil {
		return errors.New("provider ID cannot be empty")
	}

	if sa.ServiceTypeID == uuid.Nil {
		return errors.New("service type ID cannot be empty")
	}

	// Validate tag length
	for i, tag := range sa.Tags {
		if len(tag) == 0 {
			return fmt.Errorf("tag at index %d cannot be empty", i)
		}
		if len(tag) > 100 {
			return fmt.Errorf("tag at index %d exceeds maximum length of 100 characters", i)
		}
	}

	return nil
}

// Update updates the service activation's fields
func (sa *ServiceActivation) Update(tags *[]string) bool {
	updated := false

	if tags != nil {
		sa.Tags = *tags
		updated = true
	}

	return updated
}

// ServiceActivationRepository defines the interface for the ServiceActivation repository
type ServiceActivationRepository interface {
	ServiceActivationQuerier

	Create(ctx context.Context, entity *ServiceActivation) error

	Save(ctx context.Context, entity *ServiceActivation) error

	Delete(ctx context.Context, id UUID) error
}

// ServiceActivationQuerier defines the interface for the ServiceActivation read-only queries
type ServiceActivationQuerier interface {
	FindByID(ctx context.Context, id UUID) (*ServiceActivation, error)

	Exists(ctx context.Context, id UUID) (bool, error)

	List(ctx context.Context, authIdentityScope *AuthIdentityScope, req *PageRequest) (*PageResponse[ServiceActivation], error)

	AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

// ServiceActivationCommander defines the interface for service activation command operations
type ServiceActivationCommander interface {
	Create(ctx context.Context, providerID UUID, serviceTypeID UUID, tags []string) (*ServiceActivation, error)

	Update(ctx context.Context, id UUID, tags *[]string) (*ServiceActivation, error)

	Delete(ctx context.Context, id UUID) error
}

// serviceActivationCommander is the concrete implementation of ServiceActivationCommander
type serviceActivationCommander struct {
	store Store
}

// NewServiceActivationCommander creates a new default ServiceActivationCommander
func NewServiceActivationCommander(
	store Store,
) *serviceActivationCommander {
	return &serviceActivationCommander{
		store: store,
	}
}

func (s *serviceActivationCommander) Create(
	ctx context.Context,
	providerID UUID,
	serviceTypeID UUID,
	tags []string,
) (*ServiceActivation, error) {
	return nil, errors.New("not implemented")
}

func (s *serviceActivationCommander) Update(
	ctx context.Context,
	id UUID,
	tags *[]string,
) (*ServiceActivation, error) {
	return nil, errors.New("not implemented")
}

func (s *serviceActivationCommander) Delete(ctx context.Context, id UUID) error {
	return errors.New("not implemented")
}
