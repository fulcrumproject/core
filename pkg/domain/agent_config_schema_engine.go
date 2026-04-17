// Agent configuration schema engine composition and factory
package domain

import (
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
)

// AgentConfigContext carries the contextual data the agent config schema engine needs.
// Store exposes repositories to generators (e.g. the pool generator looks up AgentPool +
// allocates AgentPoolValue rows); AgentID stamps allocated values to the owning agent.
type AgentConfigContext struct {
	Store   Store
	AgentID *properties.UUID
}

// buildAgentConfigValidatorRegistry creates a registry of property validators
// Only generic validators - no domain-specific ones (no source/mutable for agents)
func buildAgentConfigValidatorRegistry() map[string]schema.PropertyValidator[AgentConfigContext] {
	return map[string]schema.PropertyValidator[AgentConfigContext]{
		// Generic validators from pkg/schema
		"minLength": &schema.MinLengthValidator[AgentConfigContext]{},
		"maxLength": &schema.MaxLengthValidator[AgentConfigContext]{},
		"pattern":   &schema.PatternValidator[AgentConfigContext]{},
		"enum":      &schema.EnumValidator[AgentConfigContext]{},
		"min":       &schema.MinValidator[AgentConfigContext]{},
		"max":       &schema.MaxValidator[AgentConfigContext]{},
		"minItems":  &schema.MinItemsValidator[AgentConfigContext]{},
		"maxItems":  &schema.MaxItemsValidator[AgentConfigContext]{},

		// Note: NO SourceValidator or MutableValidator
		// Agent config doesn't have different actors or lifecycle states
	}
}

// buildAgentConfigAuthorizerRegistry creates an empty authorizer registry
// Agent config has no authorization rules - all fields are user input
func buildAgentConfigAuthorizerRegistry() map[string]schema.Authorizer[AgentConfigContext] {
	return map[string]schema.Authorizer[AgentConfigContext]{
		// Empty - no authorization for agent config
	}
}

// buildAgentConfigSchemaValidatorRegistry creates a registry of schema-level validators
func buildAgentConfigSchemaValidatorRegistry() map[string]schema.SchemaValidator[AgentConfigContext] {
	return map[string]schema.SchemaValidator[AgentConfigContext]{
		"exactlyOne": &schema.ExactlyOneValidator[AgentConfigContext]{},
	}
}

// buildAgentConfigGeneratorRegistry registers the generators available to agent configuration
// schemas. The "pool" generator auto-allocates an AgentPoolValue at agent create time.
func buildAgentConfigGeneratorRegistry() map[string]schema.Generator[AgentConfigContext] {
	return map[string]schema.Generator[AgentConfigContext]{
		"pool": NewSchemaAgentPoolGenerator(),
	}
}

// NewAgentConfigSchemaEngine creates a new schema engine configured for agent configuration validation.
// It composes validators and schema validators with vault integration.
func NewAgentConfigSchemaEngine(vault schema.Vault) *schema.Engine[AgentConfigContext] {
	return schema.NewEngine(
		buildAgentConfigAuthorizerRegistry(),
		buildAgentConfigValidatorRegistry(),
		buildAgentConfigSchemaValidatorRegistry(),
		buildAgentConfigGeneratorRegistry(),
		vault,
	)
}
