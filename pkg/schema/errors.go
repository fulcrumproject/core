// Validation error types for schema processing
package schema

import (
	"errors"
	"fmt"

	"github.com/fulcrumproject/core/pkg/msgfmt"
)

// ValidationError represents a collection of validation errors
type ValidationError struct {
	Errors []ValidationErrorDetail `json:"errors"`
}

// ValidationErrorDetail represents a single validation error with its path
type ValidationErrorDetail struct {
	Path     string         `json:"path"`
	Message  string         `json:"message"`
	Template string         `json:"template,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
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

// PropError is a validation failure carrying a message template plus its data.
type PropError struct {
	Template string
	Data     map[string]any
}

func (e PropError) Error() string { return msgfmt.RenderMessage(e.Template, e.Data) }

// newValidationErrorDetail builds a detail for path, lifting template + data
// when err is a PropError.
func newValidationErrorDetail(path string, err error) ValidationErrorDetail {
	detail := ValidationErrorDetail{Path: path, Message: err.Error()}
	var propErr PropError
	if errors.As(err, &propErr) && len(propErr.Data) > 0 {
		detail.Template = propErr.Template
		detail.Data = propErr.Data
	}
	return detail
}
