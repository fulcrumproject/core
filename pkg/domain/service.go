package domain

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
)

// ServiceStatus represents the possible statuss of a service
type ServiceStatus string

const (
	EventTypeServiceCreated      EventType = "service.created"
	EventTypeServiceUpdated      EventType = "service.updated"
	EventTypeServiceTransitioned EventType = "service.transitioned"
	EventTypeServiceRetried      EventType = "service.retried"

	ServiceCreating     ServiceStatus = "Creating"
	ServiceCreated      ServiceStatus = "Created"
	ServiceStarting     ServiceStatus = "Starting"
	ServiceStarted      ServiceStatus = "Started"
	ServiceStopping     ServiceStatus = "Stopping"
	ServiceStopped      ServiceStatus = "Stopped"
	ServiceHotUpdating  ServiceStatus = "HotUpdating"
	ServiceColdUpdating ServiceStatus = "ColdUpdating"
	ServiceDeleting     ServiceStatus = "Deleting"
	ServiceDeleted      ServiceStatus = "Deleted"
)

// ParseServiceStatus parses a string into a ServiceStatus
func ParseServiceStatus(s string) (ServiceStatus, error) {
	status := ServiceStatus(s)
	if err := status.Validate(); err != nil {
		return "", err
	}
	return status, nil
}

// Validate checks if the service status is valid
func (s ServiceStatus) Validate() error {
	switch s {
	case
		ServiceCreating,
		ServiceCreated,
		ServiceStarting,
		ServiceStarted,
		ServiceStopping,
		ServiceStopped,
		ServiceHotUpdating,
		ServiceColdUpdating,
		ServiceDeleting,
		ServiceDeleted:
		return nil
	default:
		return fmt.Errorf("invalid service status: %s", s)
	}
}

// Service represents a service instance managed by an agent
type Service struct {
	BaseEntity

	Name string `json:"name" gorm:"not null"`

	// Status management
	CurrentStatus     ServiceStatus    `json:"currentStatus" gorm:"not null"`
	TargetStatus      *ServiceStatus   `json:"targetStatus,omitempty"`
	ErrorMessage      *string          `json:"errorMessage,omitempty"`
	FailedAction      *ServiceAction   `json:"failedAction,omitempty"`
	RetryCount        int              `json:"retryCount"`
	CurrentProperties *properties.JSON `json:"currentProperties,omitempty" gorm:"type:jsonb"`
	TargetProperties  *properties.JSON `json:"targetProperties,omitempty" gorm:"type:jsonb"`

	// To store an external ID for the agent's use to facilitate metric reporting
	ExternalID *string `json:"externalId,omitempty" gorm:"uniqueIndex:service_external_id_uniq"`
	// Safe place for the Agent for store data
	Resources *properties.JSON `json:"resources,omitempty" gorm:"type:jsonb"`

	// Relationships
	ProviderID    properties.UUID `json:"providerId" gorm:"not null"`
	Provider      *Participant    `json:"-" gorm:"foreignKey:ProviderID"`
	ConsumerID    properties.UUID `json:"consumerId" gorm:"not null"`
	Consumer      *Participant    `json:"-" gorm:"foreignKey:ConsumerID"`
	GroupID       properties.UUID `gorm:"not null" json:"groupId"`
	Group         *ServiceGroup   `json:"-" gorm:"foreignKey:GroupID"`
	AgentID       properties.UUID `json:"agentId" gorm:"not null"`
	Agent         *Agent          `json:"-" gorm:"foreignKey:AgentID"`
	ServiceTypeID properties.UUID `json:"serviceTypeId" gorm:"not null"`
	ServiceType   *ServiceType    `json:"-" gorm:"foreignKey:ServiceTypeID"`
}

// NewService creates a new Service without validation
func NewService(
	consumerID properties.UUID,
	groupID properties.UUID,
	providerID properties.UUID,
	agentID properties.UUID,
	serviceTypeID properties.UUID,
	name string,
	properties *properties.JSON,
) *Service {
	target := ServiceCreated
	return &Service{
		ConsumerID:       consumerID,
		GroupID:          groupID,
		ProviderID:       providerID,
		AgentID:          agentID,
		ServiceTypeID:    serviceTypeID,
		Name:             name,
		CurrentStatus:    ServiceCreating,
		TargetStatus:     &target,
		TargetProperties: properties,
	}
}

// Update updates the service
func (s *Service) Update(name *string, props *properties.JSON) (bool, *ServiceAction, error) {
	var (
		updated bool
		action  *ServiceAction
	)
	if name != nil {
		updated = true
		s.Name = *name
	}

	if props != nil && !reflect.DeepEqual(&s.CurrentProperties, *props) {
		updated = true

		transStatus, targetAction, err := serviceUpdateNextStatusAndAction(s.CurrentStatus)
		if err != nil {
			return false, nil, err
		}

		// Create and validate - use entity method
		target := s.CurrentStatus
		s.TargetProperties = props
		s.CurrentStatus = transStatus
		s.TargetStatus = &target

		action = &targetAction
	}

	return updated, action, nil
}

