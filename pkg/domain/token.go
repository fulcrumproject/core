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
	name string,
	role auth.Role,
	expireAt *time.Time,
	scopeID *properties.UUID,
) (*Token, error) {
	// If expireAt is nil, set it to 24 hours from now
	if expireAt == nil {
		defaultExpireAt := time.Now().Add(24 * time.Hour)
		expireAt = &defaultExpireAt
	}

	// Create token with basic fields
	token := &Token{
		Name:     name,
		Role:     role,
		ExpireAt: *expireAt,
	}

	// Set scope IDs based on role
	if scopeID != nil {
		switch role {
		case auth.RoleParticipant: // New Role (assuming it's defined, will be formally added in auth.go update)
			// Validate participant exists and set ID
			// Assuming store.ParticipantRepo().Exists(ctx, *scopeID) will be available
			exists, err := store.ParticipantRepo().Exists(ctx, *scopeID)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, NewInvalidInputErrorf("invalid participant ID: %v", scopeID)
			}
			token.ParticipantID = scopeID
		case auth.RoleAgent:
			// Validate agent exists, set agent ID, and copy the participant ID from the agent
			agent, err := store.AgentRepo().Get(ctx, *scopeID)
			if err != nil {
				return nil, NewInvalidInputErrorf("invalid agent ID: %v", err)
			}
			token.AgentID = scopeID
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
func (t *Token) Update(name *string, expireAt *time.Time) error {
	if name != nil {
		t.Name = *name
	}
	if expireAt != nil {
		t.ExpireAt = *expireAt
	}
	return t.Validate()
}

// TokenCommander defines the interface for token command operations
type TokenCommander interface {
	// Create creates a new token
	Create(ctx context.Context, name string, role auth.Role, expireAt *time.Time, scopeID *properties.UUID) (*Token, error)

	// Update updates a token
	Update(ctx context.Context, id properties.UUID, name *string, expireAt *time.Time) (*Token, error)

	// Delete removes a token by ID
	Delete(ctx context.Context, id properties.UUID) error

	// Regenerate regenerates the token value
	Regenerate(ctx context.Context, id properties.UUID) (*Token, error)
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
	name string,
	role auth.Role,
	expireAt *time.Time,
	scopeID *properties.UUID,
) (*Token, error) {
	// Validate permissions
	id := auth.MustGetIdentity(ctx)
	if !id.HasRole(auth.RoleAdmin) && role != id.Role {
		return nil, NewInvalidInputErrorf("role %s not allowed", role)
	}

	// Create, save and event
	var token *Token
	err := s.store.Atomic(ctx, func(store Store) error {
		var err error
		token, err = NewToken(ctx, store, name, role, expireAt, scopeID)
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
	id properties.UUID,
	name *string,
	expireAt *time.Time,
) (*Token, error) {
	// Validate token exists
	token, err := s.store.TokenRepo().Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Make a copy for event diff
	beforeTokenCopy := *token

	// Update, save and event
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := token.Update(name, expireAt); err != nil {
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

	// Create creates a new entity
	Create(ctx context.Context, entity *Token) error

	// Update updates an existing entity
	Save(ctx context.Context, entity *Token) error

	// Delete removes an entity by ID
	Delete(ctx context.Context, id properties.UUID) error

	// DeleteByParticipantID removes all tokens associated with a participant ID
	DeleteByParticipantID(ctx context.Context, participantID properties.UUID) error

	// DeleteByAgentID removes all tokens associated with an agent ID
	DeleteByAgentID(ctx context.Context, agentID properties.UUID) error
}

type TokenQuerier interface {
	// Get retrieves an entity by ID
	Get(ctx context.Context, id properties.UUID) (*Token, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authIdentityScope *auth.IdentityScope, req *PageRequest) (*PageResponse[Token], error)

	// FindByHashedValue finds a token by its hashed value
	FindByHashedValue(ctx context.Context, hashedValue string) (*Token, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id properties.UUID) (auth.ObjectScope, error)
}
