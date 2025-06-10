package domain

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
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
	Tags pq.StringArray `json:"tags" gorm:"type:text[]"`

	// Relationships
	ProviderID    UUID         `json:"providerId" gorm:"not null"`
	Provider      *Participant `json:"-" gorm:"foreignKey:ProviderID"`
	ServiceTypeID UUID         `json:"serviceTypeId" gorm:"not null"`
	ServiceType   *ServiceType `json:"-" gorm:"foreignKey:ServiceTypeID"`
	Agents        []Agent      `gorm:"many2many:service_activation_agents;"`
}

// NewServiceActivation creates a new service activation with proper validation
func NewServiceActivation(providerID UUID, serviceTypeID UUID, tags []string) *ServiceActivation {
	return &ServiceActivation{
		ProviderID:    providerID,
		ServiceTypeID: serviceTypeID,
		Tags:          pq.StringArray(tags),
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
	for i, tag := range []string(sa.Tags) {
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
		sa.Tags = pq.StringArray(*tags)
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

	FindByServiceTypeAndTags(ctx context.Context, serviceTypeID UUID, tags []string) ([]*ServiceActivation, error)

	FindByAgentAndServiceType(ctx context.Context, agentID UUID, serviceTypeID UUID) (*ServiceActivation, error)
}

// ServiceActivationCommander defines the interface for service activation command operations
type ServiceActivationCommander interface {
	Create(ctx context.Context, providerID UUID, serviceTypeID UUID, tags []string, agentIDs []UUID) (*ServiceActivation, error)

	Update(ctx context.Context, id UUID, tags *[]string, agentIDs *[]UUID) (*ServiceActivation, error)

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

// validateAndGetAgents validates agent IDs and returns the agents if they belong to the provider
func validateAndGetAgents(ctx context.Context, store Store, providerID UUID, agentIDs []UUID) ([]Agent, error) {
	var agents []Agent
	if len(agentIDs) == 0 {
		return agents, nil
	}

	for _, agentID := range agentIDs {
		agent, err := store.AgentRepo().FindByID(ctx, agentID)
		if err != nil {
			return nil, NewInvalidInputErrorf("agent with ID %s does not exist", agentID)
		}
		// Verify agent belongs to the same provider
		if agent.ProviderID != providerID {
			return nil, NewInvalidInputErrorf("agent %s does not belong to provider %s", agentID, providerID)
		}
		agents = append(agents, *agent)
	}

	return agents, nil
}

func (s *serviceActivationCommander) Create(
	ctx context.Context,
	providerID UUID,
	serviceTypeID UUID,
	tags []string,
	agentIDs []UUID,
) (*ServiceActivation, error) {
	// Validate references
	providerExists, err := s.store.ParticipantRepo().Exists(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if !providerExists {
		return nil, NewInvalidInputErrorf("provider with ID %s does not exist", providerID)
	}

	serviceTypeExists, err := s.store.ServiceTypeRepo().Exists(ctx, serviceTypeID)
	if err != nil {
		return nil, err
	}
	if !serviceTypeExists {
		return nil, NewInvalidInputErrorf("service type with ID %s does not exist", serviceTypeID)
	}

	// Create and save
	var sa *ServiceActivation
	err = s.store.Atomic(ctx, func(store Store) error {
		sa = NewServiceActivation(providerID, serviceTypeID, tags)

		agents, err := validateAndGetAgents(ctx, store, providerID, agentIDs)
		if err != nil {
			return err
		}
		sa.Agents = agents

		if err := sa.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.ServiceActivationRepo().Create(ctx, sa); err != nil {
			return err
		}

		if err := store.ServiceActivationRepo().Save(ctx, sa); err != nil {
			return err
		}

		auditEntry, err := NewEventAuditCtx(ctx, EventTypeServiceActivationCreated, JSON{"status": sa}, &sa.ID, nil, nil, &providerID)
		if err != nil {
			return err
		}
		if err := store.AuditEntryRepo().Create(ctx, auditEntry); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return sa, nil
}

func (s *serviceActivationCommander) Update(
	ctx context.Context,
	id UUID,
	tags *[]string,
	agentIDs *[]UUID,
) (*ServiceActivation, error) {
	// Find existing service activation
	sa, err := s.store.ServiceActivationRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store a copy for audit diff
	beforeSaCopy := *sa

	// Update and validate
	updated := false

	if agentIDs != nil {
		agents, err := validateAndGetAgents(ctx, s.store, sa.ProviderID, *agentIDs)
		if err != nil {
			return nil, err
		}
		sa.Agents = agents
		updated = true
	}

	if sa.Update(tags) {
		updated = true
	}

	if !updated {
		return sa, nil // No changes made
	}

	if err := sa.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and audit
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceActivationRepo().Save(ctx, sa); err != nil {
			return err
		}

		auditEntry, err := NewEventAuditCtxDiff(ctx, EventTypeServiceActivationUpdated, JSON{}, &id, nil, nil, &sa.ProviderID, &beforeSaCopy, sa)
		if err != nil {
			return err
		}
		if err := store.AuditEntryRepo().Create(ctx, auditEntry); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return sa, nil
}

func (s *serviceActivationCommander) Delete(ctx context.Context, id UUID) error {
	sa, err := s.store.ServiceActivationRepo().FindByID(ctx, id)
	if err != nil {
		return err // Handles NotFoundError as well
	}

	return s.store.Atomic(ctx, func(store Store) error {
		// TODO: Check for dependent Services when Service entity is updated to reference ServiceActivation
		// For now, we'll proceed with deletion

		if err := store.ServiceActivationRepo().Delete(ctx, id); err != nil {
			return err
		}

		auditEntry, err := NewEventAuditCtx(ctx, EventTypeServiceActivationDeleted, JSON{"status": sa}, &id, nil, nil, &sa.ProviderID)
		if err != nil {
			return err
		}
		if err := store.AuditEntryRepo().Create(ctx, auditEntry); err != nil {
			return err
		}

		return nil
	})
}
