// Service property schema engine composition and factory
package domain

import "github.com/fulcrumproject/core/pkg/schema"

// buildServicePropertyValidatorRegistry creates a registry of property validators
// combining generic validators from pkg/schema with domain-specific validators
func buildServicePropertyValidatorRegistry() map[string]schema.PropertyValidator[ServicePropertyContext] {
	return map[string]schema.PropertyValidator[ServicePropertyContext]{
		// Generic validators from pkg/schema
		"minLength": &schema.MinLengthValidator[ServicePropertyContext]{},
		"maxLength": &schema.MaxLengthValidator[ServicePropertyContext]{},
		"pattern":   &schema.PatternValidator[ServicePropertyContext]{},
		"enum":      &schema.EnumValidator[ServicePropertyContext]{},
		"min":       &schema.MinValidator[ServicePropertyContext]{},
		"max":       &schema.MaxValidator[ServicePropertyContext]{},
		"minItems":  &schema.MinItemsValidator[ServicePropertyContext]{},
		"maxItems":  &schema.MaxItemsValidator[ServicePropertyContext]{},

		// Domain-specific validators
		"source":        &SourceValidator{},
		"mutable":       &MutableValidator{},
		"serviceOption": NewServiceOptionValidator(),
	}
}

// buildServicePropertySchemaValidatorRegistry creates a registry of schema-level validators
func buildServicePropertySchemaValidatorRegistry() map[string]schema.SchemaValidator[ServicePropertyContext] {
	return map[string]schema.SchemaValidator[ServicePropertyContext]{
		"exactlyOne": &schema.ExactlyOneValidator[ServicePropertyContext]{},
	}
}

// buildServicePropertyGeneratorRegistry creates a registry of generators
func buildServicePropertyGeneratorRegistry() map[string]schema.Generator[ServicePropertyContext] {
	return map[string]schema.Generator[ServicePropertyContext]{
		"pool": NewSchemaPoolGenerator(),
	}
}

// NewServicePropertyEngine creates a new schema engine configured for service property validation.
// It composes generic validators, domain validators, and domain generators with vault integration.
// Note: Store is no longer passed here - validators/generators access it via ServicePropertyContext.
func NewServicePropertyEngine(vault schema.Vault) *schema.Engine[ServicePropertyContext] {
	return schema.NewEngine(
		buildServicePropertyValidatorRegistry(),
		buildServicePropertySchemaValidatorRegistry(),
		buildServicePropertyGeneratorRegistry(),
		vault,
	)
}
