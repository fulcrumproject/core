// Unit tests for ServiceType lifecycle schema validation
package domain

import (
	"testing"
)

func TestServiceType_ValidateLifecycle_Valid(t *testing.T) {
	validLifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "Stopped"},
			{Name: "Deleted"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
			{
				Name: "stop",
				Transitions: []LifecycleTransition{
					{From: "Running", To: "Stopped"},
				},
			},
			{
				Name: "delete",
				Transitions: []LifecycleTransition{
					{From: "Stopped", To: "Deleted"},
				},
			},
		},
		InitialState:   "New",
		TerminalStates: []string{"Deleted"},
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: validLifecycle,
	}

	err := st.ValidateLifecycle()
	if err != nil {
		t.Errorf("ValidateLifecycle() failed for valid lifecycle: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_NilLifecycle(t *testing.T) {
	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: nil,
	}

	err := st.ValidateLifecycle()
	if err != nil {
		t.Errorf("ValidateLifecycle() should not fail for nil lifecycle: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_EmptyStates(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for empty states")
	}
	if err.Error() != "lifecycle must have at least one state" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_EmptyStateName(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: ""},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for empty state name")
	}
	if err.Error() != "lifecycle state name cannot be empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_DuplicateStateName(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "New"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for duplicate state name")
	}
	if err.Error() != "duplicate lifecycle state name: New" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_EmptyInitialState(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
		},
		InitialState: "",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for empty initial state")
	}
	if err.Error() != "lifecycle must have an initial state" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_InvalidInitialState(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
		},
		InitialState: "NonExistent",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for invalid initial state")
	}
	expectedMsg := "lifecycle initial state \"NonExistent\" does not exist in states list"
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_InvalidTerminalState(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "Stopped"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
		},
		InitialState:   "New",
		TerminalStates: []string{"Deleted"},
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for invalid terminal state")
	}
	expectedMsg := "lifecycle terminal state \"Deleted\" does not exist in states list"
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_EmptyActions(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
		},
		Actions:      []LifecycleAction{},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for empty actions")
	}
	if err.Error() != "lifecycle must have at least one action" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_EmptyActionName(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
		},
		Actions: []LifecycleAction{
			{
				Name: "",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for empty action name")
	}
	if err.Error() != "lifecycle action name cannot be empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_DuplicateActionName(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "Stopped"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "Running", To: "Stopped"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for duplicate action name")
	}
	if err.Error() != "duplicate lifecycle action name: create" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_EmptyTransitions(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
		},
		Actions: []LifecycleAction{
			{
				Name:        "create",
				Transitions: []LifecycleTransition{},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for empty transitions")
	}
	if err.Error() != "lifecycle action \"create\" must have at least one transition" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_InvalidTransitionFromState(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "NonExistent", To: "Running"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for invalid from state")
	}
	expectedMsg := "lifecycle action \"create\" transition references invalid from state \"NonExistent\""
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_InvalidTransitionToState(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "NonExistent"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for invalid to state")
	}
	expectedMsg := "lifecycle action \"create\" transition references invalid to state \"NonExistent\""
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_InvalidErrorRegexp(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "Failed"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
					{From: "New", To: "Failed", OnError: true, OnErrorRegexp: "[invalid(regexp"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err == nil {
		t.Error("ValidateLifecycle() should fail for invalid error regexp")
	}
	// Check that error message contains the action name and mentions regexp
	if err.Error()[:len("lifecycle action \"create\" transition has invalid error regexp")] != "lifecycle action \"create\" transition has invalid error regexp" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_ValidErrorRegexp(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "Failed"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
					{From: "New", To: "Failed", OnError: true, OnErrorRegexp: "INSUFFICIENT_.*"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err != nil {
		t.Errorf("ValidateLifecycle() should not fail for valid error regexp: %v", err)
	}
}

func TestServiceType_ValidateLifecycle_ComplexLifecycle(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Creating"},
			{Name: "Created"},
			{Name: "Starting"},
			{Name: "Started"},
			{Name: "Stopping"},
			{Name: "Stopped"},
			{Name: "Deleting"},
			{Name: "Deleted"},
			{Name: "Failed"},
		},
		Actions: []LifecycleAction{
			{
				Name:              "create",
				RequestSchemaType: "properties",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Creating"},
					{From: "Creating", To: "Created"},
					{From: "Creating", To: "Failed", OnError: true},
				},
			},
			{
				Name: "start",
				Transitions: []LifecycleTransition{
					{From: "Created", To: "Starting"},
					{From: "Stopped", To: "Starting"},
					{From: "Starting", To: "Started"},
					{From: "Starting", To: "Failed", OnError: true, OnErrorRegexp: "NETWORK_.*"},
					{From: "Starting", To: "Stopped", OnError: true},
				},
			},
			{
				Name: "stop",
				Transitions: []LifecycleTransition{
					{From: "Started", To: "Stopping"},
					{From: "Stopping", To: "Stopped"},
				},
			},
			{
				Name:              "update",
				RequestSchemaType: "properties",
				Transitions: []LifecycleTransition{
					{From: "Stopped", To: "Stopped"},
					{From: "Started", To: "Started"},
				},
			},
			{
				Name: "delete",
				Transitions: []LifecycleTransition{
					{From: "Stopped", To: "Deleting"},
					{From: "Failed", To: "Deleting"},
					{From: "Deleting", To: "Deleted"},
				},
			},
		},
		InitialState:   "New",
		TerminalStates: []string{"Deleted", "Failed"},
	}

	st := &ServiceType{
		Name:            "ComplexService",
		LifecycleSchema: lifecycle,
	}

	err := st.ValidateLifecycle()
	if err != nil {
		t.Errorf("ValidateLifecycle() should not fail for complex valid lifecycle: %v", err)
	}
}

func TestServiceType_Validate_WithLifecycle(t *testing.T) {
	validLifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: validLifecycle,
	}

	err := st.Validate()
	if err != nil {
		t.Errorf("Validate() should not fail for service type with valid lifecycle: %v", err)
	}
}

func TestServiceType_Validate_WithInvalidLifecycle(t *testing.T) {
	invalidLifecycle := &LifecycleSchema{
		States: []LifecycleState{},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
				},
			},
		},
		InitialState: "New",
	}

	st := &ServiceType{
		Name:            "TestService",
		LifecycleSchema: invalidLifecycle,
	}

	err := st.Validate()
	if err == nil {
		t.Error("Validate() should fail for service type with invalid lifecycle")
	}
}
