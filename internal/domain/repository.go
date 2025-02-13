package domain

import (
	"errors"
)

var (
	// ErrNotFound indicates that the requested entity was not found
	ErrNotFound = errors.New("entity not found")

	// ErrConflict indicates that the operation cannot be completed due to a conflict
	ErrConflict = errors.New("entity conflict")

	// ErrInvalidInput indicates that the input data is invalid
	ErrInvalidInput = errors.New("invalid input")
)
