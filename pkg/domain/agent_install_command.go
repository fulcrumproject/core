package domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/google/uuid"
)

const (
	EventTypeAgentInstallCommandCreated     EventType = "agent.install_command_created"
	EventTypeAgentInstallCommandRegenerated EventType = "agent.install_command_regenerated"
)

// AgentInstallCommand is the 1:1-per-agent record that gates access to an
// install URL. TokenHashed is the SHA256 of the plain token (stored for lookup
// by the public fetch endpoint). VaultKey points at the plain-text token held
// in the vault, fetched by the authenticated re-copy endpoint.
type AgentInstallCommand struct {
	BaseEntity

	AgentID     properties.UUID `json:"agentId" gorm:"type:uuid;uniqueIndex;not null"`
	TokenHashed string          `json:"-" gorm:"uniqueIndex;not null"`
	VaultKey    string          `json:"-" gorm:"not null"`
	ExpiresAt   time.Time       `json:"expiresAt" gorm:"not null"`

	// PlainToken is transient: set only on freshly minted (Create) or rotated
	// (Regenerate) commands so the HTTP handler can render the URL without an
	// extra vault round-trip. Never persisted, never serialized. Mirrors
	// Token.PlainValue.
	PlainToken string `json:"-" gorm:"-"`

	Agent *Agent `json:"-" gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for the entity.
func (AgentInstallCommand) TableName() string {
	return "agent_install_commands"
}

// IsExpired reports whether the command is past its expiry.
func (c *AgentInstallCommand) IsExpired() bool {
	return time.Now().UTC().After(c.ExpiresAt)
}

// AgentInstallCommandCommander defines the interface for install-command write operations.
type AgentInstallCommandCommander interface {
	// Create mints a fresh install command for the agent. Fails with a ConflictError
	// if one already exists for the agent (use Regenerate instead).
	Create(ctx context.Context, agentID properties.UUID) (*AgentInstallCommand, error)

	// Regenerate rotates an existing install command's token and expiry. Fails
	// with a NotFoundError if none exists (use Create first).
	Regenerate(ctx context.Context, agentID properties.UUID) (*AgentInstallCommand, error)

	// DeleteByAgentID removes the vault entry for the agent's install command.
	// The database row is expected to be removed via the agent FK cascade.
	// Safe to call when no command exists — it becomes a no-op.
	DeleteByAgentID(ctx context.Context, agentID properties.UUID) error
}

// AgentInstallCommandRepository is the persistence interface.
type AgentInstallCommandRepository interface {
	AgentInstallCommandQuerier

	Create(ctx context.Context, cmd *AgentInstallCommand) error
	Save(ctx context.Context, cmd *AgentInstallCommand) error
	DeleteByAgentID(ctx context.Context, agentID properties.UUID) error
}

// AgentInstallCommandQuerier is the read-only interface.
type AgentInstallCommandQuerier interface {
	// GetByAgentID returns the install command for the given agent, or NotFoundError.
	GetByAgentID(ctx context.Context, agentID properties.UUID) (*AgentInstallCommand, error)

	// FindByHashedToken looks up a command by the SHA256 hash of the plain token.
	// Used by the public /install/{token} handler after hashing the inbound token.
	FindByHashedToken(ctx context.Context, hashed string) (*AgentInstallCommand, error)
}

type agentInstallCommandCommander struct {
	store Store
	vault schema.Vault
	ttl   time.Duration
}

// NewAgentInstallCommandCommander creates a new default AgentInstallCommandCommander.
func NewAgentInstallCommandCommander(store Store, vault schema.Vault, ttl time.Duration) *agentInstallCommandCommander {
	return &agentInstallCommandCommander{
		store: store,
		vault: vault,
		ttl:   ttl,
	}
}

func installVaultKey(agentID properties.UUID) string {
	return "agent-install/" + agentID.String()
}

func (c *agentInstallCommandCommander) Create(ctx context.Context, agentID properties.UUID) (*AgentInstallCommand, error) {
	agent, err := c.store.AgentRepo().Get(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if agent.AgentType == nil || agent.AgentType.CmdTemplate == "" {
		return nil, NewInvalidInputErrorf("agent type has no install templates configured")
	}

	var cmd *AgentInstallCommand
	err = c.store.Atomic(ctx, func(store Store) error {
		if _, existsErr := store.AgentInstallCommandRepo().GetByAgentID(ctx, agentID); existsErr == nil {
			return NewConflictErrorf("install command already exists for agent %s", agentID)
		} else if !errors.As(existsErr, &NotFoundError{}) {
			return existsErr
		}

		plain, err := GenerateInstallToken()
		if err != nil {
			return err
		}
		vaultKey := installVaultKey(agentID)
		if err := c.vault.Save(ctx, vaultKey, plain, nil); err != nil {
			return fmt.Errorf("failed to save install token to vault: %w", err)
		}

		now := time.Now().UTC()
		cmd = &AgentInstallCommand{
			BaseEntity:  BaseEntity{ID: properties.UUID(uuid.New())},
			AgentID:     agentID,
			TokenHashed: HashTokenValue(plain),
			VaultKey:    vaultKey,
			ExpiresAt:   now.Add(c.ttl),
			PlainToken:  plain,
		}
		if err := store.AgentInstallCommandRepo().Create(ctx, cmd); err != nil {
			return err
		}

		event, err := NewEvent(
			EventTypeAgentInstallCommandCreated,
			WithInitiatorCtx(ctx),
			WithAgent(agent),
		)
		if err != nil {
			return err
		}
		event.Payload = properties.JSON{
			"createdAt": now.Format(time.RFC3339Nano),
			"expiresAt": cmd.ExpiresAt.Format(time.RFC3339Nano),
		}
		return store.EventRepo().Create(ctx, event)
	})
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func (c *agentInstallCommandCommander) Regenerate(ctx context.Context, agentID properties.UUID) (*AgentInstallCommand, error) {
	agent, err := c.store.AgentRepo().Get(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if agent.AgentType == nil || agent.AgentType.CmdTemplate == "" {
		return nil, NewInvalidInputErrorf("agent type has no install templates configured")
	}

	var cmd *AgentInstallCommand
	err = c.store.Atomic(ctx, func(store Store) error {
		existing, err := store.AgentInstallCommandRepo().GetByAgentID(ctx, agentID)
		if err != nil {
			return err
		}

		plain, err := GenerateInstallToken()
		if err != nil {
			return err
		}
		if err := c.vault.Save(ctx, existing.VaultKey, plain, nil); err != nil {
			return fmt.Errorf("failed to save install token to vault: %w", err)
		}

		now := time.Now().UTC()
		existing.TokenHashed = HashTokenValue(plain)
		existing.ExpiresAt = now.Add(c.ttl)
		existing.PlainToken = plain

		if err := store.AgentInstallCommandRepo().Save(ctx, existing); err != nil {
			return err
		}

		event, err := NewEvent(
			EventTypeAgentInstallCommandRegenerated,
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
		cmd = existing
		return nil
	})
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

func (c *agentInstallCommandCommander) DeleteByAgentID(ctx context.Context, agentID properties.UUID) error {
	cmd, err := c.store.AgentInstallCommandRepo().GetByAgentID(ctx, agentID)
	if err != nil {
		if errors.As(err, &NotFoundError{}) {
			return nil
		}
		return err
	}
	if err := c.vault.Delete(ctx, cmd.VaultKey); err != nil {
		return fmt.Errorf("failed to delete install token from vault: %w", err)
	}
	return nil
}
