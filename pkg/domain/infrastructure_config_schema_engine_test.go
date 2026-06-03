package domain

import "testing"

func TestNewInfrastructureConfigSchemaEngine(t *testing.T) {
	engine := NewInfrastructureConfigSchemaEngine(nil)
	if engine == nil {
		t.Fatal("Expected engine to be created")
	}
}

func TestInfrastructureConfigValidatorRegistry(t *testing.T) {
	validators := buildInfrastructureConfigValidatorRegistry()

	expectedValidators := []string{
		"minLength", "maxLength", "pattern", "enum",
		"min", "max", "minItems", "maxItems",
	}

	for _, v := range expectedValidators {
		if _, ok := validators[v]; !ok {
			t.Errorf("Expected validator %s to be registered", v)
		}
	}

	// Should NOT have service-specific validators
	unexpectedValidators := []string{"serviceOption", "serviceReference"}
	for _, v := range unexpectedValidators {
		if _, ok := validators[v]; ok {
			t.Errorf("Validator %s should not be registered for infrastructure config", v)
		}
	}
}

func TestInfrastructureConfigAuthorizerRegistry(t *testing.T) {
	authorizers := buildInfrastructureConfigAuthorizerRegistry()

	if len(authorizers) != 0 {
		t.Errorf("Expected empty authorizer registry, got %d authorizers", len(authorizers))
	}
}

func TestInfrastructureConfigSchemaValidatorRegistry(t *testing.T) {
	validators := buildInfrastructureConfigSchemaValidatorRegistry()

	if _, ok := validators["exactlyOne"]; !ok {
		t.Error("Expected exactlyOne schema validator to be registered")
	}
}

func TestInfrastructureConfigGeneratorRegistry(t *testing.T) {
	generators := buildInfrastructureConfigGeneratorRegistry()

	// The "pool" generator auto-allocates a ConfigPoolValue at infrastructure create time.
	if _, ok := generators["pool"]; !ok {
		t.Error("Expected pool generator to be registered")
	}
}
