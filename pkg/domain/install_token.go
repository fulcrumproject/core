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

// InstallTokenEntityType is the kind of entity an install token belongs to.
// Mirrors EventType — a typed string so commander/repo signatures can't be
// passed a stray value.
type InstallTokenEntityType string

const (
	InstallTokenEntityTypeAgent          InstallTokenEntityType = "agent"
	InstallTokenEntityTypeInfrastructure InstallTokenEntityType = "infrastructure"
)

const (
	EventTypeAgentInstallTokenCreated     EventType = "agent.install_token_created"
	EventTypeAgentInstallTokenRegenerated EventType = "agent.install_token_regenerated"
	EventTypeAgentInstallTokenRevoked     EventType = "agent.install_token_revoked"

	EventTypeInfrastructureInstallTokenCreated     EventType = "infrastructure.install_token_created"
	EventTypeInfrastructureInstallTokenRegenerated EventType = "infrastructure.install_token_regenerated"
	EventTypeInfrastructureInstallTokenRevoked     EventType = "infrastructure.install_token_revoked"
)

// installTokenTTL is how long a freshly minted or regenerated install token
// (and its paired bootstrap bearer) remains usable. Short by design: the token
// is meant to be consumed immediately by an installer script.
const installTokenTTL = 5 * time.Minute

// InstallToken gates access to an install URL for either an Agent or an
// Infrastructure (selected by EntityType+EntityID). TokenHashed is the SHA256
// of the plain token (used by the public fetch endpoint to look up the
// record). The plain token is never persisted: it is returned to the caller
// exactly once in the Create/Regenerate response and cannot be recovered
// thereafter — if lost, Regenerate.
type InstallToken struct {
	BaseEntity

	EntityType InstallTokenEntityType `json:"entityType" gorm:"type:text;not null;uniqueIndex:ux_install_tokens_entity"`
	EntityID   properties.UUID        `json:"entityId" gorm:"type:uuid;not null;uniqueIndex:ux_install_tokens_entity"`

	TokenHashed string    `json:"-" gorm:"uniqueIndex;not null"`
	ExpiresAt   time.Time `json:"expiresAt" gorm:"not null"`

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

	// Agent / Infrastructure are polymorphic — exactly one is hydrated by the
	// repository after lookup, picked by EntityType. They aren't real FKs (we
	// can't preload both from a single column), so gorm:"-" keeps them out of
	// the schema.
	Agent          *Agent          `json:"-" gorm:"-"`
	Infrastructure *Infrastructure `json:"-" gorm:"-"`
}

// TableName returns the table name for the entity.
func (InstallToken) TableName() string {
	return "install_tokens"
}

// IsExpired reports whether the token is past its expiry.
func (c *InstallToken) IsExpired() bool {
	return time.Now().UTC().After(c.ExpiresAt)
}

// InstallTokenCommander defines the interface for install-token write operations.
type InstallTokenCommander interface {
	// Create mints a fresh install token for the entity. Fails with a
	// ConflictError if one already exists (use Regenerate instead).
	Create(ctx context.Context, entityType InstallTokenEntityType, entityID properties.UUID) (*InstallToken, error)

	// Regenerate rotates an existing install token and its expiry. Fails
	// with a NotFoundError if none exists (use Create first). Operates on
	// expired records by design — rotation is the standard recovery path.
	Regenerate(ctx context.Context, entityType InstallTokenEntityType, entityID properties.UUID) (*InstallToken, error)

	// Revoke deletes the install token for the entity without minting a new
	// one. Returns NotFoundError if none exists. Operates on expired records
	// by design so admins can clean up stale rows.
	Revoke(ctx context.Context, entityType InstallTokenEntityType, entityID properties.UUID) error
}

// InstallTokenRepository is the persistence interface.
type InstallTokenRepository interface {
	InstallTokenQuerier

	Create(ctx context.Context, tok *InstallToken) error
	Save(ctx context.Context, tok *InstallToken) error
	DeleteByEntity(ctx context.Context, entityType InstallTokenEntityType, entityID properties.UUID) error
}

// InstallTokenQuerier is the read-only interface.
type InstallTokenQuerier interface {
	// GetByEntity returns the install token for the given entity, or NotFoundError.
	GetByEntity(ctx context.Context, entityType InstallTokenEntityType, entityID properties.UUID) (*InstallToken, error)

	// FindByHashedToken looks up a record by the SHA256 hash of the plain token.
	// Used by the public /install/{token} handler after hashing the inbound token.
	FindByHashedToken(ctx context.Context, hashed string) (*InstallToken, error)
}

