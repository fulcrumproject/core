package schema

import (
	"errors"
	"fmt"
	"testing"
)

func TestPropError_Render(t *testing.T) {
	err := PropError{
		Template: "string length {length} is less than minimum {min}",
		Data:     map[string]any{"length": 2, "min": 5},
	}
	if got := err.Error(); got != "string length 2 is less than minimum 5" {
		t.Errorf("Error() = %q", got)
	}
}

func TestNewValidationErrorDetail(t *testing.T) {
	t.Run("lifts template and data from PropError", func(t *testing.T) {
		propErr := PropError{Template: "value {v} bad", Data: map[string]any{"v": 1}}
		d := newValidationErrorDetail("age", propErr)

		if d.Path != "age" {
			t.Errorf("Path = %q", d.Path)
		}
		if d.Message != "value 1 bad" {
			t.Errorf("Message = %q", d.Message)
		}
		if d.Template != "value {v} bad" {
			t.Errorf("Template = %q", d.Template)
		}
		if d.Data["v"] != 1 {
			t.Errorf("Data[v] = %v", d.Data["v"])
		}
	})

	t.Run("plain error leaves template empty", func(t *testing.T) {
		d := newValidationErrorDetail("name", fmt.Errorf("boom"))
		if d.Message != "boom" {
			t.Errorf("Message = %q", d.Message)
		}
		if d.Template != "" {
			t.Errorf("Template = %q, want empty", d.Template)
		}
		if d.Data != nil {
			t.Errorf("Data = %v, want nil", d.Data)
		}
	})

	t.Run("errors.As finds wrapped PropError", func(t *testing.T) {
		propErr := PropError{Template: "x", Data: map[string]any{}}
		wrapped := fmt.Errorf("context: %w", propErr)
		var target PropError
		if !errors.As(wrapped, &target) {
			t.Fatal("errors.As should unwrap to PropError")
		}
	})
}
