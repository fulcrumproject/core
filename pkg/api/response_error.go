package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/go-chi/render"
)

// ErrResponse represents an error response
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	ErrorText  string `json:"error,omitempty"` // application-level error message
}

// ValidationErrResponse represents a validation error response with detailed errors
type ValidationErrResponse struct {
	Err            error                          `json:"-"` // low-level runtime error
	HTTPStatusCode int                            `json:"-"` // http response status code
	StatusText     string                         `json:"status"`
	Valid          bool                           `json:"valid"`
	Errors         []domain.ValidationErrorDetail `json:"errors"`
}

func ErrDomain(err error) render.Renderer {
	slog.Error("API domain error", "error", err)
	if validationErr, ok := err.(domain.ValidationError); ok {
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
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     "Invalid request",
		ErrorText:      err.Error(),
	}
}

func ErrNotFound() render.Renderer {
	return &ErrResponse{
		HTTPStatusCode: http.StatusNotFound,
		StatusText:     "Resource not found",
	}
}

func ErrInternal(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusInternalServerError,
		StatusText:     "Internal server error",
		ErrorText:      err.Error(),
	}
}

func ErrUnauthenticated() render.Renderer {
	return &ErrResponse{
		HTTPStatusCode: http.StatusUnauthorized,
		StatusText:     "Unauthorized",
		ErrorText:      "Authentication required",
	}
}

func ErrUnauthorized(err error) render.Renderer {
	return &ErrResponse{
		HTTPStatusCode: http.StatusForbidden,
		StatusText:     "Forbidden",
		ErrorText:      err.Error(),
	}
}

func ErrValidation(err domain.ValidationError) render.Renderer {
	return &ValidationErrResponse{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     "Validation failed",
		Valid:          false,
		Errors:         err.Errors,
	}
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(e.HTTPStatusCode)
	return nil
}

func (e *ValidationErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(e.HTTPStatusCode)
	return nil
}
