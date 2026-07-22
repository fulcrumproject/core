package domain

import (
	"errors"
	"fmt"

	"github.com/fulcrumproject/core/pkg/msgfmt"
)

// asInvalidInput returns err unchanged when it already is a domain error
// (preserving any template/data), otherwise wraps it as an InvalidInputError.
func asInvalidInput(err error) error {
	var inv InvalidInputError
	if errors.As(err, &inv) {
		return err
	}
	return InvalidInputError{Err: err}
}

// MsgData holds the interpolation values for a message template.
type MsgData map[string]any

// Templated is implemented by errors that carry a message template plus data.
type Templated interface {
	MessageTemplate() string
	MessageData() MsgData
}

type NotFoundError struct {
	Err      error
	Template string
	Data     MsgData
}

func NewNotFoundErrorf(format string, a ...any) NotFoundError {
	return NotFoundError{Err: fmt.Errorf(format, a...)}
}

func NewNotFoundError(template string, data MsgData) NotFoundError {
	return NotFoundError{Template: template, Data: data}
}

func (e NotFoundError) Error() string {
	if e.Template != "" {
		return msgfmt.RenderMessage(e.Template, e.Data)
	}
	return fmt.Sprintf("resource not found: %v", e.Err)
}

func (e NotFoundError) Unwrap() error           { return e.Err }
func (e NotFoundError) MessageTemplate() string { return e.Template }
func (e NotFoundError) MessageData() MsgData    { return e.Data }

type InvalidInputError struct {
	Err      error
	Template string
	Data     MsgData
}

func NewInvalidInputErrorf(format string, a ...any) InvalidInputError {
	return InvalidInputError{Err: fmt.Errorf(format, a...)}
}

func NewInvalidInputError(template string, data MsgData) InvalidInputError {
	return InvalidInputError{Template: template, Data: data}
}

func (e InvalidInputError) Error() string {
	if e.Template != "" {
		return msgfmt.RenderMessage(e.Template, e.Data)
	}
	return fmt.Sprintf("invalid input: %v", e.Err)
}

func (e InvalidInputError) Unwrap() error           { return e.Err }
func (e InvalidInputError) MessageTemplate() string { return e.Template }
func (e InvalidInputError) MessageData() MsgData    { return e.Data }

type UnauthorizedError struct {
	Err      error
	Template string
	Data     MsgData
}

func NewUnauthorizedErrorf(format string, a ...any) UnauthorizedError {
	return UnauthorizedError{Err: fmt.Errorf(format, a...)}
}

func NewUnauthorizedError(template string, data MsgData) UnauthorizedError {
	return UnauthorizedError{Template: template, Data: data}
}

func (e UnauthorizedError) Error() string {
	if e.Template != "" {
		return msgfmt.RenderMessage(e.Template, e.Data)
	}
	return fmt.Sprintf("unauthorized: %v", e.Err)
}

func (e UnauthorizedError) Unwrap() error           { return e.Err }
func (e UnauthorizedError) MessageTemplate() string { return e.Template }
func (e UnauthorizedError) MessageData() MsgData    { return e.Data }

type ConflictError struct {
	Err      error
	Template string
	Data     MsgData
}

func NewConflictErrorf(format string, a ...any) ConflictError {
	return ConflictError{Err: fmt.Errorf(format, a...)}
}

func NewConflictError(template string, data MsgData) ConflictError {
	return ConflictError{Template: template, Data: data}
}

func (e ConflictError) Error() string {
	if e.Template != "" {
		return msgfmt.RenderMessage(e.Template, e.Data)
	}
	return fmt.Sprintf("conflict: %v", e.Err)
}

func (e ConflictError) Unwrap() error           { return e.Err }
func (e ConflictError) MessageTemplate() string { return e.Template }
func (e ConflictError) MessageData() MsgData    { return e.Data }
