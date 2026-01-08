package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
)

const (
	EventTypeTokenCreated     EventType = "token.created"
	EventTypeTokenUpdated     EventType = "token.updated"
	EventTypeTokenDeleted     EventType = "token.deleted"
	EventTypeTokenRegenerated EventType = "token.regenerate"
)

// Token represents an authentication token
type Token struct {
	BaseEntity

	Name        string    `json:"name" gorm:"not null"`
	Role        auth.Role `json:"role" gorm:"not null"`
	PlainValue  string    `json:"-" gorm:"-"`
	HashedValue string    `json:"-" gorm:"not null"`
	ExpireAt    time.Time `json:"expireAt" gorm:"not null"`

	// Relationships
	ParticipantID *properties.UUID `json:"participantId,omitempty"`           // New field
	Participant   *Participant     `json:"-" gorm:"foreignKey:ParticipantID"` // New field
	AgentID       *properties.UUID `json:"agentId,omitempty"`
	Agent         *Agent           `json:"-" gorm:"foreignKey:AgentID"`
}

// NewToken is an helper method to create a token with appropriate scope settings
func NewToken(
	ctx context.Context,
	store Store,
	params CreateTokenParams,
) (*Token, error) {
	// If expireAt is nil, set it to 24 hours from now
	if params.ExpireAt == nil {
		defaultExpireAt := time.Now().Add(24 * time.Hour)
		params.ExpireAt = &defaultExpireAt
	}

	// Create token with basic fields
	token := &Token{
		Name:     params.Name,
		Role:     params.Role,
		ExpireAt: *params.ExpireAt,
	}

	// Set scope IDs based on role
	if params.ScopeID != nil {
		switch params.Role {
		case auth.RoleParticipant: // New Role (assuming it's defined, will be formally added in auth.go update)
			// Validate participant exists and set ID
			// Assuming store.ParticipantRepo().Exists(ctx, *scopeID) will be available
			exists, err := store.ParticipantRepo().Exists(ctx, *params.ScopeID)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, NewInvalidInputErrorf("invalid participant ID: %v", params.ScopeID)
			}
			token.ParticipantID = params.ScopeID
		case auth.RoleAgent:
			// Validate agent exists, set agent ID, and copy the participant ID from the agent
			agent, err := store.AgentRepo().Get(ctx, *params.ScopeID)
			if err != nil {
				return nil, NewInvalidInputErrorf("invalid agent ID: %v", err)
			}
			token.AgentID = params.ScopeID
			token.ParticipantID = &agent.ProviderID
		}
	}

	err := token.GenerateTokenValue()
	if err != nil {
		return nil, err
	}

	// Validate again after setting scope IDs and generating token
	if err := token.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}

	return token, nil
}

// TableName returns the table name for the token
func (Token) TableName() string {
	return "tokens"
}

// Validate ensures all Token fields are valid
func (t *Token) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("token name cannot be empty")
	}
	if t.HashedValue == "" {
		return fmt.Errorf("token hashed value cannot be empty")
	}
	if err := t.Role.Validate(); err != nil {
		return err
	}
	if t.ExpireAt.IsZero() {
		return fmt.Errorf("token expire at cannot be empty")
	}

	// Validate scope ID based on role
	switch t.Role {
	case auth.RoleAdmin:
		// No scope ID needed for admin
		if t.ParticipantID != nil || t.AgentID != nil { // Updated to check ParticipantID
			return fmt.Errorf("fulcrum admin tokens should not have any scope IDs")
		}
	case auth.RoleParticipant: // New Role (assuming it's defined)
		// Participant ID required for participant role
		if t.ParticipantID == nil {
			return fmt.Errorf("participant ID is required for participant role")
		}
		if t.AgentID != nil {
			return fmt.Errorf("participant tokens should only have participant ID set")
		}
	case auth.RoleAgent:
		// Agent ID and ParticipantID (from agent) required for agent role
		if t.AgentID == nil {
			return fmt.Errorf("agent ID is required for agent role")
		}
		if t.ParticipantID == nil { // Agent's ParticipantID
			return fmt.Errorf("participant ID is required for agent role")
		}
	}

	return nil
}

// IsExpired checks if the token is expired
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpireAt)
}

// GenerateTokenValue creates a secure random token and sets the HashedValue field
// The plain text value is only returned and never stored in the entity
func (t *Token) GenerateTokenValue() error {
	// Generate a secure random token (32 bytes = 256 bits)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Convert to base64
	t.PlainValue = base64.URLEncoding.EncodeToString(tokenBytes)

	// Store only the hash of the token
	t.HashedValue = HashTokenValue(t.PlainValue)

	return nil
}

