package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/google/uuid"
)

const (
	EventTypeAgentInstallTokenCreated     EventType = "agent.install_token_created"
	EventTypeAgentInstallTokenRegenerated EventType = "agent.install_token_regenerated"
	EventTypeAgentInstallTokenRevoked     EventType = "agent.install_token_revoked"
)

// installTokenTTL is how long a freshly minted or regenerated install token
// (and its paired bootstrap bearer) remains usable. Short by design: the token
// is meant to be consumed immediately by an installer script.
const installTokenTTL = 5 * time.Minute

// AgentInstallToken is the 1:1-per-agent record that gates access to an
// install URL. TokenHashed is the SHA256 of the plain token (used by the public
// fetch endpoint to look up the record). The plain token is never persisted:
// it is returned to the caller exactly once in the Create/Regenerate response
// and cannot be recovered thereafter — if lost, Regenerate.
type AgentInstallToken struct {
	BaseEntity

	AgentID     properties.UUID `json:"agentId" gorm:"type:uuid;uniqueIndex;not null"`
	TokenHashed string          `json:"-" gorm:"uniqueIndex;not null"`
	ExpiresAt   time.Time       `json:"expiresAt" gorm:"not null"`

	// BootstrapTokenID references an agent-role Token minted alongside the
	// install token. The plain value of that token is rendered into the
	// cmdTemplate's Authorization header so the installer can authenticate
	// against the protected fetch endpoint. Its lifecycle is tied to this
	// record: rotated on Regenerate, deleted on Revoke.
	BootstrapTokenID *properties.UUID `json:"-" gorm:"type:uuid"`

	// PlainToken is transient: set only on freshly minted (Create) or rotated
	// (Regenerate) records so the HTTP handler can render the URL in the same
	// response. Never persisted, never serialized.
	PlainToken string `json:"-" gorm:"-"`

	// PlainBootstrapToken is transient: the plain value of the bootstrap bearer
	// token, returned once at Create/Regenerate and never recoverable after.
	PlainBootstrapToken string `json:"-" gorm:"-"`

	Agent *Agent `json:"-" gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for the entity.
func (AgentInstallToken) TableName() string {
	return "agent_install_tokens"
}

// IsExpired reports whether the token is past its expiry.
func (c *AgentInstallToken) IsExpired() bool {
	return time.Now().UTC().After(c.ExpiresAt)
}

// AgentInstallTokenCommander defines the interface for install-token write operations.
type AgentInstallTokenCommander interface {
	// Create mints a fresh install token for the agent. Fails with a ConflictError
	// if one already exists for the agent (use Regenerate instead).
	Create(ctx context.Context, agentID properties.UUID) (*AgentInstallToken, error)

	// Regenerate rotates an existing install token and its expiry. Fails
	// with a NotFoundError if none exists (use Create first). Operates on
	// expired records by design — rotation is the standard recovery path.
	Regenerate(ctx context.Context, agentID properties.UUID) (*AgentInstallToken, error)

	// Revoke deletes the install token for the agent without minting a new one.
	// Returns NotFoundError if none exists. Operates on expired records by
	// design so admins can clean up stale rows.
	Revoke(ctx context.Context, agentID properties.UUID) error
}

// AgentInstallTokenRepository is the persistence interface.
type AgentInstallTokenRepository interface {
	AgentInstallTokenQuerier

	Create(ctx context.Context, tok *AgentInstallToken) error
	Save(ctx context.Context, tok *AgentInstallToken) error
	DeleteByAgentID(ctx context.Context, agentID properties.UUID) error
}

// AgentInstallTokenQuerier is the read-only interface.
type AgentInstallTokenQuerier interface {
	// GetByAgentID returns the install token for the given agent, or NotFoundError.
	GetByAgentID(ctx context.Context, agentID properties.UUID) (*AgentInstallToken, error)

	// FindByHashedToken looks up a record by the SHA256 hash of the plain token.
	// Used by the public /install/{token} handler after hashing the inbound token.
	FindByHashedToken(ctx context.Context, hashed string) (*AgentInstallToken, error)
}

// mintBootstrapToken creates an agent-role Token scoped to agentID, expiring at
// expiresAt, and persists it via store.TokenRepo(). The returned token carries
// PlainValue (needed for the installer's Authorization header) which is never
// persisted.
//
// The token is written directly through the repo (no `token.created` /
// `token.deleted` events) because the bootstrap token's lifecycle is owned by
// the install-token flow — it is created, rotated, and revoked alongside the
// surrounding AgentInstallToken, and those transitions are captured by the
// `agent.install_token_*` events. Going through tokenCommander would emit
// duplicate audit entries with no extra information.
func mintBootstrapToken(ctx context.Context, store Store, agentID properties.UUID, expiresAt time.Time) (*Token, error) {
	scope := agentID
	token, err := NewToken(ctx, store, CreateTokenParams{
		Name:     fmt.Sprintf("install-bootstrap-%s", agentID),
		Role:     auth.RoleAgent,
		ExpireAt: &expiresAt,
		ScopeID:  &scope,
	})
	if err != nil {
		return nil, err
	}
	if err := store.TokenRepo().Create(ctx, token); err != nil {
		return nil, err
	}
	return token, nil
}

type agentInstallTokenCommander struct {
	store Store
}

// NewAgentInstallTokenCommander creates a new default AgentInstallTokenCommander.
func NewAgentInstallTokenCommander(store Store) *agentInstallTokenCommander {
	return &agentInstallTokenCommander{store: store}
}

func (c *agentInstallTokenCommander) Create(ctx context.Context, agentID properties.UUID) (*AgentInstallToken, error) {
	var tok *AgentInstallToken
	err := c.store.Atomic(ctx, func(store Store) error {
		agent, err := store.AgentRepo().Get(ctx, agentID)
		if err != nil {
			return err
		}
		if agent.AgentType == nil || !agent.AgentType.HasInstallTemplates() {
			return NewInvalidInputErrorf("agent type has no install templates configured")
		}

		if _, existsErr := store.AgentInstallTokenRepo().GetByAgentID(ctx, agentID); existsErr == nil {
			return NewConflictErrorf("install token already exists for agent %s", agentID)
		} else if !errors.As(existsErr, &NotFoundError{}) {
			return existsErr
		}

		plain, err := generateSecureToken()
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		expiresAt := now.Add(installTokenTTL)

		bootstrap, err := mintBootstrapToken(ctx, store, agentID, expiresAt)
		if err != nil {
			return err
		}

		tok = &AgentInstallToken{
			BaseEntity:          BaseEntity{ID: properties.UUID(uuid.New())},
			AgentID:             agentID,
			TokenHashed:         HashTokenValue(plain),
			ExpiresAt:           expiresAt,
			BootstrapTokenID:    &bootstrap.ID,
			PlainToken:          plain,
			PlainBootstrapToken: bootstrap.PlainValue,
			Agent:               agent,
		}
		if err := store.AgentInstallTokenRepo().Create(ctx, tok); err != nil {
			return err
		}

		event, err := NewEvent(
			EventTypeAgentInstallTokenCreated,
			WithInitiatorCtx(ctx),
			WithAgent(agent),
		)
		if err != nil {
			return err
		}
		event.Payload = properties.JSON{
			"createdAt": now.Format(time.RFC3339Nano),
			"expiresAt": tok.ExpiresAt.Format(time.RFC3339Nano),
		}
		return store.EventRepo().Create(ctx, event)
	})
	if err != nil {
		return nil, err
	}
	return tok, nil
}

func (c *agentInstallTokenCommander) Regenerate(ctx context.Context, agentID properties.UUID) (*AgentInstallToken, error) {
	var tok *AgentInstallToken
	err := c.store.Atomic(ctx, func(store Store) error {
		agent, err := store.AgentRepo().Get(ctx, agentID)
		if err != nil {
			return err
		}
		if agent.AgentType == nil || !agent.AgentType.HasInstallTemplates() {
			return NewInvalidInputErrorf("agent type has no install templates configured")
		}

		existing, err := store.AgentInstallTokenRepo().GetByAgentID(ctx, agentID)
		if err != nil {
			return err
		}

		plain, err := generateSecureToken()
		if err != nil {
			return err
		}

		if existing.BootstrapTokenID != nil {
			if err := store.TokenRepo().Delete(ctx, *existing.BootstrapTokenID); err != nil && !errors.As(err, &NotFoundError{}) {
				return err
			}
		}

		now := time.Now().UTC()
		expiresAt := now.Add(installTokenTTL)

		bootstrap, err := mintBootstrapToken(ctx, store, agentID, expiresAt)
		if err != nil {
			return err
		}

		existing.TokenHashed = HashTokenValue(plain)
		existing.ExpiresAt = expiresAt
		existing.BootstrapTokenID = &bootstrap.ID
		existing.PlainToken = plain
		existing.PlainBootstrapToken = bootstrap.PlainValue
		existing.Agent = agent

		if err := store.AgentInstallTokenRepo().Save(ctx, existing); err != nil {
			return err
		}

		event, err := NewEvent(
			EventTypeAgentInstallTokenRegenerated,
			WithInitiatorCtx(ctx),
			WithAgent(agent),
		)
		if err != nil {
			return err
		}
		event.Payload = properties.JSON{
			"regeneratedAt": now.Format(time.RFC3339Nano),
			"expiresAt":     existing.ExpiresAt.Format(time.RFC3339Nano),
		}
		if err := store.EventRepo().Create(ctx, event); err != nil {
			return err
		}
		tok = existing
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tok, nil
}

func (c *agentInstallTokenCommander) Revoke(ctx context.Context, agentID properties.UUID) error {
	return c.store.Atomic(ctx, func(store Store) error {
		agent, err := store.AgentRepo().Get(ctx, agentID)
		if err != nil {
			return err
		}

		existing, err := store.AgentInstallTokenRepo().GetByAgentID(ctx, agentID)
		if err != nil {
			return err
		}
		if existing.BootstrapTokenID != nil {
			if err := store.TokenRepo().Delete(ctx, *existing.BootstrapTokenID); err != nil && !errors.As(err, &NotFoundError{}) {
				return err
			}
		}
		if err := store.AgentInstallTokenRepo().DeleteByAgentID(ctx, agentID); err != nil {
			return err
		}

		event, err := NewEvent(
			EventTypeAgentInstallTokenRevoked,
			WithInitiatorCtx(ctx),
			WithAgent(agent),
		)
		if err != nil {
			return err
		}
		event.Payload = properties.JSON{
			"revokedAt": time.Now().UTC().Format(time.RFC3339Nano),
		}
		return store.EventRepo().Create(ctx, event)
	})
}
