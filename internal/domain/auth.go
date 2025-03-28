package domain

import (
	"context"
	"errors"
	"fmt"
)

type authContextKey string

var EmptyAuthScope = AuthScope{}

const (
	identityContextKey = authContextKey("identity")
)

type AuthRole string

const (
	RoleFulcrumAdmin  AuthRole = "fulcrum_admin"
	RoleProviderAdmin AuthRole = "provider_admin"
	RoleBroker        AuthRole = "broker"
	RoleAgent         AuthRole = "agent"
)

// Validate ensures the AuthRole is one of the predefined values
func (r AuthRole) Validate() error {
	switch r {
	case RoleFulcrumAdmin, RoleProviderAdmin, RoleBroker, RoleAgent:
		return nil
	default:
		return fmt.Errorf("invalid auth role: %s", r)
	}
}

// AuthSubject defines the resource type the action is performed on
type AuthSubject string

const (
	SubjectProvider     AuthSubject = "provider"
	SubjectBroker       AuthSubject = "broker"
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
	case SubjectProvider, SubjectBroker, SubjectAgent, SubjectAgentType,
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
	Scope() *AuthScope
}

// AuthScope contains additional information for authorization decisions
type AuthScope struct {
	ProviderID *UUID
	AgentID    *UUID
	BrokerID   *UUID
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
	Authorize(identity AuthIdentity, subject AuthSubject, action AuthAction, scope *AuthScope) error
	AuthorizeCtx(ctx context.Context, subject AuthSubject, action AuthAction, scope *AuthScope) error
}

type AuthScopeRetriever interface {
	AuthScope(ctx context.Context, id UUID) (*AuthScope, error)
}

func ValidateAuthScope(id AuthIdentity, target *AuthScope) error {
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
	if source.ProviderID == nil && source.AgentID == nil && source.BrokerID == nil {
		return nil
	}

	// Provider check: If source requires a provider, caller must have same provider
	if source.ProviderID != nil && target.ProviderID != nil && *target.ProviderID != *source.ProviderID {
		return errors.New("invalid provider authorization scope")
	}

	// Agent check: If source requires an agent, caller must have same agent
	if source.AgentID != nil && target.AgentID != nil && *target.AgentID != *source.AgentID {
		return errors.New("invalid agent authorization scope")
	}

	// Broker check: If source requires a broker, caller must have same broker
	if source.BrokerID != nil && target.BrokerID != nil && *target.BrokerID != *source.BrokerID {
		return errors.New("invalid broker authorization scope")
	}

	return nil
}
