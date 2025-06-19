package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestNotFoundError(t *testing.T) {
	// Test constructor
	t.Run("Constructor", func(t *testing.T) {
		err := NewNotFoundErrorf("resource %s with id %d", "user", 123)
		if err.Err == nil {
			t.Fatal("Expected err.Err to be non-nil")
		}
		expectedErrMsg := fmt.Sprintf("resource %s with id %d", "user", 123)
		if err.Err.Error() != expectedErrMsg {
			t.Errorf("Expected err.Err.Error() to be %q, got %q", expectedErrMsg, err.Err.Error())
		}
	})

	// Test Error() method
	t.Run("Error method", func(t *testing.T) {
		err := NewNotFoundErrorf("resource %s with id %d", "user", 123)
		expectedErrMsg := "resource not found: resource user with id 123"
		if err.Error() != expectedErrMsg {
			t.Errorf("Expected err.Error() to be %q, got %q", expectedErrMsg, err.Error())
		}
	})

	// Test Unwrap() method
	t.Run("Unwrap method", func(t *testing.T) {
		innerErr := fmt.Errorf("resource user with id 123")
		err := NotFoundError{Err: innerErr}
		if err.Unwrap() != innerErr {
			t.Errorf("Expected err.Unwrap() to return the inner error")
		}
	})

	// Test errors.Is functionality
	t.Run("errors.Is functionality", func(t *testing.T) {
		innerErr := fmt.Errorf("resource user with id 123")
		err := NotFoundError{Err: innerErr}

		// Should match the specific inner error
		if !errors.Is(err, innerErr) {
			t.Errorf("Expected errors.Is(err, innerErr) to be true")
		}

		// Should not match a different error
		otherErr := fmt.Errorf("something else")
		if errors.Is(err, otherErr) {
			t.Errorf("Expected errors.Is(err, otherErr) to be false")
		}
	})

	// Test errors.As functionality
	t.Run("errors.As functionality", func(t *testing.T) {
		err := NewNotFoundErrorf("resource user with id 123")

		var notFoundErr NotFoundError
		if !errors.As(err, &notFoundErr) {
			t.Errorf("Expected errors.As(err, &notFoundErr) to be true")
		}

		var invalidInputErr InvalidInputError
		if errors.As(err, &invalidInputErr) {
			t.Errorf("Expected errors.As(err, &invalidInputErr) to be false")
		}
	})
}

func TestInvalidInputError(t *testing.T) {
	// Test constructor
	t.Run("Constructor", func(t *testing.T) {
		err := NewInvalidInputErrorf("field %s must be %s", "name", "non-empty")
		if err.Err == nil {
			t.Fatal("Expected err.Err to be non-nil")
		}
		expectedErrMsg := fmt.Sprintf("field %s must be %s", "name", "non-empty")
		if err.Err.Error() != expectedErrMsg {
			t.Errorf("Expected err.Err.Error() to be %q, got %q", expectedErrMsg, err.Err.Error())
		}
	})

	// Test Error() method
	t.Run("Error method", func(t *testing.T) {
		err := NewInvalidInputErrorf("field %s must be %s", "name", "non-empty")
		expectedErrMsg := "invalid input: field name must be non-empty"
		if err.Error() != expectedErrMsg {
			t.Errorf("Expected err.Error() to be %q, got %q", expectedErrMsg, err.Error())
		}
	})

	// Test Unwrap() method
	t.Run("Unwrap method", func(t *testing.T) {
		innerErr := fmt.Errorf("field name must be non-empty")
		err := InvalidInputError{Err: innerErr}
		if err.Unwrap() != innerErr {
			t.Errorf("Expected err.Unwrap() to return the inner error")
		}
	})

	// Test errors.Is functionality
	t.Run("errors.Is functionality", func(t *testing.T) {
		innerErr := fmt.Errorf("field name must be non-empty")
		err := InvalidInputError{Err: innerErr}

		// Should match the specific inner error
		if !errors.Is(err, innerErr) {
			t.Errorf("Expected errors.Is(err, innerErr) to be true")
		}

		// Should not match a different error
		otherErr := fmt.Errorf("something else")
		if errors.Is(err, otherErr) {
			t.Errorf("Expected errors.Is(err, otherErr) to be false")
		}
	})

	// Test errors.As functionality
	t.Run("errors.As functionality", func(t *testing.T) {
		err := NewInvalidInputErrorf("field name must be non-empty")

		var invalidInputErr InvalidInputError
		if !errors.As(err, &invalidInputErr) {
			t.Errorf("Expected errors.As(err, &invalidInputErr) to be true")
		}

		var notFoundErr NotFoundError
		if errors.As(err, &notFoundErr) {
			t.Errorf("Expected errors.As(err, &notFoundErr) to be false")
		}
	})
}

