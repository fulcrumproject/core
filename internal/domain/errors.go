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
