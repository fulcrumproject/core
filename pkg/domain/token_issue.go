// Token issuance certifies persisted Core principals and optional service access.
// Generic audit records never include JWTs, opaque extensions or private keys.
package domain

import (
	"context"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/properties"
)

const EventTypeTokenIssued EventType = "token.issued"

// TokenPrincipal is trusted Core context, not caller-selected claims.
// An agent's participant is its persisted provider; Service is optional.
type TokenPrincipal struct {
	Role          auth.Role
	ParticipantID properties.UUID
	AgentID       *properties.UUID
	Service       *Service
	Relation      string
}

type TokenIssueCommander struct {
	store Store
	authz authz.Authorizer
}

func NewTokenIssueCommander(store Store, authorizer authz.Authorizer) *TokenIssueCommander {
	return &TokenIssueCommander{store: store, authz: authorizer}
}

// Resolve validates authenticated scope against persisted domain relationships.
// Existing native service read authorization is mandatory before adding context.
func (c *TokenIssueCommander) Resolve(ctx context.Context, serviceID *properties.UUID) (*TokenPrincipal, error) {
	id := auth.MustGetIdentity(ctx)
	denied := func() error {
		return NewUnauthorizedErrorf("a persisted participant or agent with consistent scope is required")
	}
	if id.Scope.ParticipantID == nil || *id.Scope.ParticipantID == (properties.UUID{}) {
		return nil, denied()
	}
	switch id.Role {
	case auth.RoleParticipant:
		if id.Scope.AgentID != nil {
			return nil, denied()
		}
	case auth.RoleAgent:
		if id.Scope.AgentID == nil || *id.Scope.AgentID == (properties.UUID{}) {
			return nil, denied()
		}
	default:
		return nil, denied()
	}
	participant, err := c.store.ParticipantRepo().Get(ctx, *id.Scope.ParticipantID)
	if err != nil {
		return nil, err
	}
	if participant == nil || participant.ID != *id.Scope.ParticipantID {
		return nil, denied()
	}
	principal := &TokenPrincipal{Role: id.Role, ParticipantID: participant.ID}
	if id.Role == auth.RoleAgent {
		agent, err := c.store.AgentRepo().Get(ctx, *id.Scope.AgentID)
		if err != nil {
			return nil, err
		}
		if agent == nil || agent.ID != *id.Scope.AgentID || agent.ProviderID != participant.ID {
			return nil, denied()
		}
		principal.AgentID = &agent.ID
	}
	if serviceID == nil {
		return principal, nil
	}
	scope, err := c.store.ServiceRepo().AuthScope(ctx, *serviceID)
	if err != nil {
		return nil, err
	}
	if scope == nil || c.authz.Authorize(id, authz.ActionRead, authz.ObjectTypeService, scope) != nil {
		return nil, NewUnauthorizedErrorf("service access denied")
	}
	svc, err := c.store.ServiceRepo().Get(ctx, *serviceID)
	if err != nil {
		return nil, err
	}
	if svc == nil || svc.ID != *serviceID {
		return nil, NewUnauthorizedErrorf("service context does not match persisted scope")
	}
	// Recheck the loaded record as well as the repository authorization scope.
	scope = &authz.DefaultObjectScope{ProviderID: &svc.ProviderID, ConsumerID: &svc.ConsumerID, AgentID: &svc.AgentID}
	if err := c.authz.Authorize(id, authz.ActionRead, authz.ObjectTypeService, scope); err != nil {
		return nil, NewUnauthorizedErrorf("service access denied")
	}
	if principal.AgentID != nil {
		if svc.AgentID != *principal.AgentID || svc.ProviderID != principal.ParticipantID {
			return nil, NewUnauthorizedErrorf("service is not assigned to the authenticated agent and provider")
		}
		principal.Relation = "provider"
	} else {
		switch principal.ParticipantID {
		case svc.ProviderID:
			principal.Relation = "provider"
		case svc.ConsumerID:
			principal.Relation = "consumer"
		default:
			return nil, NewUnauthorizedErrorf("participant is neither service provider nor consumer")
		}
	}
	principal.Service = svc
	return principal, nil
}

// Record must succeed before the signed token leaves the issuance handler.
// Only certified identity/context and requested audience are audited.
func (c *TokenIssueCommander) Record(ctx context.Context, p *TokenPrincipal, audience string) error {
	event, err := NewEvent(EventTypeTokenIssued, WithInitiatorCtx(ctx))
	if err != nil {
		return err
	}
	event.ParticipantID = &p.ParticipantID
	event.EntityID = &p.ParticipantID
	if p.AgentID != nil {
		event.AgentID = p.AgentID
		event.ProviderID = &p.ParticipantID
		event.EntityID = p.AgentID
	}
	event.Payload = properties.JSON{"role": string(p.Role), "audience": audience}
	if p.Service != nil {
		if err := WithService(p.Service)(event); err != nil {
			return err
		}
		event.Payload["relation"] = p.Relation
	}
	return c.store.EventRepo().Create(ctx, event)
}