// serviceUpdateNextStatusAndAction determines the transition status and action when updating a service's properties based on its current status
func serviceUpdateNextStatusAndAction(status ServiceStatus) (ServiceStatus, ServiceAction, error) {
	switch status {
	case ServiceStopped:
		return ServiceColdUpdating, ServiceActionColdUpdate, nil
	case ServiceStarted:
		return ServiceHotUpdating, ServiceActionHotUpdate, nil
	default:
		return "", "", NewInvalidInputErrorf("cannot update properties on a service with status %v", status)
	}
}

// Transition sets the service statuss for a transition
func (s *Service) Transition(targetStatus ServiceStatus) (*ServiceAction, error) {
	transStatus, action, err := serviceNextStatusAndAction(s.CurrentStatus, targetStatus)
	if err != nil {
		return nil, InvalidInputError{Err: err}
	}

	s.CurrentStatus = transStatus
	s.TargetStatus = &targetStatus

	return &action, nil
}

// serviceNextStatusAndAction determines the intermediate status and action for a service transition
func serviceNextStatusAndAction(curr, target ServiceStatus) (ServiceStatus, ServiceAction, error) {
	switch curr {
	case ServiceCreated:
		if target == ServiceStarted {
			return ServiceStarting, ServiceActionStart, nil
		}
	case ServiceStarted:
		if target == ServiceStopped {
			return ServiceStopping, ServiceActionStop, nil
		}
	case ServiceStopped:
		if target == ServiceStarted {
			return ServiceStarting, ServiceActionStart, nil
		}
		if target == ServiceDeleted {
			return ServiceDeleting, ServiceActionDelete, nil
		}
	}
	return "", "", fmt.Errorf("invalid transition from %s to %s", curr, target)
}

// RetryFailedAction prepares a service for retry and returns a job for the failed action
func (s *Service) RetryFailedAction() *ServiceAction {
	if s.FailedAction == nil {
		return nil
	}

	s.RetryCount += 1

	return s.FailedAction
}

// Validate a service
func (s Service) Validate() error {
	if s.Name == "" {
		return errors.New("service name cannot be empty")
	}
	if err := s.CurrentStatus.Validate(); err != nil {
		return err
	}
	if s.TargetStatus != nil {
		if err := s.TargetStatus.Validate(); err != nil {
			return err
		}
	}
	if s.GroupID == uuid.Nil {
		return errors.New("service group ID cannot be nil")
	}
	if s.AgentID == uuid.Nil {
		return errors.New("service agent ID cannot be nil")
	}
	if s.ServiceTypeID == uuid.Nil {
		return errors.New("service type ID cannot be nil")
	}
	return nil
}

// TableName returns the table name for the service
func (Service) TableName() string {
	return "services"
}

// ServiceCommander defines the interface for service command operations
type ServiceCommander interface {
	// Create handles service creation and creates a job for the agent
	Create(ctx context.Context, agentID properties.UUID, serviceTypeID properties.UUID, groupID properties.UUID, name string, properties properties.JSON) (*Service, error)

	// CreateWithTags handles service creation using agent discovery by tags
	CreateWithTags(ctx context.Context, serviceTypeID properties.UUID, groupID properties.UUID, name string, properties properties.JSON, serviceTags []string) (*Service, error)

	// Update handles service updates and creates a job for the agent
	Update(ctx context.Context, id properties.UUID, name *string, props *properties.JSON) (*Service, error)

	// Transition transitions a service to a new status
	Transition(ctx context.Context, id properties.UUID, target ServiceStatus) (*Service, error)

	// Retry retries a failed service operation
	Retry(ctx context.Context, id properties.UUID) (*Service, error)

	// FailTimeoutServicesAndJobs fails services and jobs that have timed out
	FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error)
}

// serviceCommander is the concrete implementation of ServiceCommander
type serviceCommander struct {
	store Store
}

// NewServiceCommander creates a new commander for services
func NewServiceCommander(
	store Store,
) *serviceCommander {
	return &serviceCommander{
		store: store,
	}
}

func (s *serviceCommander) Create(
	ctx context.Context,
	agentID properties.UUID,
	serviceTypeID properties.UUID,
	groupID properties.UUID,
	name string,
	properties properties.JSON,
) (*Service, error) {
	agent, err := s.store.AgentRepo().Get(ctx, agentID)
	if err != nil {
		return nil, NewInvalidInputErrorf("agent with ID %s does not exist", agentID)
	}

	return CreateServiceWithAgent(ctx, s.store, agent, serviceTypeID, groupID, name, properties)
}

