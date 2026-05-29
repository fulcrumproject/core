// Infrastructure configuration schema engine composition and factory.
package domain

import (
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/schema"
)

// InfrastructureConfigContext carries the contextual data the infrastructure
// config schema engine needs. Mirrors AgentConfigContext: Store exposes
// repositories to generators; InfrastructureID stamps allocated values to the
// owning infrastructure.
type InfrastructureConfigContext struct {
	Store                    Store
	InfrastructureID         *properties.UUID
	InfrastructureProviderID properties.UUID
}

func (c InfrastructureConfigContext) poolStore() Store { return c.Store }
func (c InfrastructureConfigContext) poolEntityType() ConfigPoolValueEntityType {
	return ConfigPoolValueEntityTypeInfrastructure
}
func (c InfrastructureConfigContext) poolEntityID() *properties.UUID { return c.InfrastructureID }
func (c InfrastructureConfigContext) poolProviderID() properties.UUID {
	return c.InfrastructureProviderID
}

// buildInfrastructureConfigValidatorRegistry registers the generic property
// validators. No domain-specific validators (no source/mutable for infra).
func buildInfrastructureConfigValidatorRegistry() map[string]schema.PropertyValidator[InfrastructureConfigContext] {
	return map[string]schema.PropertyValidator[InfrastructureConfigContext]{
		"minLength": &schema.MinLengthValidator[InfrastructureConfigContext]{},
		"maxLength": &schema.MaxLengthValidator[InfrastructureConfigContext]{},
		"pattern":   &schema.PatternValidator[InfrastructureConfigContext]{},
		"enum":      &schema.EnumValidator[InfrastructureConfigContext]{},
		"min":       &schema.MinValidator[InfrastructureConfigContext]{},
		"max":       &schema.MaxValidator[InfrastructureConfigContext]{},
		"minItems":  &schema.MinItemsValidator[InfrastructureConfigContext]{},
		"maxItems":  &schema.MaxItemsValidator[InfrastructureConfigContext]{},
	}
}

// buildInfrastructureConfigAuthorizerRegistry returns an empty authorizer set.
// Infrastructure config has no actor-based authorization rules in Phase 1.
func buildInfrastructureConfigAuthorizerRegistry() map[string]schema.Authorizer[InfrastructureConfigContext] {
	return map[string]schema.Authorizer[InfrastructureConfigContext]{}
}

// buildInfrastructureConfigSchemaValidatorRegistry registers schema-level
// validators (cross-property constraints).
func buildInfrastructureConfigSchemaValidatorRegistry() map[string]schema.SchemaValidator[InfrastructureConfigContext] {
	return map[string]schema.SchemaValidator[InfrastructureConfigContext]{
		"exactlyOne": &schema.ExactlyOneValidator[InfrastructureConfigContext]{},
	}
}

// buildInfrastructureConfigGeneratorRegistry registers generators. The "pool"
// generator auto-allocates a ConfigPoolValue at infrastructure create time.
func buildInfrastructureConfigGeneratorRegistry() map[string]schema.Generator[InfrastructureConfigContext] {
	return map[string]schema.Generator[InfrastructureConfigContext]{
		"pool": NewSchemaConfigPoolGenerator[InfrastructureConfigContext](),
	}
}

// NewInfrastructureConfigSchemaEngine creates a new schema engine configured
// for infrastructure configuration validation.
func NewInfrastructureConfigSchemaEngine(vault schema.Vault) *schema.Engine[InfrastructureConfigContext] {
	return schema.NewEngine(
		buildInfrastructureConfigAuthorizerRegistry(),
		buildInfrastructureConfigValidatorRegistry(),
		buildInfrastructureConfigSchemaValidatorRegistry(),
		buildInfrastructureConfigGeneratorRegistry(),
		vault,
	)
}
