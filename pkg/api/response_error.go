package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/render"
)

// ErrRes represents an error response
type ErrRes struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	ErrorText  string `json:"error,omitempty"` // application-level error message
}

// ValidationErrRes represents a validation error response with detailed errors
type ValidationErrRes struct {
	Err            error                          `json:"-"` // low-level runtime error
	HTTPStatusCode int                            `json:"-"` // http response status code
	StatusText     string                         `json:"status"`
	Valid          bool                           `json:"valid"`
	Errors         []schema.ValidationErrorDetail `json:"errors"`
}

func ErrDomain(err error) render.Renderer {
	slog.Error("API domain error", "error", err)
	if validationErr, ok := err.(schema.ValidationError); ok {
		return ErrValidation(validationErr)
	}
	if errors.As(err, &domain.InvalidInputError{}) {
		return ErrInvalidRequest(err)
	}
	if errors.As(err, &domain.NotFoundError{}) {
		return ErrNotFound()
	}
	if errors.As(err, &domain.UnauthorizedError{}) {
		return ErrUnauthorized(err)
	}
	return ErrInternal(err)
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrRes{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     "Invalid request",
		ErrorText:      err.Error(),
	}
}

func ErrNotFound() render.Renderer {
	return &ErrRes{
		HTTPStatusCode: http.StatusNotFound,
		StatusText:     "Resource not found",
	}
}

func ErrInternal(err error) render.Renderer {
	return &ErrRes{
		Err:            err,
		HTTPStatusCode: http.StatusInternalServerError,
		StatusText:     "Internal server error",
		ErrorText:      err.Error(),
	}
}

func ErrUnauthenticated() render.Renderer {
	return &ErrRes{
		HTTPStatusCode: http.StatusUnauthorized,
		StatusText:     "Unauthorized",
		ErrorText:      "Authentication required",
	}
}

func ErrUnauthorized(err error) render.Renderer {
	return &ErrRes{
		HTTPStatusCode: http.StatusForbidden,
		StatusText:     "Forbidden",
		ErrorText:      err.Error(),
	}
}

func ErrValidation(err schema.ValidationError) render.Renderer {
	return &ValidationErrRes{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     "Validation failed",
		Valid:          false,
		Errors:         err.Errors,
	}
}

func (e *ErrRes) Render(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(e.HTTPStatusCode)
	return nil
}

func (e *ValidationErrRes) Render(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(e.HTTPStatusCode)
	return nil
}
