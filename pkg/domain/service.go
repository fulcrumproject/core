package domain

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
)

// Event types
const (
	EventTypeServiceCreated      EventType = "service.created"
	EventTypeServiceUpdated      EventType = "service.updated"
	EventTypeServiceTransitioned EventType = "service.transitioned"
	EventTypeServiceRetried      EventType = "service.retried"
)

// ServiceStatus represents the possible statuss of a service
type ServiceStatus string

const (
	ServiceNew     ServiceStatus = "New"
	ServiceStarted ServiceStatus = "Started"
	ServiceStopped ServiceStatus = "Stopped"
	ServiceDeleted ServiceStatus = "Deleted"
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
		ServiceNew,
		ServiceStopped,
		ServiceStarted,
		ServiceDeleted:
		return nil
	default:
		return fmt.Errorf("invalid service status: %s", s)
	}
}

// ServiceAction represents the type of operation a job performs
type ServiceAction string

const (
	ServiceActionCreate ServiceAction = "Create"
	ServiceActionStart  ServiceAction = "Start"
	ServiceActionStop   ServiceAction = "Stop"
	ServiceActionUpdate ServiceAction = "Update"
	ServiceActionDelete ServiceAction = "Delete"
)

// ParseServiceAction parses a string into a JobType
func ParseServiceAction(s string) (ServiceAction, error) {
	jobType := ServiceAction(s)
	if err := jobType.Validate(); err != nil {
		return "", err
	}
	return jobType, nil
}

// Validate checks if the job type is valid
func (t ServiceAction) Validate() error {
	switch t {
	case
		ServiceActionCreate,
		ServiceActionStart,
		ServiceActionStop,
		ServiceActionUpdate,
		ServiceActionDelete:
		return nil
	}
	return fmt.Errorf("invalid job type: %s", t)
}

// currentStatusActionNextStatus maps the current status to the next status for a given action
// it'll be included in the service type schema
var currentStatusActionNextStatus = map[ServiceStatus]map[ServiceAction]ServiceStatus{
	ServiceNew: {
		ServiceActionCreate: ServiceStopped,
	},
	ServiceStopped: {
		ServiceActionStart:  ServiceStarted,
		ServiceActionUpdate: ServiceStopped,
		ServiceActionDelete: ServiceDeleted,
	},
	ServiceStarted: {
		ServiceActionStop:   ServiceStopped,
		ServiceActionUpdate: ServiceStarted,
	},
}

// serviceNextStatus determines the intermediate status and action for a service transition
func serviceNextStatus(curr ServiceStatus, action ServiceAction) (ServiceStatus, error) {
	next, ok := currentStatusActionNextStatus[curr][action]
	if !ok {
		return "", fmt.Errorf("invalid action %s on service with status %s", action, curr)
	}
	return next, nil
}

// Service represents a service instance managed by an agent
type Service struct {
	BaseEntity

	Name       string           `json:"name" gorm:"not null"`
	Status     ServiceStatus    `json:"status" gorm:"not null"`
	Properties *properties.JSON `json:"properties,omitempty" gorm:"type:jsonb"`

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
	agent *Agent,
	group *ServiceGroup,
	params CreateServiceParams,
) *Service {
	return &Service{
		ConsumerID:    group.ConsumerID,
		GroupID:       group.ID,
		ProviderID:    agent.ProviderID,
		AgentID:       agent.ID,
		ServiceTypeID: params.ServiceTypeID,
		Name:          params.Name,
		Status:        ServiceNew,
		Properties:    &params.Properties,
	}
}

// HandleJobComplete handles the completion of a job
func (s *Service) HandleJobComplete(action ServiceAction, params *properties.JSON, resources *properties.JSON, externalID *string) error {
	// Update status
	nextStatus, err := serviceNextStatus(s.Status, action)
	if err != nil {
		return err
	}
	s.Status = nextStatus

	// Update resources and external ID if provided
	if resources != nil {
		s.Resources = resources
	}
	if externalID != nil {
		s.ExternalID = externalID
	}

	// Update properties if the action is an update
	if action == ServiceActionUpdate {
		s.Properties = params
	}

	return nil
}