func (s *serviceCommander) CreateWithTags(
	ctx context.Context,
	serviceTypeID properties.UUID,
	groupID properties.UUID,
	name string,
	properties properties.JSON,
	serviceTags []string,
) (*Service, error) {
	return CreateServiceWithTags(ctx, s.store, serviceTypeID, groupID, name, properties, serviceTags)
}

func CreateServiceWithTags(
	ctx context.Context,
	store Store,
	serviceTypeID properties.UUID,
	groupID properties.UUID,
	name string,
	properties properties.JSON,
	serviceTags []string,
) (*Service, error) {
	agents, err := store.AgentRepo().FindByServiceTypeAndTags(ctx, serviceTypeID, serviceTags)
	if err != nil {
		return nil, err
	}

	if len(agents) == 0 {
		return nil, NewInvalidInputErrorf("no agent found for service type %s with tags %v", serviceTypeID, serviceTags)
	}

	agent := agents[0]
	return CreateServiceWithAgent(ctx, store, agent, serviceTypeID, groupID, name, properties)
}

func CreateServiceWithAgent(
	ctx context.Context,
	store Store,
	agent *Agent,
	serviceTypeID properties.UUID,
	groupID properties.UUID,
	name string,
	properties properties.JSON,
) (*Service, error) {
	group, err := store.ServiceGroupRepo().Get(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// Validate properties against schema
	validatedProperties, err := validatePropertiesAgainstSchema(ctx, store, properties, serviceTypeID)
	if err != nil {
		return nil, err
	}
	properties = validatedProperties

	// Check if the agent's type supports the requested service type
	supported := false
	for _, agentServiceType := range agent.AgentType.ServiceTypes {
		if agentServiceType.ID == serviceTypeID {
			supported = true
			break
		}
	}
	if !supported {
		return nil, NewInvalidInputErrorf("agent type %s does not support service type %s", agent.AgentType.Name, serviceTypeID)
	}

	svc := NewService(
		group.ConsumerID,
		groupID,
		agent.ProviderID,
		agent.ID,
		serviceTypeID,
		name,
		&properties,
	)
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Create(ctx, svc); err != nil {
			return err
		}

		job := NewJob(svc, ServiceActionCreate, 1)
		if err := job.Validate(); err != nil {
			return err
		}
		if err := store.JobRepo().Create(ctx, job); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeServiceCreated, WithInitiatorCtx(ctx), WithService(svc))
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

	return svc, nil
}

func (s *serviceCommander) Update(ctx context.Context, id properties.UUID, name *string, props *properties.JSON) (*Service, error) {
	return UpdateService(ctx, s.store, id, name, props)
}

func UpdateService(ctx context.Context, store Store, id properties.UUID, name *string, props *properties.JSON) (*Service, error) {
	// Find it
	svc, err := store.ServiceRepo().Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate properties against schema if provided
	if props != nil {
		validatedProperties, err := validatePropertiesAgainstSchema(ctx, store, *props, svc.ServiceTypeID)
		if err != nil {
			return nil, err
		}
		props = &validatedProperties
	}

	// Event copy
	originalSvc := *svc

	// Update
	updateSvc, action, err := svc.Update(name, props)
	if err != nil {
		return nil, err
	}
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save, event and create job
	err = store.Atomic(ctx, func(store Store) error {
		if updateSvc {
			if err := store.ServiceRepo().Save(ctx, svc); err != nil {
				return err
			}
			eventEntry, err := NewEvent(EventTypeServiceUpdated, WithInitiatorCtx(ctx), WithDiff(&originalSvc, svc), WithService(svc))
			if err != nil {
				return err
			}
			if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
				return err
			}
		}
		if action != nil {
			job := NewJob(svc, *action, 1)
			if err := job.Validate(); err != nil {
				return err
			}
			if err := store.JobRepo().Create(ctx, job); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return svc, nil
}

// validatePropertiesAgainstSchema validates properties against a service type's schema
func validatePropertiesAgainstSchema(ctx context.Context, store Store, props properties.JSON, serviceTypeID properties.UUID) (properties.JSON, error) {
	// Fetch the service type to get its schema
	serviceType, err := store.ServiceTypeRepo().Get(ctx, serviceTypeID)
	if err != nil {
		return nil, err
	}

	// If no schema, return properties as-is
	if serviceType.PropertySchema == nil {
		return props, nil
	}

	// Validate properties against schema
	propertiesMap := map[string]any(props)
	propertiesWithDefaults, validationErrors := schema.ValidateWithDefaults(propertiesMap, *serviceType.PropertySchema)

	if len(validationErrors) > 0 {
		// Convert schema validation errors to domain validation error details
		var errorDetails []ValidationErrorDetail
		for _, err := range validationErrors {
			errorDetails = append(errorDetails, ValidationErrorDetail{
				Path:    err.Path,
				Message: err.Message,
			})
		}
		return nil, NewValidationError(errorDetails)
	}

	// Return properties with defaults applied
	return properties.JSON(propertiesWithDefaults), nil
}

func (s *serviceCommander) Transition(ctx context.Context, id properties.UUID, target ServiceStatus) (*Service, error) {
	return TransitionService(ctx, s.store, id, target)
}

func TransitionService(ctx context.Context, store Store, id properties.UUID, target ServiceStatus) (*Service, error) {
	// Find it
	svc, err := store.ServiceRepo().Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Event copy
	originalSvc := *svc

	// Transition
	action, err := svc.Transition(target)
	if err != nil {
		return nil, err
	}
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save, event and create job if needed
	err = store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}
		job := NewJob(svc, *action, 1)
		if err := job.Validate(); err != nil {
			return err
		}
		if err := store.JobRepo().Create(ctx, job); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeServiceTransitioned, WithInitiatorCtx(ctx), WithDiff(&originalSvc, svc), WithService(svc))
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

	return svc, nil
}
func (s *serviceCommander) Retry(ctx context.Context, id properties.UUID) (*Service, error) {
	return RetryService(ctx, s.store, id)
}

