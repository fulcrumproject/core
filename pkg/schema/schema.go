// Package schema provides a generalized schema management system for defining,
// validating, and processing structured data with pluggable validators and generators.
package schema

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Operation represents the type of write operation being performed
type Operation string

const (
	OperationCreate Operation = "create"
	OperationUpdate Operation = "update"
)

// Schema defines the structure and validation rules for a set of properties
type Schema struct {
	Properties map[string]PropertyDefinition `json:"properties"` // Property definitions
	Validators []SchemaValidatorConfig       `json:"validators"` // Cross-field validators
}

// Value implements driver.Valuer interface for database serialization
func (s Schema) Value() (driver.Value, error) {
	// Always marshal, even if empty - this ensures non-null JSON in DB
	return json.Marshal(s)
}

// Scan implements sql.Scanner interface for database deserialization
func (s *Schema) Scan(value any) error {
	if value == nil {
		*s = Schema{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal Schema value: %v", value)
	}

	return json.Unmarshal(bytes, s)
}

// SchemaValidatorConfig defines a schema-level validator configuration
type SchemaValidatorConfig struct {
	Type   string         `json:"type"`   // "exactlyOne", etc.
	Config map[string]any `json:"config"` // Validator-specific configuration
}

// PropertyDefinition defines a single property with all its rules
type PropertyDefinition struct {
	// Basic metadata
	Type      string `json:"type"`      // string, integer, number, boolean, object, array, json
	Label     string `json:"label"`     // Human-readable name
	Required  bool   `json:"required"`  // Must be present
	Immutable bool   `json:"immutable"` // Cannot be updated after creation

	// Authorization rules (all must pass - AND logic)
	Authorizers []AuthorizerConfig `json:"authorizers,omitempty"`

	// Default value (applied when no value provided)
	Default any `json:"default,omitempty"`

	// Secret handling (vault integration)
	Secret *SecretConfig `json:"secret,omitempty"`

	// Value generation (zero or one)
	Generator *GeneratorConfig `json:"generator,omitempty"`

	// Validation rules (execute in order)
	Validators []ValidatorConfig `json:"validators,omitempty"`

	// Recursive structures
	Properties map[string]PropertyDefinition `json:"properties,omitempty"` // For type: object
	Items      *PropertyDefinition           `json:"items,omitempty"`      // For type: array
}

// SecretConfig defines secret handling configuration
type SecretConfig struct {
	Type string `json:"type"` // "persistent" or "ephemeral"
}

// GeneratorConfig defines a value generator configuration
type GeneratorConfig struct {
	Type   string         `json:"type"`   // "pool", "computed", "function"
	Config map[string]any `json:"config"` // Type-specific configuration
}

// ValidatorConfig defines a validation rule configuration
type ValidatorConfig struct {
	Type   string         `json:"type"`   // "source", "mutable", "minLength", "pattern", etc.
	Config map[string]any `json:"config"` // Type-specific configuration
}

// AuthorizerConfig defines an authorization rule configuration
type AuthorizerConfig struct {
	Type   string         `json:"type"`   // "actor", "state", etc.
	Config map[string]any `json:"config"` // Type-specific configuration
}

// Authorizer checks if an operation is authorized
// C is the context type specific to the domain (e.g., ServicePropertyContext)
type Authorizer[C any] interface {
	// Authorize checks if the actor can perform the operation
	// hasNewValue: true if a new value is being provided
	Authorize(ctx context.Context, schemaCtx C, operation Operation, propPath string, hasNewValue bool, config map[string]any) error

	// ValidateConfig checks if the authorizer configuration is valid
	ValidateConfig(propPath string, config map[string]any) error
}

// PropertyValidator validates property values and operations
// C is the context type specific to the domain (e.g., ServicePropertyContext)
type PropertyValidator[C any] interface {
	// Validate checks the property value and operation
	// oldValue: previous value (nil for create)
	// newValue: proposed new value (nil if not provided)
	Validate(ctx context.Context, schemaCtx C, operation Operation, propPath string, oldValue, newValue any, config map[string]any) error

	// ValidateConfig checks if the validator configuration is valid
	ValidateConfig(propPath string, config map[string]any) error
}

// SchemaValidator validates relationships between multiple properties
// C is the context type specific to the domain (e.g., ServicePropertyContext)
type SchemaValidator[C any] interface {
	// Validate checks cross-field rules across all properties
	// oldProperties: previous property values (nil for create)
	// newProperties: processed property values after individual validation
	Validate(ctx context.Context, schemaCtx C, operation Operation, oldProperties, newProperties map[string]any, config map[string]any) error

	// ValidateConfig checks if the validator configuration is valid
	ValidateConfig(config map[string]any) error
}

// Generator produces values for properties
// C is the context type specific to the domain (e.g., ServicePropertyContext)
type Generator[C any] interface {
	// Generate produces a value for the property
	// currentValue: existing value (nil for create, actual value for update)
	// Returns (value, generated=true) or (nil, generated=false) if no value produced
	Generate(ctx context.Context, schemaCtx C, propPath string, currentValue any, config map[string]any) (any, bool, error)

	// ValidateConfig checks if the generator configuration is valid
	ValidateConfig(propPath string, config map[string]any) error
}
