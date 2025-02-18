package domain

import (
	"fmt"
)

type NotFoundError struct {
	Err error
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

func (e InvalidInputError) Error() string {
	return fmt.Sprintf("invalid input: %v", e.Err)
}

func (e InvalidInputError) Unwrap() error {
	return e.Err
}
