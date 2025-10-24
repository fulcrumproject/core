// Validation error types for schema processing
package schema

import "fmt"

// ValidationError represents a collection of validation errors
type ValidationError struct {
	Errors []ValidationErrorDetail `json:"errors"`
}

// ValidationErrorDetail represents a single validation error with its path
type ValidationErrorDetail struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// NewValidationError creates a new ValidationError from a list of details
func NewValidationError(errors []ValidationErrorDetail) ValidationError {
	return ValidationError{Errors: errors}
}

// Error implements the error interface
func (e ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("validation failed: %s", e.Errors[0].Message)
	}
	return fmt.Sprintf("validation failed: %d errors", len(e.Errors))
}

