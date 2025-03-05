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
	Name        string        `gorm:"not null"`
	State       ProviderState `gorm:"not null"`
	CountryCode CountryCode   `gorm:"size:2"`
	Attributes  Attributes    `gorm:"type:jsonb"`
	Agents      []Agent       `gorm:"foreignKey:ProviderID"`
}

// TableName returns the table name for the provider
func (*Provider) TableName() string {
	return "providers"
}

// Validate ensures all Provider fields are valid
func (p *Provider) Validate() error {
	if err := p.State.Validate(); err != nil {
		return err
	}
	if err := p.CountryCode.Validate(); err != nil {
		return err
	}
	if p.Attributes != nil {
		return p.Attributes.Validate()
	}
	return nil
}

// ProviderCommander handles provider operations with validation
type ProviderCommander struct {
	repo      ProviderRepository
	agentRepo AgentRepository
}

// NewProviderCommander creates a new ProviderService
func NewProviderCommander(
	repo ProviderRepository,
	agentRepo AgentRepository,
) *ProviderCommander {
	return &ProviderCommander{
		repo:      repo,
		agentRepo: agentRepo,
	}
}

// Create creates a new provider with validation
func (s *ProviderCommander) Create(
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
		return nil, err
	}
	if err := s.repo.Create(ctx, provider); err != nil {
		return nil, err
	}
	return provider, nil
}

// Update updates a provider with validation
func (s *ProviderCommander) Update(ctx context.Context,
	id UUID,
	name *string,
	state *ProviderState,
	countryCode *CountryCode,
	attributes *Attributes,
) (*Provider, error) {
	provider, err := s.repo.FindByID(ctx, id)
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
		return nil, err
	}
	err = s.repo.Save(ctx, provider)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// Delete removes a provider by ID after checking for dependencies
func (s *ProviderCommander) Delete(ctx context.Context, id UUID) error {
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	// Check if provider has agents
	numOfAgents, err := s.agentRepo.CountByProvider(ctx, id)
	if err != nil {
		return err
	}
	if numOfAgents > 0 {
		return errors.New("cannot delete provider with associated agents")
	}

	return s.repo.Delete(ctx, id)
}

type ProviderRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *Provider) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *Provider) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Provider, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[Provider], error)
}

type ProviderQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Provider, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[Provider], error)
}
