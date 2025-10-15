package domain

import (
	"context"
	"fmt"
	"regexp"

	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeServiceTypeCreated EventType = "service_type.created"
	EventTypeServiceTypeUpdated EventType = "service_type.updated"
	EventTypeServiceTypeDeleted EventType = "service_type.deleted"
)

// LifecycleSchema defines the state machine for a service type
type LifecycleSchema struct {
	States         []LifecycleState  `json:"states"`
	Actions        []LifecycleAction `json:"actions"`
	InitialState   string            `json:"initialState"`
	TerminalStates []string          `json:"terminalStates"`
}

// LifecycleState represents a state in the service lifecycle
type LifecycleState struct {
	Name string `json:"name"`
}

// LifecycleAction represents an action that can be performed on a service
type LifecycleAction struct {
	Name              string                `json:"name"`
	RequestSchemaType string                `json:"requestSchemaType,omitempty"`
	Transitions       []LifecycleTransition `json:"transitions"`
}

// LifecycleTransition represents a state transition triggered by an action
type LifecycleTransition struct {
	From          string `json:"from"`
	To            string `json:"to"`
	OnError       bool   `json:"onError,omitempty"`
	OnErrorRegexp string `json:"onErrorRegexp,omitempty"`
}

// ServiceType represents a type of service that can be provided
type ServiceType struct {
	BaseEntity
	Name            string           `json:"name" gorm:"not null;unique"`
	PropertySchema  *ServiceSchema   `json:"propertySchema,omitempty" gorm:"type:jsonb"`
	LifecycleSchema *LifecycleSchema `json:"lifecycleSchema,omitempty" gorm:"type:jsonb"`
}

// NewServiceType creates a new service type without validation
func NewServiceType(params CreateServiceTypeParams) *ServiceType {
	return &ServiceType{
		Name:            params.Name,
		PropertySchema:  params.PropertySchema,
		LifecycleSchema: params.LifecycleSchema,
	}
}

// TableName returns the table name for the service type
func (ServiceType) TableName() string {
	return "service_types"
}

// Validate ensures all ServiceType fields are valid
func (st *ServiceType) Validate() error {
	if st.Name == "" {
		return fmt.Errorf("service type name cannot be empty")
	}
	if st.LifecycleSchema != nil {
		if err := st.ValidateLifecycle(); err != nil {
			return err
		}
	}
	return nil
}

// ValidateLifecycle validates the lifecycle schema structure and rules
func (st *ServiceType) ValidateLifecycle() error {
	if st.LifecycleSchema == nil {
		return nil
	}

	lc := st.LifecycleSchema

	// Validate we have at least one state
	if len(lc.States) == 0 {
		return fmt.Errorf("lifecycle must have at least one state")
	}

	// Build a set of valid state names for quick lookup
	stateNames := make(map[string]bool)
	for _, state := range lc.States {
		if state.Name == "" {
			return fmt.Errorf("lifecycle state name cannot be empty")
		}
		if stateNames[state.Name] {
			return fmt.Errorf("duplicate lifecycle state name: %s", state.Name)
		}
		stateNames[state.Name] = true
	}

	// Validate initial state exists
	if lc.InitialState == "" {
		return fmt.Errorf("lifecycle must have an initial state")
	}
	if !stateNames[lc.InitialState] {
		return fmt.Errorf("lifecycle initial state %q does not exist in states list", lc.InitialState)
	}

	// Validate terminal states exist
	for _, terminalState := range lc.TerminalStates {
		if !stateNames[terminalState] {
			return fmt.Errorf("lifecycle terminal state %q does not exist in states list", terminalState)
		}
	}

	// Validate actions
	if len(lc.Actions) == 0 {
		return fmt.Errorf("lifecycle must have at least one action")
	}

	actionNames := make(map[string]bool)
	for _, action := range lc.Actions {
		if action.Name == "" {
			return fmt.Errorf("lifecycle action name cannot be empty")
		}
		if actionNames[action.Name] {
			return fmt.Errorf("duplicate lifecycle action name: %s", action.Name)
		}
		actionNames[action.Name] = true

		// Validate action has at least one transition
		if len(action.Transitions) == 0 {
			return fmt.Errorf("lifecycle action %q must have at least one transition", action.Name)
		}

		// Validate transitions
		for _, transition := range action.Transitions {
			if !stateNames[transition.From] {
				return fmt.Errorf("lifecycle action %q transition references invalid from state %q", action.Name, transition.From)
			}
			if !stateNames[transition.To] {
				return fmt.Errorf("lifecycle action %q transition references invalid to state %q", action.Name, transition.To)
			}

			// Validate error regexp if provided
			if transition.OnErrorRegexp != "" {
				if _, err := regexp.Compile(transition.OnErrorRegexp); err != nil {
					return fmt.Errorf("lifecycle action %q transition has invalid error regexp %q: %w", action.Name, transition.OnErrorRegexp, err)
				}
			}
		}
	}

	return nil
}

// Update updates the service type fields if the pointers are non-nil
func (st *ServiceType) Update(params UpdateServiceTypeParams) {
	if params.Name != nil {
		st.Name = *params.Name
	}
	if params.PropertySchema != nil {
		st.PropertySchema = params.PropertySchema
	}
	if params.LifecycleSchema != nil {
		st.LifecycleSchema = params.LifecycleSchema
	}
}

// ServiceTypeRepository defines the interface for the ServiceType repository
type ServiceTypeRepository interface {
	ServiceTypeQuerier
	BaseEntityRepository[ServiceType]
}

