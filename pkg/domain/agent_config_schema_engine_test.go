package domain

import (
	"testing"
)

func TestNewAgentConfigSchemaEngine(t *testing.T) {
	// Test that engine can be created without vault
	engine := NewAgentConfigSchemaEngine(nil)
	if engine == nil {
		t.Fatal("Expected engine to be created")
	}
}

func TestAgentConfigValidatorRegistry(t *testing.T) {
	validators := buildAgentConfigValidatorRegistry()

	// Should have generic validators
	expectedValidators := []string{
		"minLength", "maxLength", "pattern", "enum",
		"min", "max", "minItems", "maxItems",
	}

	for _, v := range expectedValidators {
		if _, ok := validators[v]; !ok {
			t.Errorf("Expected validator %s to be registered", v)
		}
	}

	// Should NOT have domain-specific validators
	unexpectedValidators := []string{"serviceOption", "serviceReference"}
	for _, v := range unexpectedValidators {
		if _, ok := validators[v]; ok {
			t.Errorf("Validator %s should not be registered for agent config", v)
		}
	}
}

func TestAgentConfigAuthorizerRegistry(t *testing.T) {
	authorizers := buildAgentConfigAuthorizerRegistry()

	// Should be empty - no authorization for agent config
	if len(authorizers) != 0 {
		t.Errorf("Expected empty authorizer registry, got %d authorizers", len(authorizers))
	}
}

func TestAgentConfigSchemaValidatorRegistry(t *testing.T) {
	validators := buildAgentConfigSchemaValidatorRegistry()

	// Should have exactlyOne
	if _, ok := validators["exactlyOne"]; !ok {
		t.Error("Expected exactlyOne schema validator to be registered")
	}
}

func TestAgentConfigGeneratorRegistry(t *testing.T) {
	generators := buildAgentConfigGeneratorRegistry()

	// Should be empty - no generators for agent config
	if len(generators) != 0 {
		t.Errorf("Expected empty generator registry, got %d generators", len(generators))
	}
}
