package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

// AgentState represents the possible states of an Agent
type AgentState string

const (
	AgentNew          AgentState = "New"
	AgentConnected    AgentState = "Connected"
	AgentDisconnected AgentState = "Disconnected"
	AgentError        AgentState = "Error"
	AgentDisabled     AgentState = "Disabled"
)

// Validate checks if the agent state is valid
func (s AgentState) Validate() error {
	switch s {
	case AgentNew, AgentConnected, AgentDisconnected, AgentError, AgentDisabled:
		return nil
	default:
		return fmt.Errorf("invalid agent state: %s", s)
	}
}

func ParseAgentState(value string) (AgentState, error) {
	state := AgentState(value)
	if err := state.Validate(); err != nil {
		return "", err
	}
	return state, nil
}

// Agent represents a service manager agent
type Agent struct {
	BaseEntity
	Name             string      `gorm:"not null"`
	State            AgentState  `gorm:"not null"`
	TokenHash        string      `gorm:"not null"`
	Token            string      `gorm:"-"`
	CountryCode      CountryCode `gorm:"size:2"`
	Attributes       Attributes  `gorm:"type:jsonb"`
	ProviderID       UUID        `gorm:"not null"`
	AgentTypeID      UUID        `gorm:"not null"`
	Provider         *Provider   `gorm:"foreignKey:ProviderID"`
	AgentType        *AgentType  `gorm:"foreignKey:AgentTypeID"`
	LastStatusUpdate time.Time   `gorm:"index"`
}

// TableName returns the table name for the agent
func (*Agent) TableName() string {
	return "agents"
}

// Validate ensures all agent fields are valid
func (a *Agent) Validate() error {
	if err := a.State.Validate(); err != nil {
		return err
	}
	return nil
}

// GenerateToken creates a secure random token and sets the TokenHash field
func (a *Agent) GenerateToken() (string, error) {
	// Generate a secure random token (32 bytes = 256 bits)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Convert to base64 for readability
	a.Token = base64.URLEncoding.EncodeToString(tokenBytes)

	// Store only the hash of the token
	a.TokenHash = HashToken(a.Token)

	return a.Token, nil
}

// VerifyToken checks if a token matches the stored hash
func (a *Agent) VerifyToken(token string) bool {
	return a.TokenHash == HashToken(token)
}

// HashToken creates a secure hash of a token
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// AgentCommander handles agent operations with validation
type AgentCommander struct {
	store Store
}

// NewAgentCommander creates a new AgentService
func NewAgentCommander(
	store Store,
) *AgentCommander {
	return &AgentCommander{
		store: store,
	}
}

// Create creates a new agent with validation
func (s *AgentCommander) Create(
	ctx context.Context,
	name string,
	countryCode CountryCode,
	attributes Attributes,
	providerID UUID,
	agentTypeID UUID,
) (*Agent, error) {
	agent := &Agent{
		Name:        name,
		State:       AgentDisconnected,
		CountryCode: countryCode,
		Attributes:  attributes,
		ProviderID:  providerID,
		AgentTypeID: agentTypeID,
	}
	_, err := agent.GenerateToken()
	if err != nil {
		return nil, err
	}
	if err := agent.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.AgentRepo().Create(ctx, agent); err != nil {
		return nil, err
	}
	return agent, nil
}

// Update updates a agent with validation
func (s *AgentCommander) Update(ctx context.Context,
	id UUID,
	name *string,
	countryCode *CountryCode,
	attributes *Attributes,
	state *AgentState,
) (*Agent, error) {
	agent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != nil {
		agent.Name = *name
	}
	if countryCode != nil {
		agent.CountryCode = *countryCode
	}
	if attributes != nil {
		agent.Attributes = *attributes
	}
	if state != nil {
		agent.State = *state
	}
	if err := agent.Validate(); err != nil {
		return nil, err
	}
	err = s.store.AgentRepo().Save(ctx, agent)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

// Delete removes a agent by ID after checking for dependencies
func (s *AgentCommander) Delete(ctx context.Context, id UUID) error {
	_, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}
	return s.store.Atomic(ctx, func(store Store) error {
		numOfServices, err := store.ServiceRepo().CountByAgent(ctx, id)
		if err != nil {
			return err
		}
		if numOfServices > 0 {
			return errors.New("cannot delete agent with associated services")
		}
		return store.AgentRepo().Delete(ctx, id)
	})
}

// RotateToken regenerates the auth token
func (s *AgentCommander) RotateToken(ctx context.Context, id UUID) (*Agent, error) {
	agent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	_, err = agent.GenerateToken()
	if err != nil {
		return nil, err
	}
	if err := agent.Validate(); err != nil {
		return nil, err
	}
	err = s.store.AgentRepo().Save(ctx, agent)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

// UpdateState updates the agent state and the related timestamp
func (s *AgentCommander) UpdateState(ctx context.Context, id UUID, state AgentState) (*Agent, error) {
	agent, err := s.store.AgentRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	agent.State = state
	agent.LastStatusUpdate = time.Now()
	if err := agent.Validate(); err != nil {
		return nil, err
	}
	err = s.store.AgentRepo().Save(ctx, agent)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

type AgentRepository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity *Agent) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *Agent) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id UUID) error

	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Agent, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[Agent], error)

	// CountByProvider returns the number of agents for a specific provider
	CountByProvider(ctx context.Context, providerID UUID) (int64, error)

	// FindByTokenHash finds an agent by its token hash
	FindByTokenHash(ctx context.Context, tokenHash string) (*Agent, error)

	// MarkInactiveAgentsAsDisconnected marks agents that haven't updated their status in the given duration as disconnected
	MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error)
}

type AgentQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Agent, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, req *PageRequest) (*PageResponse[Agent], error)

	// CountByProvider returns the number of agents for a specific provider
	CountByProvider(ctx context.Context, providerID UUID) (int64, error)

	// FindByTokenHash finds an agent by its token hash
	FindByTokenHash(ctx context.Context, tokenHash string) (*Agent, error)

	// MarkInactiveAgentsAsDisconnected marks agents that haven't updated their status in the given duration as disconnected
	MarkInactiveAgentsAsDisconnected(ctx context.Context, inactiveDuration time.Duration) (int64, error)
}