// mintBootstrapToken creates an agent-role Token scoped to entityID and
// persists it via store.TokenRepo(). The returned token carries PlainValue
// (needed for the installer's Authorization header) which is never persisted.
//
// The same RoleAgent + AgentID=entityID shape works for both Agent and
// Infrastructure installs: each entity's repo treats the IdentityScope's
// AgentID coordinate as a self-reference, so a token issued here passes the
// entity-specific AuthScope check.
//
// This bypasses NewToken's RoleAgent scope-validation path on purpose:
//   - The install-token commander has already loaded and validated the
//     entity, so re-checking would be redundant.
//   - NewToken's RoleAgent branch hardcodes AgentRepo.Get, which would
//     spuriously fail when entityID points at an Infrastructure row.
//
// The token is also written directly through the repo (no `token.created` /
// `token.deleted` events) because the bootstrap token's lifecycle is owned by
// the install-token flow — it is created, rotated, and revoked alongside the
// surrounding InstallToken, and those transitions are captured by the
// `*.install_token_*` events. Going through tokenCommander would emit
// duplicate audit entries with no extra information.
func mintBootstrapToken(ctx context.Context, store Store, entityID, providerID properties.UUID, expiresAt time.Time) (*Token, error) {
	id := entityID
	pid := providerID
	token := &Token{
		Name:          fmt.Sprintf("install-bootstrap-%s", entityID),
		Role:          auth.RoleAgent,
		ExpireAt:      expiresAt,
		AgentID:       &id,
		ParticipantID: &pid,
	}
	if err := token.GenerateTokenValue(); err != nil {
		return nil, err
	}
	if err := token.Validate(); err != nil {
		return nil, InvalidInputError{Err: err}
	}
	if err := store.TokenRepo().Create(ctx, token); err != nil {
		return nil, err
	}
	return token, nil
}

type installTokenCommander struct {
	store Store
}

// NewInstallTokenCommander creates a new default InstallTokenCommander.
func NewInstallTokenCommander(store Store) *installTokenCommander {
	return &installTokenCommander{store: store}
}

// entityCtx bundles the per-entity bits the commander needs: the embedded
// template state to validate against, the matching event option, the
// provider id (used as the bootstrap token's ParticipantID), and a hydrate
// function attaching the loaded entity to the token. Loaded once per call;
// lets the three commander methods stay mostly entity-agnostic.
type entityCtx struct {
	templates  *TemplateValidation
	providerID properties.UUID
	eventOpt   EventOption
	hydrate    func(*InstallToken)
}

func loadEntityCtx(ctx context.Context, store Store, entityType InstallTokenEntityType, entityID properties.UUID) (entityCtx, error) {
	switch entityType {
	case InstallTokenEntityTypeAgent:
		agent, err := store.AgentRepo().Get(ctx, entityID)
		if err != nil {
			return entityCtx{}, err
		}
		if agent.AgentType == nil {
			return entityCtx{}, NewInvalidInputErrorf("agent type not loaded for agent %s", entityID)
		}
		return entityCtx{
			templates:  &agent.AgentType.TemplateValidation,
			providerID: agent.ProviderID,
			eventOpt:   WithAgent(agent),
			hydrate:    func(tok *InstallToken) { tok.Agent = agent },
		}, nil
	case InstallTokenEntityTypeInfrastructure:
		infra, err := store.InfrastructureRepo().Get(ctx, entityID)
		if err != nil {
			return entityCtx{}, err
		}
		if infra.InfrastructureType == nil {
			return entityCtx{}, NewInvalidInputErrorf("infrastructure type not loaded for infrastructure %s", entityID)
		}
		return entityCtx{
			templates:  &infra.InfrastructureType.TemplateValidation,
			providerID: infra.ProviderID,
			eventOpt:   WithInfrastructure(infra),
			hydrate:    func(tok *InstallToken) { tok.Infrastructure = infra },
		}, nil
	default:
		return entityCtx{}, NewInvalidInputErrorf("unknown install-token entity type %q", entityType)
	}
}

