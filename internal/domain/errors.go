package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInternal     = errors.New("internal error")
	ErrInvalidInput = errors.New("invalid input")
)

type NotFoundError struct {
	Resource string
	ID       string
	Err      error
}

func NewNotFoundError(resource, id string, err error) error {
	return &NotFoundError{
		Resource: resource,
		ID:       id,
		Err:      err,
	}
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with id %s not found: %v", e.Resource, e.ID, e.Err)
}

func (e *NotFoundError) Unwrap() error {
	return e.Err
}

func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

type InvalidInputError struct {
	Field string
	Err   error
}

func NewInvalidInputError(field string, err error) error {
	return &InvalidInputError{
		Field: field,
		Err:   err,
	}
}

func (e *InvalidInputError) Error() string {
	return fmt.Sprintf("invalid input for field %s: %v", e.Field, e.Err)
}

func (e *InvalidInputError) Unwrap() error {
	return e.Err
}

func (e *InvalidInputError) Is(target error) bool {
	return target == ErrInvalidInput
}

type InternalError struct {
	Err error
}

func NewInternalError(err error) error {
	return &InternalError{
		Err: err,
	}
}

func (e *InternalError) Error() string {
	return fmt.Sprintf("internal unexpected error: %v", e.Err)
}

func (e *InternalError) Unwrap() error {
	return e.Err
}

func (e *InternalError) Is(target error) bool {
	return target == ErrInternal
}
