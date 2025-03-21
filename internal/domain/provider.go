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

	Name        string      `json:"name" gorm:"not null"`
	CountryCode CountryCode `json:"countryCode,omitempty" gorm:"size:2"`
	Attributes  Attributes  `json:"attributes,omitempty" gorm:"type:jsonb"`

	// State management
	State ProviderState `json:"state" gorm:"not null"`

	// Relationships
	Agents []Agent `json:"agents,omitempty" gorm:"foreignKey:ProviderID"`
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
	store          Store
	auditCommander AuditEntryCommander
}

// NewProviderCommander creates a new ProviderService
func NewProviderCommander(
	store Store,
	auditCommander AuditEntryCommander,
) *providerCommander {
	return &providerCommander{
		store:          store,
		auditCommander: auditCommander,
	}
}

func (s *providerCommander) Create(
	ctx context.Context,
	name string,
	state ProviderState,
	countryCode CountryCode,
	attributes Attributes,
) (*Provider, error) {
	var provider *Provider
	err := s.store.Atomic(ctx, func(store Store) error {
		provider = &Provider{
			Name:        name,
			State:       state,
			CountryCode: countryCode,
			Attributes:  attributes,
		}
		if err := provider.Validate(); err != nil {
			return InvalidInputError{Err: err}
		}

		if err := store.ProviderRepo().Create(ctx, provider); err != nil {
			return err
		}

		_, err := s.auditCommander.CreateCtx(
			ctx,
			EventTypeProviderCreated,
			JSON{"state": provider},
			&provider.ID, &provider.ID, nil, nil)
		return err
	})
	if err != nil {
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
	beforeProvider, err := s.store.ProviderRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store a copy of the provider before modifications for audit diff
	beforeProviderCopy := *beforeProvider

	if name != nil {
		beforeProvider.Name = *name
	}
	if state != nil {
		beforeProvider.State = *state
	}
	if countryCode != nil {
		beforeProvider.CountryCode = *countryCode
	}
	if attributes != nil {
		beforeProvider.Attributes = *attributes
	}
	if err := beforeProvider.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	err = s.store.Atomic(ctx, func(store Store) error {
		err := store.ProviderRepo().Save(ctx, beforeProvider)
		if err != nil {
			return err
		}

		_, err = s.auditCommander.CreateCtxWithDiff(
			ctx,
			EventTypeProviderUpdated,
			&id, &id, nil, nil,
			&beforeProviderCopy, beforeProvider)
		return err
	})
	if err != nil {
		return nil, err
	}
	return beforeProvider, nil
}

func (s *providerCommander) Delete(ctx context.Context, id UUID) error {
	// Get provider before deletion for audit purposes
	provider, err := s.store.ProviderRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}

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

		if err := store.ProviderRepo().Delete(ctx, id); err != nil {
			return err
		}

		_, err = s.auditCommander.CreateCtx(
			ctx,
			EventTypeProviderDeleted,
			JSON{"state": provider}, &id, &id, nil, nil)
		return err
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
