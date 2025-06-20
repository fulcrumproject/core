package auth

import (
	"fmt"
)

// AuthorizationRule represents a single authorization rule with roles, action, and object
type AuthorizationRule struct {
	Roles  []Role
	Action Action
	Object ObjectType
}

// RuleBasedAuthorizer implements the Authorizer interface using a set of predefined rules
type RuleBasedAuthorizer struct {
	rules []AuthorizationRule
}

// NewRuleBasedAuthorizer creates a new RuleBasedAuthorizer with the given rules
func NewRuleBasedAuthorizer(rules []AuthorizationRule) *RuleBasedAuthorizer {
	return &RuleBasedAuthorizer{
		rules: rules,
	}
}

// Authorize checks if the given identity is authorized to perform the action on the object
// It matches against the predefined rules based on the identity's roles
func (a *RuleBasedAuthorizer) Authorize(identity *Identity, action Action, object ObjectType, objectContext ObjectScope) error {
	// Check if the object context matches the identity (for context-specific authorization)
	if objectContext != nil && !objectContext.Matches(identity) {
		return fmt.Errorf("access denied: object context does not match identity")
	}

	// Check if any of the identity's roles match the authorization rules
	for _, rule := range a.rules {
		if rule.Action == action && rule.Object == object {
			// Check if identity has any of the required roles
			for _, requiredRole := range rule.Roles {
				if identity.HasRole(requiredRole) {
					return nil // Authorization successful
				}
			}
		}
	}

	return fmt.Errorf("access denied: no matching authorization rule found for action '%s' on object '%s'", action, object)
}