// ServiceTypeQuerier defines the interface for the ServiceType read-only queries
type ServiceTypeQuerier interface {
	BaseEntityQuerier[ServiceType]
}

// ServiceTypeCommander defines the interface for the ServiceType commands
type ServiceTypeCommander interface {
	// Create creates a new service type
	Create(ctx context.Context, params CreateServiceTypeParams) (*ServiceType, error)

	// Update updates a service type
	Update(ctx context.Context, params UpdateServiceTypeParams) (*ServiceType, error)

	// Delete removes a service type by ID after checking for dependencies
	Delete(ctx context.Context, id properties.UUID) error

	// ValidateServiceProperties validates properties against a service type's schema
	ValidateServiceProperties(ctx context.Context, params *ServicePropertyValidationParams) (map[string]any, error)
}

// ServicePropertyValidationParams provides the parameters for validating service properties
type ServicePropertyValidationParams struct {
	ServiceTypeID properties.UUID
	GroupID       properties.UUID
	Properties    map[string]any
}

type CreateServiceTypeParams struct {
	Name            string           `json:"name"`
	PropertySchema  *ServiceSchema   `json:"propertySchema,omitempty"`
	LifecycleSchema *LifecycleSchema `json:"lifecycleSchema,omitempty"`
}

type UpdateServiceTypeParams struct {
	ID              properties.UUID  `json:"id"`
	Name            *string          `json:"name"`
	PropertySchema  *ServiceSchema   `json:"propertySchema,omitempty"`
	LifecycleSchema *LifecycleSchema `json:"lifecycleSchema,omitempty"`
}

// serviceTypeCommander is the concrete implementation of ServiceTypeCommander
type serviceTypeCommander struct {
	store Store
}

// NewServiceTypeCommander creates a new ServiceTypeCommander
func NewServiceTypeCommander(store Store) ServiceTypeCommander {
	return &serviceTypeCommander{store: store}
}

// Create creates a new service type
func (c *serviceTypeCommander) Create(
	ctx context.Context,
	params CreateServiceTypeParams,
) (*ServiceType, error) {
	var serviceType *ServiceType
	err := c.store.Atomic(ctx, func(store Store) error {
		serviceType = NewServiceType(params)
		if err := serviceType.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.ServiceTypeRepo().Create(ctx, serviceType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeServiceTypeCreated, WithInitiatorCtx(ctx), WithServiceType(serviceType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return serviceType, nil
}

// Update updates a service type
func (c *serviceTypeCommander) Update(
	ctx context.Context,
	params UpdateServiceTypeParams,
) (*ServiceType, error) {
	serviceType, err := c.store.ServiceTypeRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Store a copy of the service type before modifications for event diff
	beforeServiceType := *serviceType

	// Update and validate
	serviceType.Update(params)
	if err := serviceType.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	// Save and event
	err = c.store.Atomic(ctx, func(store Store) error {
		if err := store.ServiceTypeRepo().Save(ctx, serviceType); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeServiceTypeUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeServiceType, serviceType), WithServiceType(serviceType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return serviceType, nil
}

// Delete removes a service type by ID after checking for dependencies
func (c *serviceTypeCommander) Delete(ctx context.Context, id properties.UUID) error {
	serviceType, err := c.store.ServiceTypeRepo().Get(ctx, id)
	if err != nil {
		return err
	}

	return c.store.Atomic(ctx, func(store Store) error {
		// Check for dependent Services
		serviceCount, err := store.ServiceRepo().CountByServiceType(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to count services for service type %s: %w", id, err)
		}
		if serviceCount > 0 {
			return NewInvalidInputErrorf("cannot delete service type %s: %d dependent service(s) exist", id, serviceCount)
		}

		eventEntry, err := NewEvent(EventTypeServiceTypeDeleted, WithInitiatorCtx(ctx), WithServiceType(serviceType))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}

		if err := store.ServiceTypeRepo().Delete(ctx, id); err != nil {
			return err
		}

		return nil
	})
}

// ValidateServiceProperties validates the properties against the service type schema
func (c *serviceTypeCommander) ValidateServiceProperties(ctx context.Context, params *ServicePropertyValidationParams) (map[string]any, error) {
	return ValidateServiceProperties(ctx, c.store, params)
}

// ValidateProperties validates the properties against the service type schema
func ValidateServiceProperties(ctx context.Context, store Store, params *ServicePropertyValidationParams) (map[string]any, error) {
	// Fetch the service type to get its schema
	serviceType, err := store.ServiceTypeRepo().Get(ctx, params.ServiceTypeID)
	if err != nil {
		return nil, err
	}

	// If no schema, return properties as-is
	if serviceType.PropertySchema == nil {
		return params.Properties, nil
	}

	// Apply defaults to properties
	propertiesWithDefaults := applyServicePropertiesDefaults(params.Properties, *serviceType.PropertySchema)

	// Create validation context
	validationCtx := &ServicePropertyValidationCtx{
		Context:    ctx,
		Store:      store,
		Schema:     *serviceType.PropertySchema,
		GroupID:    params.GroupID,
		Properties: propertiesWithDefaults,
	}

	// Validate properties against schema
	validationErrors, err := validateServiceProperties(validationCtx)
	if err != nil {
		return nil, err
	}
	if len(validationErrors) > 0 {
		return nil, NewValidationError(validationErrors)
	}

	return propertiesWithDefaults, nil
}
