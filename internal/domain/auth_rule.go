package domain

import (
	"context"
	"errors"
	"fmt"
)

// AuthRule represents a rule that maps a subject-action pair to the roles that have permission
type AuthRule struct {
	Subject AuthSubject
	Action  AuthAction
	Roles   []AuthRole
}

// String returns the string representation of the rule
func (r AuthRule) String() string {
	return string(r.Subject) + ":" + string(r.Action)
}

// RuleAuthorizer implements the Authorizer interface with a rule-based approach
type RuleAuthorizer struct {
	rules map[AuthSubject]map[AuthAction]map[AuthRole]bool
}

// NewRuleAuthorizer creates a new SimpleAuthorizer with custom authz rules
func NewRuleAuthorizer(authzRules ...AuthRule) *RuleAuthorizer {
	rules := make(map[AuthSubject]map[AuthAction]map[AuthRole]bool)

	// Convert the auth rules to a nested map for efficient lookup
	for _, rule := range authzRules {
		// Initialize subject map if it doesn't exist
		if _, exists := rules[rule.Subject]; !exists {
			rules[rule.Subject] = make(map[AuthAction]map[AuthRole]bool)
		}

		// Initialize action map if it doesn't exist
		if _, exists := rules[rule.Subject][rule.Action]; !exists {
			rules[rule.Subject][rule.Action] = make(map[AuthRole]bool)
		}

		// Add roles to the map
		for _, role := range rule.Roles {
			rules[rule.Subject][rule.Action][role] = true
		}
	}

	return &RuleAuthorizer{
		rules: rules,
	}
}

// NewDefaultRuleAuthorizer creates a new RuleAuthorizer with the default auth rules
func NewDefaultRuleAuthorizer() *RuleAuthorizer {
	return NewRuleAuthorizer(defaultAuthzRules...)
}

// Authorize checks if the given identity has permission to perform the action on the subject within the provided context
func (a *RuleAuthorizer) Authorize(identity AuthIdentity, subject AuthSubject, action AuthAction, targetScope *AuthTargetScope) error {
	if identity == nil {
		return errors.New("missing identity for authorization")
	}

	// Check if the role has permission for the subject:action pair
	if !a.hasPermission(subject, action, identity.Role()) {
		return fmt.Errorf("role %s does not have permission %s on %s", identity.Role(), action, subject)
	}

	// Check the scope
	err := ValidateAuthScope(identity, targetScope)
	if err != nil {
		return err
	}

	return nil
}

func (a *RuleAuthorizer) AuthorizeCtx(ctx context.Context, subject AuthSubject, action AuthAction, targetScope *AuthTargetScope) error {
	id := MustGetAuthIdentity(ctx)
	return a.Authorize(id, subject, action, targetScope)
}

// hasPermission checks if a role has permission for a subject-action pair
func (a *RuleAuthorizer) hasPermission(subject AuthSubject, action AuthAction, role AuthRole) bool {
	// Check if we have entries for this subject
	if actionMap, subjectExists := a.rules[subject]; subjectExists {
		// Check if we have entries for this action
		if roleMap, actionExists := actionMap[action]; actionExists {
			// Check if this role is allowed
			return roleMap[role]
		}
	}
	return false
}

// Default authorization rules for the system
var defaultAuthzRules = []AuthRule{
	// Participant permissions
	{Subject: SubjectParticipant, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectParticipant, Action: ActionCreate, Roles: []AuthRole{RoleFulcrumAdmin}},
	{Subject: SubjectParticipant, Action: ActionUpdate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectParticipant, Action: ActionDelete, Roles: []AuthRole{RoleFulcrumAdmin}},

	// Agent permissions
	{Subject: SubjectAgent, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant, RoleAgent}},
	{Subject: SubjectAgent, Action: ActionCreate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectAgent, Action: ActionUpdate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectAgent, Action: ActionDelete, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectAgent, Action: ActionUpdateState, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant, RoleAgent}},

	// AgentType permissions
	{Subject: SubjectAgentType, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant, RoleAgent}},

	// Service permissions
	{Subject: SubjectService, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant, RoleAgent}},
	{Subject: SubjectService, Action: ActionCreate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectService, Action: ActionUpdate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectService, Action: ActionStart, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectService, Action: ActionStop, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectService, Action: ActionDelete, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},

	// ServiceType permissions
	{Subject: SubjectServiceType, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant, RoleAgent}},

	// ServiceGroup permissions
	{Subject: SubjectServiceGroup, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectServiceGroup, Action: ActionCreate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectServiceGroup, Action: ActionUpdate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectServiceGroup, Action: ActionDelete, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},

	// Job permissions
	{Subject: SubjectJob, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant, RoleAgent}},
	{Subject: SubjectJob, Action: ActionClaim, Roles: []AuthRole{RoleAgent}},
	{Subject: SubjectJob, Action: ActionComplete, Roles: []AuthRole{RoleAgent}},
	{Subject: SubjectJob, Action: ActionFail, Roles: []AuthRole{RoleAgent}},
	{Subject: SubjectJob, Action: ActionListPending, Roles: []AuthRole{RoleAgent}},

	// MetricType permissions
	{Subject: SubjectMetricType, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant, RoleAgent}},
	{Subject: SubjectMetricType, Action: ActionCreate, Roles: []AuthRole{RoleFulcrumAdmin}},
	{Subject: SubjectMetricType, Action: ActionUpdate, Roles: []AuthRole{RoleFulcrumAdmin}},
	{Subject: SubjectMetricType, Action: ActionDelete, Roles: []AuthRole{RoleFulcrumAdmin}},

	// MetricEntry permissions
	{Subject: SubjectMetricEntry, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectMetricEntry, Action: ActionCreate, Roles: []AuthRole{RoleAgent}},

	// AuditEntry permissions
	{Subject: SubjectAuditEntry, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},

	// Token permissions
	{Subject: SubjectToken, Action: ActionRead, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectToken, Action: ActionCreate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectToken, Action: ActionUpdate, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectToken, Action: ActionDelete, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
	{Subject: SubjectToken, Action: ActionGenerateToken, Roles: []AuthRole{RoleFulcrumAdmin, RoleParticipant}},
}