func eventType(entityType InstallTokenEntityType, action string) EventType {
	switch entityType {
	case InstallTokenEntityTypeAgent:
		switch action {
		case "created":
			return EventTypeAgentInstallTokenCreated
		case "regenerated":
			return EventTypeAgentInstallTokenRegenerated
		case "revoked":
			return EventTypeAgentInstallTokenRevoked
		}
	case InstallTokenEntityTypeInfrastructure:
		switch action {
		case "created":
			return EventTypeInfrastructureInstallTokenCreated
		case "regenerated":
			return EventTypeInfrastructureInstallTokenRegenerated
		case "revoked":
			return EventTypeInfrastructureInstallTokenRevoked
		}
	}
	return EventType("")
}

func (c *installTokenCommander) Create(ctx context.Context, entityType InstallTokenEntityType, entityID properties.UUID) (*InstallToken, error) {
	var tok *InstallToken
	err := c.store.Atomic(ctx, func(store Store) error {
		ec, err := loadEntityCtx(ctx, store, entityType, entityID)
		if err != nil {
			return err
		}
		if !ec.templates.HasInstallTemplates() {
			return NewInvalidInputErrorf("entity has no install templates configured")
		}

		if _, existsErr := store.InstallTokenRepo().GetByEntity(ctx, entityType, entityID); existsErr == nil {
			return NewConflictErrorf("install token already exists for %s %s", entityType, entityID)
		} else if !errors.As(existsErr, &NotFoundError{}) {
			return existsErr
		}

		plain, err := generateSecureToken()
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		expiresAt := now.Add(installTokenTTL)

		bootstrap, err := mintBootstrapToken(ctx, store, entityID, ec.providerID, expiresAt)
		if err != nil {
			return err
		}

		tok = &InstallToken{
			BaseEntity:          BaseEntity{ID: properties.UUID(uuid.New())},
			EntityType:          entityType,
			EntityID:            entityID,
			TokenHashed:         HashTokenValue(plain),
			ExpiresAt:           expiresAt,
			BootstrapTokenID:    &bootstrap.ID,
			PlainToken:          plain,
			PlainBootstrapToken: bootstrap.PlainValue,
		}
		ec.hydrate(tok)
		if err := store.InstallTokenRepo().Create(ctx, tok); err != nil {
			return err
		}

		event, err := NewEvent(
			eventType(entityType, "created"),
			WithInitiatorCtx(ctx),
			ec.eventOpt,
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

func (c *installTokenCommander) Regenerate(ctx context.Context, entityType InstallTokenEntityType, entityID properties.UUID) (*InstallToken, error) {
	var tok *InstallToken
	err := c.store.Atomic(ctx, func(store Store) error {
		ec, err := loadEntityCtx(ctx, store, entityType, entityID)
		if err != nil {
			return err
		}
		if !ec.templates.HasInstallTemplates() {
			return NewInvalidInputErrorf("entity has no install templates configured")
		}

		existing, err := store.InstallTokenRepo().GetByEntity(ctx, entityType, entityID)
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

		bootstrap, err := mintBootstrapToken(ctx, store, entityID, ec.providerID, expiresAt)
		if err != nil {
			return err
		}

		existing.TokenHashed = HashTokenValue(plain)
		existing.ExpiresAt = expiresAt
		existing.BootstrapTokenID = &bootstrap.ID
		existing.PlainToken = plain
		existing.PlainBootstrapToken = bootstrap.PlainValue
		ec.hydrate(existing)

		if err := store.InstallTokenRepo().Save(ctx, existing); err != nil {
			return err
		}

		event, err := NewEvent(
			eventType(entityType, "regenerated"),
			WithInitiatorCtx(ctx),
			ec.eventOpt,
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

func (c *installTokenCommander) Revoke(ctx context.Context, entityType InstallTokenEntityType, entityID properties.UUID) error {
	return c.store.Atomic(ctx, func(store Store) error {
		ec, err := loadEntityCtx(ctx, store, entityType, entityID)
		if err != nil {
			return err
		}

		existing, err := store.InstallTokenRepo().GetByEntity(ctx, entityType, entityID)
		if err != nil {
			return err
		}
		if existing.BootstrapTokenID != nil {
			if err := store.TokenRepo().Delete(ctx, *existing.BootstrapTokenID); err != nil && !errors.As(err, &NotFoundError{}) {
				return err
			}
		}
		if err := store.InstallTokenRepo().DeleteByEntity(ctx, entityType, entityID); err != nil {
			return err
		}

		event, err := NewEvent(
			eventType(entityType, "revoked"),
			WithInitiatorCtx(ctx),
			ec.eventOpt,
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
