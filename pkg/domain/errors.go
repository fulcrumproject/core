package domain

import (
	"fmt"
)

type NotFoundError struct {
	Err error
}

func NewNotFoundErrorf(format string, a ...any) NotFoundError {
	return NotFoundError{Err: fmt.Errorf(format, a...)}
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("resource not found: %v", e.Err)
}

func (e NotFoundError) Unwrap() error {
	return e.Err
}

type InvalidInputError struct {
	Err error
}

func NewInvalidInputErrorf(format string, a ...any) InvalidInputError {
	return InvalidInputError{Err: fmt.Errorf(format, a...)}
}

func (e InvalidInputError) Error() string {
	return fmt.Sprintf("invalid input: %v", e.Err)
}

func (e InvalidInputError) Unwrap() error {
	return e.Err
}

type UnauthorizedError struct {
	Err error
}

func NewUnauthorizedErrorf(format string, a ...any) UnauthorizedError {
	return UnauthorizedError{Err: fmt.Errorf(format, a...)}
}

func (e UnauthorizedError) Error() string {
	return fmt.Sprintf("unauthorized: %v", e.Err)
}

func (e UnauthorizedError) Unwrap() error {
	return e.Err
}

type ValidationError struct {
	Errors []ValidationErrorDetail `json:"errors"`
}

type ValidationErrorDetail struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func NewValidationError(errors []ValidationErrorDetail) ValidationError {
	return ValidationError{Errors: errors}
}

func (e ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("validation failed: %s", e.Errors[0].Message)
	}
	return fmt.Sprintf("validation failed: %d errors", len(e.Errors))
}