// VerifyTokenValue checks if a token matches the stored hash
func (t *Token) VerifyTokenValue(value string) bool {
	return t.HashedValue == HashTokenValue(value)
}

// HashTokenValue creates a secure hash of a token value
func HashTokenValue(value string) string {
	hash := sha256.Sum256([]byte(value))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// Update updates the token properties
func (t *Token) Update(params UpdateTokenParams) error {
	if params.Name != nil {
		t.Name = *params.Name
	}
	if params.ExpireAt != nil {
		t.ExpireAt = *params.ExpireAt
	}
	return t.Validate()
}

// TokenCommander defines the interface for token command operations
type TokenCommander interface {
	// Create creates a new token
	Create(ctx context.Context, params CreateTokenParams) (*Token, error)

	// Update updates a token
	Update(ctx context.Context, params UpdateTokenParams) (*Token, error)

	// Delete removes a token by ID
	Delete(ctx context.Context, id properties.UUID) error

	// Regenerate regenerates the token value
	Regenerate(ctx context.Context, id properties.UUID) (*Token, error)
}

type CreateTokenParams struct {
	Name     string           `json:"name"`
	Role     auth.Role        `json:"role"`
	ExpireAt *time.Time       `json:"expireAt"`
	ScopeID  *properties.UUID `json:"scopeId"`
}

type UpdateTokenParams struct {
	ID       properties.UUID `json:"id"`
	Name     *string         `json:"name"`
	ExpireAt *time.Time      `json:"expireAt"`
}

// tokenCommander is the concrete implementation of TokenCommander
type tokenCommander struct {
	store Store
}

// NewTokenCommander creates a new TokenCommander
func NewTokenCommander(
	store Store,
) TokenCommander {
	return &tokenCommander{
		store: store,
	}
}

func (s *tokenCommander) Create(
	ctx context.Context,
	params CreateTokenParams,
) (*Token, error) {
	// Create, save and event
	var token *Token
	err := s.store.Atomic(ctx, func(store Store) error {
		var err error
		token, err = NewToken(ctx, store, params)
		if err != nil {
			return err
		}
		if err := store.TokenRepo().Create(ctx, token); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeTokenCreated, WithInitiatorCtx(ctx), WithToken(token))
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

	return token, nil
}

func (s *tokenCommander) Update(ctx context.Context,
	params UpdateTokenParams,
) (*Token, error) {
	// Validate token exists
	token, err := s.store.TokenRepo().Get(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	// Make a copy for event diff
	beforeTokenCopy := *token

	// Update, save and event
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := token.Update(params); err != nil {
			return err
		}
		if err := store.TokenRepo().Save(ctx, token); err != nil {
			return err
		}
		eventEntry, err := NewEvent(EventTypeTokenUpdated, WithInitiatorCtx(ctx), WithDiff(&beforeTokenCopy, token), WithToken(token))
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

	return token, nil
}

func (s *tokenCommander) Delete(ctx context.Context, id properties.UUID) error {
	// Get token before deletion for event purposes
	token, err := s.store.TokenRepo().Get(ctx, id)
	if err != nil {
		return err
	}

	// Delete and event
	return s.store.Atomic(ctx, func(store Store) error {
		if err := store.TokenRepo().Delete(ctx, id); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeTokenDeleted, WithInitiatorCtx(ctx), WithToken(token))
		if err != nil {
			return err
		}
		if err := store.EventRepo().Create(ctx, eventEntry); err != nil {
			return err
		}
		return err
	})
}

func (s *tokenCommander) Regenerate(ctx context.Context, id properties.UUID) (*Token, error) {
	// Validate token exists
	token, err := s.store.TokenRepo().Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Regenerate, save and event
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := token.GenerateTokenValue(); err != nil {
			return err
		}
		if err := store.TokenRepo().Save(ctx, token); err != nil {
			return err
		}

		eventEntry, err := NewEvent(EventTypeTokenRegenerated, WithInitiatorCtx(ctx), WithToken(token))
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

	return token, nil
}

type TokenRepository interface {
	TokenQuerier
	BaseEntityRepository[Token]

	// DeleteByParticipantID removes all tokens associated with a participant ID
	DeleteByParticipantID(ctx context.Context, participantID properties.UUID) error

	// DeleteByAgentID removes all tokens associated with an agent ID
	DeleteByAgentID(ctx context.Context, agentID properties.UUID) error
}

type TokenQuerier interface {
	BaseEntityQuerier[Token]

	// FindByHashedValue finds a token by its hashed value
	FindByHashedValue(ctx context.Context, hashedValue string) (*Token, error)
}
