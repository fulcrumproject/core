// Unit tests for service lifecycle state transition resolver
package domain

import (
	"testing"
)

func TestResolveNextState_SuccessTransition(t *testing.T) {
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
				Name: "stop",
				Transitions: []LifecycleTransition{
					{From: "Running", To: "Stopped"},
				},
			},
		},
		InitialState: "New",
	}

	nextState, err := lifecycle.ResolveNextState("New", "create", nil)
	if err != nil {
		t.Errorf("ResolveNextState() error = %v", err)
	}
	if nextState != "Running" {
		t.Errorf("expected next state 'Running', got '%s'", nextState)
	}

	nextState, err = lifecycle.ResolveNextState("Running", "stop", nil)
	if err != nil {
		t.Errorf("ResolveNextState() error = %v", err)
	}
	if nextState != "Stopped" {
		t.Errorf("expected next state 'Stopped', got '%s'", nextState)
	}
}

func TestResolveNextState_ErrorTransitionWithRegexp(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "Failed"},
			{Name: "NetworkError"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
					{From: "New", To: "NetworkError", OnError: true, OnErrorRegexp: "NETWORK_.*"},
					{From: "New", To: "Failed", OnError: true},
				},
			},
		},
		InitialState: "New",
	}

	// Test network error matches specific regexp
	errorCode := "NETWORK_TIMEOUT"
	nextState, err := lifecycle.ResolveNextState("New", "create", &errorCode)
	if err != nil {
		t.Errorf("ResolveNextState() error = %v", err)
	}
	if nextState != "NetworkError" {
		t.Errorf("expected next state 'NetworkError', got '%s'", nextState)
	}
}

func TestResolveNextState_ErrorTransitionWithoutRegexp(t *testing.T) {
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
					{From: "New", To: "Failed", OnError: true},
				},
			},
		},
		InitialState: "New",
	}

	// Test any error code matches transition without regexp
	errorCode := "ANY_ERROR"
	nextState, err := lifecycle.ResolveNextState("New", "create", &errorCode)
	if err != nil {
		t.Errorf("ResolveNextState() error = %v", err)
	}
	if nextState != "Failed" {
		t.Errorf("expected next state 'Failed', got '%s'", nextState)
	}
}

func TestResolveNextState_ErrorNoMatchingTransition(t *testing.T) {
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
		InitialState: "New",
	}

	// No error transition defined
	errorCode := "SOME_ERROR"
	_, err := lifecycle.ResolveNextState("New", "create", &errorCode)
	if err == nil {
		t.Error("ResolveNextState() should fail when no error transition exists")
	}
	expectedMsg := "no valid error transition found for action \"create\" from state \"New\" with error code \"SOME_ERROR\""
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestResolveNextState_ActionNotFound(t *testing.T) {
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
		InitialState: "New",
	}

	_, err := lifecycle.ResolveNextState("New", "nonexistent", nil)
	if err == nil {
		t.Error("ResolveNextState() should fail for nonexistent action")
	}
	expectedMsg := "action \"nonexistent\" not found in lifecycle schema"
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestResolveNextState_InvalidCurrentState(t *testing.T) {
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
		InitialState: "New",
	}

	_, err := lifecycle.ResolveNextState("Stopped", "create", nil)
	if err == nil {
		t.Error("ResolveNextState() should fail when current state has no valid transition")
	}
	expectedMsg := "no valid transition found for action \"create\" from state \"Stopped\""
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestResolveNextState_MultipleErrorTransitionsFirstMatchWins(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "NetworkError"},
			{Name: "Failed"},
		},
		Actions: []LifecycleAction{
			{
				Name: "create",
				Transitions: []LifecycleTransition{
					{From: "New", To: "Running"},
					{From: "New", To: "NetworkError", OnError: true, OnErrorRegexp: "NETWORK_.*"},
					{From: "New", To: "Failed", OnError: true}, // Catch-all
				},
			},
		},
		InitialState: "New",
	}

	// Network error should match first specific regexp
	errorCode := "NETWORK_TIMEOUT"
	nextState, err := lifecycle.ResolveNextState("New", "create", &errorCode)
	if err != nil {
		t.Errorf("ResolveNextState() error = %v", err)
	}
	if nextState != "NetworkError" {
		t.Errorf("expected 'NetworkError', got '%s'", nextState)
	}

	// Non-network error should fall through to catch-all
	errorCode = "DISK_FULL"
	nextState, err = lifecycle.ResolveNextState("New", "create", &errorCode)
	if err != nil {
		t.Errorf("ResolveNextState() error = %v", err)
	}
	if nextState != "Failed" {
		t.Errorf("expected 'Failed', got '%s'", nextState)
	}
}

func TestValidateActionAllowed_ActionAllowed(t *testing.T) {
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
				Name: "stop",
				Transitions: []LifecycleTransition{
					{From: "Running", To: "Stopped"},
				},
			},
		},
		InitialState: "New",
	}

	err := lifecycle.ValidateActionAllowed("New", "create")
	if err != nil {
		t.Errorf("ValidateActionAllowed() should not fail: %v", err)
	}

	err = lifecycle.ValidateActionAllowed("Running", "stop")
	if err != nil {
		t.Errorf("ValidateActionAllowed() should not fail: %v", err)
	}
}

func TestValidateActionAllowed_ActionNotAllowed(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "Stopped"},
		},
		Actions: []LifecycleAction{
			{
				Name: "stop",
				Transitions: []LifecycleTransition{
					{From: "Running", To: "Stopped"},
				},
			},
		},
		InitialState: "New",
	}

	err := lifecycle.ValidateActionAllowed("New", "stop")
	if err == nil {
		t.Error("ValidateActionAllowed() should fail for action not allowed from state")
	}
	expectedMsg := "action \"stop\" is not allowed from state \"New\""
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateActionAllowed_ActionNotFound(t *testing.T) {
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
		InitialState: "New",
	}

	err := lifecycle.ValidateActionAllowed("New", "nonexistent")
	if err == nil {
		t.Error("ValidateActionAllowed() should fail for nonexistent action")
	}
	expectedMsg := "action \"nonexistent\" not found in lifecycle schema"
	if err.Error() != expectedMsg {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestIsTerminalState_True(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "Deleted"},
			{Name: "Failed"},
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
		TerminalStates: []string{"Deleted", "Failed"},
	}

	if !lifecycle.IsTerminalState("Deleted") {
		t.Error("IsTerminalState() should return true for 'Deleted'")
	}

	if !lifecycle.IsTerminalState("Failed") {
		t.Error("IsTerminalState() should return true for 'Failed'")
	}
}

func TestIsTerminalState_False(t *testing.T) {
	lifecycle := &LifecycleSchema{
		States: []LifecycleState{
			{Name: "New"},
			{Name: "Running"},
			{Name: "Deleted"},
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

	if lifecycle.IsTerminalState("New") {
		t.Error("IsTerminalState() should return false for 'New'")
	}

	if lifecycle.IsTerminalState("Running") {
		t.Error("IsTerminalState() should return false for 'Running'")
	}
}

func TestIsTerminalState_EmptyTerminalStates(t *testing.T) {
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
		InitialState:   "New",
		TerminalStates: []string{},
	}

	if lifecycle.IsTerminalState("New") {
		t.Error("IsTerminalState() should return false when terminal states list is empty")
	}
}
