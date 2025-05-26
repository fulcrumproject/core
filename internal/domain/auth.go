package domain

import (
	"context"
	"errors"
	"fmt"
)

type authContextKey string

var (
	EmptyAuthTargetScope   = AuthTargetScope{}
	EmptyAuthIdentityScope = AuthIdentityScope{}
)

const (
	identityContextKey = authContextKey("identity")
)

type AuthRole string

const (
	RoleFulcrumAdmin AuthRole = "fulcrum_admin"
	RoleParticipant  AuthRole = "participant"
	RoleAgent        AuthRole = "agent"
)

// Validate ensures the AuthRole is one of the predefined values
func (r AuthRole) Validate() error {
	switch r {
	case RoleFulcrumAdmin, RoleParticipant, RoleAgent:
		return nil
	default:
		return fmt.Errorf("invalid auth role: %s", r)
	}
}

// AuthSubject defines the resource type the action is performed on
type AuthSubject string

const (
	SubjectParticipant  AuthSubject = "participant"
	SubjectAgent        AuthSubject = "agent"
	SubjectAgentType    AuthSubject = "agent_type"
	SubjectService      AuthSubject = "service"
	SubjectServiceType  AuthSubject = "service_type"
	SubjectServiceGroup AuthSubject = "service_group"
	SubjectJob          AuthSubject = "job"
	SubjectMetricType   AuthSubject = "metric_type"
	SubjectMetricEntry  AuthSubject = "metric_entry"
	SubjectAuditEntry   AuthSubject = "audit_entry"
	SubjectToken        AuthSubject = "token"
)

// Validate ensures the AuthSubject is one of the predefined values
func (s AuthSubject) Validate() error {
	switch s {
	case SubjectParticipant, SubjectAgent, SubjectAgentType,
		SubjectService, SubjectServiceType, SubjectServiceGroup,
		SubjectJob, SubjectMetricType, SubjectMetricEntry,
		SubjectAuditEntry, SubjectToken:
		return nil
	default:
		return fmt.Errorf("invalid auth subject: %s", s)
	}
}

// AuthAction defines the operation performed on a resource
type AuthAction string

const (
	// Standard CRUD actions
	ActionCreate AuthAction = "create"
	ActionRead   AuthAction = "read"
	ActionUpdate AuthAction = "update"
	ActionDelete AuthAction = "delete"

	// Special actions
	ActionUpdateState   AuthAction = "update_state"
	ActionGenerateToken AuthAction = "generate_token"
	ActionStart         AuthAction = "start"
	ActionStop          AuthAction = "stop"
	ActionClaim         AuthAction = "claim"
	ActionComplete      AuthAction = "complete"
	ActionFail          AuthAction = "fail"
	ActionListPending   AuthAction = "list_pending"
)

// Validate ensures the AuthAction is one of the predefined values
func (a AuthAction) Validate() error {
	switch a {
	case ActionCreate, ActionRead, ActionUpdate, ActionDelete,
		ActionUpdateState, ActionGenerateToken, ActionStart, ActionStop,
		ActionClaim, ActionComplete, ActionFail, ActionListPending:
		return nil
	default:
		return fmt.Errorf("invalid auth action: %s", a)
	}
}

type AuthIdentity interface {
	ID() UUID
	Name() string
	Role() AuthRole
	IsRole(role AuthRole) bool
	Scope() *AuthIdentityScope
}

// AuthScope contains additional information for authorization decisions
type AuthIdentityScope struct {
	ParticipantID *UUID
	AgentID       *UUID
}

// AuthTargetScope contains additional information for authorization decisions
type AuthTargetScope struct {
	ParticipantID *UUID
	ProviderID    *UUID
	ConsumerID    *UUID
	AgentID       *UUID
}

type Authenticator interface {
	Authenticate(ctx context.Context, token string) AuthIdentity
}

// WithAuthIdentity adds to the context the identity
func WithAuthIdentity(ctx context.Context, id AuthIdentity) context.Context {
	return context.WithValue(ctx, identityContextKey, id)
}

// MustGetAuthIdentity retrieves the authenticated identity from the request context
func MustGetAuthIdentity(ctx context.Context) AuthIdentity {
	id, ok := ctx.Value(identityContextKey).(AuthIdentity)
	if !ok {
		panic("cannot find identity in context")
	}
	return id
}

type Authorizer interface {
	Authorize(identity AuthIdentity, subject AuthSubject, action AuthAction, scope *AuthTargetScope) error
	AuthorizeCtx(ctx context.Context, subject AuthSubject, action AuthAction, scope *AuthTargetScope) error
}

type AuthScopeRetriever interface {
	AuthScope(ctx context.Context, id UUID) (*AuthTargetScope, error)
}

func ValidateAuthScope(id AuthIdentity, target *AuthTargetScope) error {
	if id == nil {
		return errors.New("nil identity")
	}

	if target == nil {
		return errors.New("nil authorization target scope")
	}

	source := id.Scope()
	if source == nil {
		return errors.New("nil authorization source scope")
	}

	// If all fields are nil in the caller scope, it has unrestricted access (admin)
	if source.ParticipantID == nil && source.AgentID == nil {
		return nil
	}

	// If all fields are nil in the target scope, it has unrestricted access (admin)
	if target.ParticipantID == nil && target.ProviderID == nil && target.ConsumerID == nil && target.AgentID == nil {
		return nil
	}

	// Participant check: If source requires a participant, caller must have same participant or provider or consumer
	if source.ParticipantID != nil {
		if target.ParticipantID != nil && *target.ParticipantID == *source.ParticipantID {
			return nil
		}
		if target.ProviderID != nil && *target.ProviderID == *source.ParticipantID {
			return nil
		}
		if target.ConsumerID != nil && *target.ConsumerID == *source.ParticipantID {
			return nil
		}
	}

	// Agent check: If source requires an agent, caller must have same agent
	if source.AgentID != nil && target.AgentID != nil && *target.AgentID == *source.AgentID {
		return nil
	}

	return errors.New("invalid authorization scope")
}
