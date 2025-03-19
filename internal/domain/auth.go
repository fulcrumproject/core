package domain

import (
	"context"
	"fmt"
)

type authContextKey string

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

// AuthAction defines the operation performed on a resource
type AuthAction string

const (
	// Standard CRUD actions
	ActionCreate AuthAction = "create"
	ActionRead   AuthAction = "read"
	ActionUpdate AuthAction = "update"
	ActionDelete AuthAction = "delete"
	ActionList   AuthAction = "list"

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

// GetAuthIdentity retrieves the authenticated identity from the request context
func GetAuthIdentity(ctx context.Context) AuthIdentity {
	id, _ := ctx.Value(identityContextKey).(AuthIdentity)
	return id
}

type Authorizer interface {
	Authorize(ctx context.Context, identity AuthIdentity, subject AuthSubject, action AuthAction) error
}

func ValidateAuthScope(ctx context.Context, target *AuthScope) error {
	if target == nil {
		return NewUnauthorizedErrorf("nil authorization target scope")
	}

	id := GetAuthIdentity(ctx)
	if id == nil {
		return nil
	}
	source := id.Scope()
	if source == nil {
		return NewUnauthorizedErrorf("nil authorization source scope")
	}

	// If all fields are nil in the caller scope, it has unrestricted access (admin)
	if source.ProviderID == nil && source.AgentID == nil && source.BrokerID == nil {
		return nil
	}

	// Provider check: If source requires a provider, caller must have same provider
	if source.ProviderID != nil && (target.ProviderID == nil || *target.ProviderID != *source.ProviderID) {
		return NewUnauthorizedErrorf("provider out of authorization scope")
	}

	// Agent check: If source requires an agent, caller must have same agent
	if source.AgentID != nil && (target.AgentID == nil || *target.AgentID != *source.AgentID) {
		return NewUnauthorizedErrorf("agent out of authorization scope")
	}

	// Broker check: If source requires a broker, caller must have same broker
	if source.BrokerID != nil && (target.BrokerID == nil || *target.BrokerID != *source.BrokerID) {
		return NewUnauthorizedErrorf("broker out of authorization scope")
	}

	return nil
}
