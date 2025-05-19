package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// Token represents an authentication token
type Token struct {
	BaseEntity

	Name        string    `json:"name" gorm:"not null"`
	Role        AuthRole  `json:"role" gorm:"not null"`
	PlainValue  string    `json:"-" gorm:"-"`
	HashedValue string    `json:"-" gorm:"not null"`
	ExpireAt    time.Time `json:"expireAt" gorm:"not null"`

	// Relationships
	ParticipantID *UUID        `json:"participantId,omitempty"`           // New field
	Participant   *Participant `json:"-" gorm:"foreignKey:ParticipantID"` // New field
	AgentID       *UUID        `json:"agentId,omitempty"`
	Agent         *Agent       `json:"-" gorm:"foreignKey:AgentID"`
}

// NewToken is an helper method to create a token with appropriate scope settings
func NewToken(
	ctx context.Context,
	store Store,
	name string,
	role AuthRole,
	expireAt time.Time,
	scopeID *UUID,
) (*Token, error) {
	// Create token with basic fields
	token := &Token{
		Name:     name,
		Role:     role,
		ExpireAt: expireAt,
	}

	// Set scope IDs based on role
	if scopeID != nil {
		switch role {
		case RoleParticipant: // New Role (assuming it's defined, will be formally added in auth.go update)
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
		case RoleAgent:
			// Validate agent exists, set agent ID, and copy the participant ID from the agent
			agent, err := store.AgentRepo().FindByID(ctx, *scopeID)
			if err != nil {
				return nil, NewInvalidInputErrorf("invalid agent ID: %v", err)
			}
			token.AgentID = scopeID

			// Make a copy of the agent's participant ID and set it
			if agent.ParticipantID != (UUID{}) { // Check if ParticipantID is not empty UUID
				participantID := agent.ParticipantID
				token.ParticipantID = &participantID
			} else {
				// This case should ideally not happen if an agent always belongs to a participant
				return nil, NewInvalidInputErrorf("agent %v does not have an associated participant ID", agent.ID)
			}
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
	case RoleFulcrumAdmin:
		// No scope ID needed for admin
		if t.ParticipantID != nil || t.AgentID != nil { // Updated to check ParticipantID
			return fmt.Errorf("fulcrum admin tokens should not have any scope IDs")
		}
	case RoleParticipant: // New Role (assuming it's defined)
		// Participant ID required for participant role
		if t.ParticipantID == nil {
			return fmt.Errorf("participant ID is required for participant role")
		}
		if t.AgentID != nil {
			return fmt.Errorf("participant tokens should only have participant ID set")
		}
	case RoleAgent:
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
	Create(ctx context.Context, name string, role AuthRole, expireAt time.Time, scopeID *UUID) (*Token, error)

	// Update updates a token
	Update(ctx context.Context, id UUID, name *string, expireAt *time.Time) (*Token, error)

	// Delete removes a token by ID
	Delete(ctx context.Context, id UUID) error

	// Regenerate regenerates the token value
	Regenerate(ctx context.Context, id UUID) (*Token, error)
}

// tokenCommander is the concrete implementation of TokenCommander
type tokenCommander struct {
	store          Store
	auditCommander AuditEntryCommander
}

// NewTokenCommander creates a new TokenCommander
func NewTokenCommander(
	store Store,
	auditCommander AuditEntryCommander,
) TokenCommander {
	return &tokenCommander{
		store:          store,
		auditCommander: auditCommander,
	}
}

func (s *tokenCommander) Create(
	ctx context.Context,
	name string,
	role AuthRole,
	expireAt time.Time,
	scopeID *UUID,
) (*Token, error) {
	// Validate permissions
	id := MustGetAuthIdentity(ctx)
	if !id.IsRole(RoleFulcrumAdmin) && role != id.Role() {
		return nil, NewInvalidInputErrorf("role %s not allowed", role)
	}

	// Create, save and audit
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
		_, err = s.auditCommander.CreateCtx(
			ctx,
			EventTypeTokenCreated,
			JSON{"state": token},
			&token.ID,
			token.ParticipantID, // Updated
			token.AgentID,
			nil) // BrokerID removed
		return err
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (s *tokenCommander) Update(ctx context.Context,
	id UUID,
	name *string,
	expireAt *time.Time,
) (*Token, error) {
	// Validate token exists
	token, err := s.store.TokenRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Make a copy for audit diff
	beforeTokenCopy := *token

	// Update, save and audit
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := token.Update(name, expireAt); err != nil {
			return err
		}
		if err := store.TokenRepo().Save(ctx, token); err != nil {
			return err
		}
		_, err = s.auditCommander.CreateCtxWithDiff(
			ctx,
			EventTypeTokenUpdated,
			&id,
			token.ParticipantID, // Updated
			token.AgentID,
			nil, // BrokerID removed
			&beforeTokenCopy,
			token)
		return err
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (s *tokenCommander) Delete(ctx context.Context, id UUID) error {
	// Get token before deletion for audit purposes
	token, err := s.store.TokenRepo().FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete and audit
	return s.store.Atomic(ctx, func(store Store) error {
		if err := store.TokenRepo().Delete(ctx, id); err != nil {
			return err
		}

		_, err := s.auditCommander.CreateCtx(
			ctx,
			EventTypeTokenDeleted,
			JSON{"state": token},
			&id,
			token.ParticipantID, // Updated
			token.AgentID,
			nil) // BrokerID removed
		return err
	})
}

func (s *tokenCommander) Regenerate(ctx context.Context, id UUID) (*Token, error) {
	// Validate token exists
	token, err := s.store.TokenRepo().FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Regenerate, save and audit
	err = s.store.Atomic(ctx, func(store Store) error {
		if err := token.GenerateTokenValue(); err != nil {
			return err
		}
		if err := store.TokenRepo().Save(ctx, token); err != nil {
			return err
		}
		_, err = s.auditCommander.CreateCtx(
			ctx,
			EventTypeTokenRegenerated,
			JSON{"state": token},
			&id,
			token.ParticipantID, // Updated
			token.AgentID,
			nil) // BrokerID removed
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
	Delete(ctx context.Context, id UUID) error

	// DeleteByParticipantID removes all tokens associated with a participant ID
	DeleteByParticipantID(ctx context.Context, participantID UUID) error

	// DeleteByAgentID removes all tokens associated with an agent ID
	DeleteByAgentID(ctx context.Context, agentID UUID) error
}

type TokenQuerier interface {
	// FindByID retrieves an entity by ID
	FindByID(ctx context.Context, id UUID) (*Token, error)

	// List retrieves a list of entities based on the provided filters
	List(ctx context.Context, authScope *AuthScope, req *PageRequest) (*PageResponse[Token], error)

	// FindByHashedValue finds a token by its hashed value
	FindByHashedValue(ctx context.Context, hashedValue string) (*Token, error)

	// Retrieve the auth scope for the entity
	AuthScope(ctx context.Context, id UUID) (*AuthScope, error)
}