func TestUnauthorizedError(t *testing.T) {
	// Test constructor
	t.Run("Constructor", func(t *testing.T) {
		err := NewUnauthorizedErrorf("user %s lacks permission %s", "user1", "admin")
		if err.Err == nil {
			t.Fatal("Expected err.Err to be non-nil")
		}
		expectedErrMsg := fmt.Sprintf("user %s lacks permission %s", "user1", "admin")
		if err.Err.Error() != expectedErrMsg {
			t.Errorf("Expected err.Err.Error() to be %q, got %q", expectedErrMsg, err.Err.Error())
		}
	})

	// Test Error() method
	t.Run("Error method", func(t *testing.T) {
		err := NewUnauthorizedErrorf("user %s lacks permission %s", "user1", "admin")
		expectedErrMsg := "unauthorized: user user1 lacks permission admin"
		if err.Error() != expectedErrMsg {
			t.Errorf("Expected err.Error() to be %q, got %q", expectedErrMsg, err.Error())
		}
	})

	// Test Unwrap() method
	t.Run("Unwrap method", func(t *testing.T) {
		innerErr := fmt.Errorf("user user1 lacks permission admin")
		err := UnauthorizedError{Err: innerErr}
		if err.Unwrap() != innerErr {
			t.Errorf("Expected err.Unwrap() to return the inner error")
		}
	})

	// Test errors.Is functionality
	t.Run("errors.Is functionality", func(t *testing.T) {
		innerErr := fmt.Errorf("user user1 lacks permission admin")
		err := UnauthorizedError{Err: innerErr}

		// Should match the specific inner error
		if !errors.Is(err, innerErr) {
			t.Errorf("Expected errors.Is(err, innerErr) to be true")
		}

		// Should not match a different error
		otherErr := fmt.Errorf("something else")
		if errors.Is(err, otherErr) {
			t.Errorf("Expected errors.Is(err, otherErr) to be false")
		}
	})

	// Test errors.As functionality
	t.Run("errors.As functionality", func(t *testing.T) {
		err := NewUnauthorizedErrorf("user user1 lacks permission admin")

		var unauthorizedErr UnauthorizedError
		if !errors.As(err, &unauthorizedErr) {
			t.Errorf("Expected errors.As(err, &unauthorizedErr) to be true")
		}

		var notFoundErr NotFoundError
		if errors.As(err, &notFoundErr) {
			t.Errorf("Expected errors.As(err, &notFoundErr) to be false")
		}
	})
}

func TestErrorChaining(t *testing.T) {
	// Test error wrapping and unwrapping through multiple levels
	t.Run("Error wrapping chain", func(t *testing.T) {
		baseErr := fmt.Errorf("original error")
		notFoundErr := NotFoundError{Err: baseErr}
		invalidInputErr := InvalidInputError{Err: notFoundErr}

		// Check if we can unwrap to the original error
		if !errors.Is(invalidInputErr, baseErr) {
			t.Errorf("Expected errors.Is(invalidInputErr, baseErr) to be true")
		}

		// Check if we can identify the error types in the chain
		var nfErr NotFoundError
		if !errors.As(invalidInputErr, &nfErr) {
			t.Errorf("Expected errors.As(invalidInputErr, &nfErr) to be true")
		}
	})
}