func RetryService(ctx context.Context, store Store, id properties.UUID) (*Service, error) {
	// Find it
	svc, err := store.ServiceRepo().Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Event copy
	originalSvc := *svc

	// Retry
	action := svc.RetryFailedAction()
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}
	if action == nil {
		return svc, nil // Nothing to retry
	}

	// Save, event and create job if needed
	err = store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Save(ctx, svc); err != nil {
			return err
		}
		job := NewJob(svc, *action, 1)
		if err := job.Validate(); err != nil {
			return err
		}
		if err := store.JobRepo().Create(ctx, job); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeServiceRetried, WithInitiatorCtx(ctx), WithDiff(&originalSvc, svc), WithService(svc))
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

	return svc, nil
}

func (s *serviceCommander) FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error) {
	timedOutJobs, err := s.store.JobRepo().GetTimeOutJobs(ctx, timeout)
	if err != nil {
		return 0, fmt.Errorf("failed to retrive timeout jobs: %v", err)
	}

	counter := 0
	errorMsg := "Job marked as failed due to exceeding maximum processing time"
	for _, job := range timedOutJobs {
		err := s.store.Atomic(ctx, func(s Store) error {
			// Update job
			job.Status = JobFailed
			job.ErrorMessage = errorMsg
			now := time.Now()
			job.CompletedAt = &now

			// Update associated service status
			svc, err := s.ServiceRepo().Get(ctx, job.ServiceID)
			if err != nil {
				return fmt.Errorf("failed to find service %s for timed out job: %w", job.ServiceID, err)
			}

			svc.ErrorMessage = &errorMsg
			svc.FailedAction = &job.Action

			if err := s.JobRepo().Save(ctx, job); err != nil {
				return err
			}
			return s.ServiceRepo().Save(ctx, svc)
		})
		counter++
		if err != nil {
			return 0, fmt.Errorf("Error marking timed out job %s as failed: %v", job.ID, err)
		}
	}

	return counter, nil
}

// HandleJobComplete updates the service status when a job completes successfully
func (s *Service) HandleJobComplete(resources *properties.JSON, externalID *string) error {
	if s.TargetStatus == nil {
		return InvalidInputError{Err: errors.New("cannot complete a job on service that is not in transition")}
	}

	s.CurrentStatus = *s.TargetStatus
	s.TargetStatus = nil
	s.FailedAction = nil
	s.ErrorMessage = nil
	s.RetryCount = 0

	if resources != nil {
		s.Resources = resources
	}
	if externalID != nil {
		s.ExternalID = externalID
	}
	if s.TargetProperties != nil {
		s.CurrentProperties = s.TargetProperties
		s.TargetProperties = nil
	}

	return nil
}

// HandleJobFailure updates the service status when a job fails
func (s *Service) HandleJobFailure(errorMessage string, action ServiceAction) {
	s.ErrorMessage = &errorMessage
	s.FailedAction = &action
}

// ServiceRepository defines the interface for the Service repository
type ServiceRepository interface {
	ServiceQuerier
	BaseEntityRepository[Service]
}

// ServiceQuerier defines the interface for the Service read-only queries
type ServiceQuerier interface {
	BaseEntityQuerier[Service]

	// FindByExternalID retrieves a service by its external ID and agent ID
	FindByExternalID(ctx context.Context, agentID properties.UUID, externalID string) (*Service, error)

	// CountByGroup returns the number of services in a specific group
	CountByGroup(ctx context.Context, groupID properties.UUID) (int64, error)

	// CountByAgent returns the number of services handled by a specific agent
	CountByAgent(ctx context.Context, agentID properties.UUID) (int64, error)
}