// Update updates the service
func (s *Service) Update(name *string, properties *properties.JSON) (update bool, action bool, err error) {
	if name != nil {
		s.Name = *name
		update = true
	}

	if properties != nil {
		action = true
	}

	return update, action, nil
}

// ApplyAgentPropertyUpdates applies property updates from an agent
func (s *Service) ApplyAgentPropertyUpdates(
	serviceType *ServiceType,
	updates map[string]any,
) error {
	if len(updates) == 0 {
		return nil
	}

	// Validate that agent can only update agent-source properties
	if err := ValidatePropertiesForUpdate(
		updates,
		string(s.Status),
		serviceType.PropertySchema,
		"agent",
	); err != nil {
		return fmt.Errorf("invalid agent property updates: %w", err)
	}

	// Apply updates
	if s.Properties == nil {
		props := make(properties.JSON)
		s.Properties = &props
	}
	for k, v := range updates {
		(*s.Properties)[k] = v
	}

	return nil
}

// Validate a service
func (s *Service) Validate() error {
	if s.Name == "" {
		return errors.New("service name cannot be empty")
	}
	if err := s.Status.Validate(); err != nil {
		return err
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
	Create(ctx context.Context, params CreateServiceParams) (*Service, error)

	// CreateWithTags handles service creation using agent discovery by tags
	CreateWithTags(ctx context.Context, params CreateServiceWithTagsParams) (*Service, error)

	// Update handles service updates and creates a job for the agent
	Update(ctx context.Context, params UpdateServiceParams) (*Service, error)

	// DoAction handles service actions
	DoAction(ctx context.Context, params DoServiceActionParams) (*Service, error)

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

type CreateServiceParams struct {
	AgentID       properties.UUID `json:"agentId"`
	ServiceTypeID properties.UUID `json:"serviceTypeId"`
	GroupID       properties.UUID `json:"groupId"`
	Name          string          `json:"name"`
	Properties    properties.JSON `json:"targetProperties"`
}

type CreateServiceWithTagsParams struct {
	CreateServiceParams
	ServiceTags []string `json:"agentTags,omitempty"`
}

type UpdateServiceParams struct {
	ID         properties.UUID  `json:"id"`
	Name       *string          `json:"name,omitempty"`
	Properties *properties.JSON `json:"properties,omitempty"`
}

type DoServiceActionParams struct {
	ID     properties.UUID `json:"id"`
	Action ServiceAction   `json:"action"`
}

func (s *serviceCommander) Create(
	ctx context.Context,
	params CreateServiceParams,
) (*Service, error) {
	agent, err := s.store.AgentRepo().Get(ctx, params.AgentID)
	if err != nil {
		return nil, NewInvalidInputErrorf("agent with ID %s does not exist", params.AgentID)
	}

	return CreateServiceWithAgent(ctx, s.store, agent, params)
}

func (s *serviceCommander) CreateWithTags(
	ctx context.Context,
	params CreateServiceWithTagsParams,
) (*Service, error) {
	return CreateServiceWithTags(ctx, s.store, params)
}

func CreateServiceWithTags(
	ctx context.Context,
	store Store,
	params CreateServiceWithTagsParams,
) (*Service, error) {
	agents, err := store.AgentRepo().FindByServiceTypeAndTags(ctx, params.ServiceTypeID, params.ServiceTags)
	if err != nil {
		return nil, err
	}

	if len(agents) == 0 {
		return nil, NewInvalidInputErrorf("no agent found for service type %s with tags %v", params.ServiceTypeID, params.ServiceTags)
	}

	agent := agents[0]
	return CreateServiceWithAgent(ctx, store, agent, params.CreateServiceParams)
}

func CreateServiceWithAgent(
	ctx context.Context,
	store Store,
	agent *Agent,
	params CreateServiceParams,
) (*Service, error) {
	group, err := store.ServiceGroupRepo().Get(ctx, params.GroupID)
	if err != nil {
		return nil, err
	}

	// Load ServiceType to get property schema
	serviceType, err := store.ServiceTypeRepo().Get(ctx, params.ServiceTypeID)
	if err != nil {
		return nil, err
	}

	// Validate property source (user cannot set agent-source properties during creation)
	if err := ValidatePropertiesForCreation(params.Properties, serviceType.PropertySchema, "user"); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Validate properties against schema
	validationParams := &ServicePropertyValidationParams{
		ServiceTypeID: params.ServiceTypeID,
		GroupID:       params.GroupID,
		Properties:    params.Properties,
	}
	validatedProperties, err := ValidateServiceProperties(ctx, store, validationParams)
	if err != nil {
		return nil, err
	}
	params.Properties = validatedProperties

	// Check if the agent's type supports the requested service type
	supported := false
	for _, agentServiceType := range agent.AgentType.ServiceTypes {
		if agentServiceType.ID == params.ServiceTypeID {
			supported = true
			break
		}
	}
	if !supported {
		return nil, NewInvalidInputErrorf("agent type %s does not support service type %s", agent.AgentType.Name, params.ServiceTypeID)
	}

	svc := NewService(
		agent,
		group,
		params,
	)
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceRepo().Create(ctx, svc); err != nil {
			return err
		}

		job := NewJob(svc, ServiceActionCreate, &params.Properties, 1)
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

func (s *serviceCommander) Update(ctx context.Context, params UpdateServiceParams) (*Service, error) {
	return UpdateService(ctx, s.store, params)
}

func UpdateService(ctx context.Context, store Store, params UpdateServiceParams) (*Service, error) {
	// Find it
	svc, err := store.ServiceRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Merge and validate properties if provided
	if params.Properties != nil {
		// Load ServiceType to get property schema
		serviceType, err := store.ServiceTypeRepo().Get(ctx, svc.ServiceTypeID)
		if err != nil {
			return nil, err
		}

		// Validate property source and updatability
		if err := ValidatePropertiesForUpdate(*params.Properties, string(svc.Status), serviceType.PropertySchema, "user"); err != nil {
			return nil, InvalidInputError{Err: err}
		}

		// Merge partial properties with existing properties
		mergedProperties := mergeServiceProperties(svc.Properties, *params.Properties)

		// Validate merged properties against schema
		validationParams := &ServicePropertyValidationParams{
			ServiceTypeID: svc.ServiceTypeID,
			GroupID:       svc.GroupID,
			Properties:    mergedProperties,
		}
		validatedProperties, err := ValidateServiceProperties(ctx, store, validationParams)
		if err != nil {
			return nil, err
		}
		convertedProperties := properties.JSON(validatedProperties)
		params.Properties = &convertedProperties
	}

	// Update, if needed
	originalSvc := *svc
	update, action, err := svc.Update(params.Name, params.Properties)
	if err != nil {
		return nil, err
	}
	if err := svc.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save, event and create job
	err = store.Atomic(ctx, func(store Store) error {
		if update {
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
		if action {
			// Check if the service is in a valid state to be updated with a job
			if _, err := serviceNextStatus(svc.Status, ServiceActionUpdate); err != nil {
				return err
			}

			// If pending job exists, fail it
			err = checkHasNotActiveJob(ctx, store, svc)
			if err != nil {
				return err
			}

			// Create new job
			job := NewJob(svc, ServiceActionUpdate, params.Properties, 1)
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

func (s *serviceCommander) DoAction(ctx context.Context, params DoServiceActionParams) (*Service, error) {
	return DoServiceAction(ctx, s.store, params)
}

func DoServiceAction(ctx context.Context, store Store, params DoServiceActionParams) (*Service, error) {
	// Find it
	svc, err := store.ServiceRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Check if the service is in a valid state to be updated with a job
	if _, err := serviceNextStatus(svc.Status, params.Action); err != nil {
		return nil, err
	}

	// If pending job exists, fail it
	err = checkHasNotActiveJob(ctx, store, svc)
	if err != nil {
		return nil, err
	}

	// Create the new job
	err = store.Atomic(ctx, func(store Store) error {
		job := NewJob(svc, params.Action, nil, 1)
		if err := job.Validate(); err != nil {
			return err
		}
		if err := store.JobRepo().Create(ctx, job); err != nil {
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
	// Check if the service exists
	svc, err := store.ServiceRepo().Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get last job and check if it's failed
	job, err := store.JobRepo().GetLastJobForService(ctx, svc.ID)
	if err != nil {
		return nil, err
	}
	if job == nil || job.Status != JobFailed {
		return nil, NewInvalidInputErrorf("no failed job found for service %s", svc.ID)
	}

	// Create the new job as a copy of the failed one
	err = store.Atomic(ctx, func(store Store) error {
		job := NewJob(svc, job.Action, job.Params, 1)
		if err := job.Validate(); err != nil {
			return err
		}
		if err := store.JobRepo().Create(ctx, job); err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	return svc, nil
}

func checkHasNotActiveJob(ctx context.Context, store Store, svc *Service) error {
	job, err := store.JobRepo().GetLastJobForService(ctx, svc.ID)
	if err != nil {
		return err
	}
	if job != nil && job.IsActive() {
		return NewInvalidInputErrorf("cannot update service %s while there is an active job %s", svc.ID, job.ID)
	}
	return nil
}

func (s *serviceCommander) FailTimeoutServicesAndJobs(ctx context.Context, timeout time.Duration) (int, error) {
	timedOutJobs, err := s.store.JobRepo().GetTimeOutJobs(ctx, timeout)
	if err != nil {
		return 0, fmt.Errorf("failed to retrive timeout jobs: %v", err)
	}

	counter := 0
	errorMsg := "Job marked as failed due to exceeding maximum processing time"
	for _, job := range timedOutJobs {
		// Update job to failed
		job.Status = JobFailed
		job.ErrorMessage = errorMsg
		now := time.Now()
		job.CompletedAt = &now
		if err := s.store.JobRepo().Save(ctx, job); err != nil {
			return counter, err
		}
		counter++
	}

	return counter, nil
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

	// CountByServiceType returns the number of services of a specific type
	CountByServiceType(ctx context.Context, serviceTypeID properties.UUID) (int64, error)
}

// mergeServiceProperties merges partial properties with existing properties
func mergeServiceProperties(existing *properties.JSON, partial properties.JSON) properties.JSON {
	// Start with existing properties
	merged := make(map[string]any)
	if existing != nil {
		maps.Copy(merged, *existing)
	}

	// Overlay partial properties with deep merge for objects
	for k, v := range partial {
		if existingObj, exists := merged[k].(map[string]any); exists {
			if partialObj, ok := v.(map[string]any); ok {
				// Deep merge nested objects
				merged[k] = mergeNestedObjects(existingObj, partialObj)
			} else {
				// Replace with new value
				merged[k] = v
			}
		} else {
			// New key or non-object value
			merged[k] = v
		}
	}

	return properties.JSON(merged)
}

// mergeNestedObjects performs deep merge of nested objects
func mergeNestedObjects(existing, partial map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy existing values
	maps.Copy(result, existing)

	// Overlay partial values
	for k, v := range partial {
		if existingObj, exists := result[k].(map[string]any); exists {
			if partialObj, ok := v.(map[string]any); ok {
				// Recursively merge nested objects
				result[k] = mergeNestedObjects(existingObj, partialObj)
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}
