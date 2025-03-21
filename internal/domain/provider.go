package domain

import (
	"context"
	"errors"
	"fmt"
)

// ProviderState represents the possible states of a Provider
type ProviderState string

const (
	ProviderEnabled  ProviderState = "Enabled"
	ProviderDisabled ProviderState = "Disabled"
)

// Validate checks if the provider state is valid
func (s ProviderState) Validate() error {
	switch s {
	case ProviderEnabled, ProviderDisabled:
		return nil
	default:
		return fmt.Errorf("invalid provider state: %s", s)
	}
}

func ParseProviderState(value string) (ProviderState, error) {
	state := ProviderState(value)
	if err := state.Validate(); err != nil {
		return "", err
	}
	return state, nil
}

// Provider represents a cloud service provider
type Provider struct {
	BaseEntity

	Name        string      `gorm:"not null"`
	CountryCode CountryCode `gorm:"size:2"`
	Attributes  Attributes  `gorm:"type:jsonb"`

	// State management
	State ProviderState `gorm:"not null"`

	// Relationships
	Agents []Agent `gorm:"foreignKey:ProviderID"`
}

// TableName returns the table name for the provider
func (Provider) TableName() string {
	return "providers"
}

// Validate ensures all Provider fields are valid
func (p *Provider) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}
	if err := p.CountryCode.Validate(); err != nil {
		return err
	}
	if p.Attributes != nil {
		return p.Attributes.Validate()
	}
	if err := p.State.Validate(); err != nil {
		return err
	}
	return nil
}

// ProviderCommander defines the interface for provider command operations
type ProviderCommander interface {
	// Create creates a new provider
	Create(ctx context.Context, name string, state ProviderState, countryCode CountryCode, attributes Attributes) (*Provider, error)

	// Update updates a provider
	Update(ctx context.Context, id UUID, name *string, state *ProviderState, countryCode *CountryCode, attributes *Attributes) (*Provider, error)

	// Delete removes a provider by ID after checking for dependencies
	Delete(ctx context.Context, id UUID) error
}

// providerCommander is the concrete implementation of ProviderCommander
type providerCommander struct {
	store Store
}

// NewProviderCommander creates a new ProviderService
func NewProviderCommander(
	store Store,
) *providerCommander {
	return &providerCommander{
		store: store,
	}
}

func (s *providerCommander) Create(
	ctx context.Context,
	name string,
	state ProviderState,
	countryCode CountryCode,
	attributes Attributes,
) (*Provider, error) {
	provider := &Provider{
		Name:        name,
		State:       state,
		CountryCode: countryCode,
		Attributes:  attributes,
	}
	if err := provider.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}
	if err := s.store.ProviderRepo().Create(ctx, provider); err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *providerCommander) Update(ctx context.Context,
	id UUID,
	name *string,
	state *ProviderState,
	countryCode *CountryCode,
	attributes *Attributes,
) (*Provider, error) {
	provider, err := s.store.ProviderRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != nil {
		provider.Name = *name
	}
	if state != nil {
		provider.State = *state
	}
	if countryCode != nil {
		provider.CountryCode = *countryCode
	}
	if attributes != nil {
		provider.Attributes = *attributes
	}
	if err := provider.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = s.store.ProviderRepo().Save(ctx, provider)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

func (s *providerCommander) Delete(ctx context.Context, id UUID) error {
	return s.store.Atomic(ctx, func(store Store) error {
		numOfAgents, err := s.store.AgentRepo().CountByProvider(ctx, id)
		if err != nil {
			return err
		}
		if numOfAgents > 0 {
			return errors.New("cannot delete provider with associated agents")
		}

		// Delete all tokens associated with this provider before deleting the provider
		if err := store.TokenRepo().DeleteByProviderID(ctx, id); err != nil {
			return err
		}

		return store.ProviderRepo().Delete(ctx, id)
	})
}

type ProviderRepository interface {
	ProviderQuerier

	// Create creates a new entity
	Create(ctx context.Context, entity *Provider) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *Provider) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error
}

type ProviderQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Provider, error)

	// Exists checks if an entity with the given ID exists
	Exists(ctx context.Context, id UUID) (bool, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authScope *AuthScope, req *PageRequest) (*PageResponse[Provider], error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthScope, error)
}
