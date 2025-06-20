package response

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
)

var (
	ErrInvalidFields = errors.New("invalid fields in request")
)

// ErrResponse represents an error response
type ErrResponse struct {
	Err       error  `json:"-"`               // low-level runtime error
	ErrorText string `json:"error,omitempty"` // application-level error message

	HTTPStatusCode int    `json:"-"`      // http response status code
	StatusText     string `json:"status"` // user-level status message

	ValidationErrors []ValidationError `json:"validationErrors,omitempty"` // validation errors if any
}

type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(e.HTTPStatusCode)
	return nil
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		ErrorText:      err.Error(),
		HTTPStatusCode: http.StatusBadRequest,
		StatusText:     "Invalid request",
	}
}

func MultiErrInvalidRequest(validationErrs []ValidationError) render.Renderer {
	return &ErrResponse{
		Err:              ErrInvalidFields,
		ErrorText:        ErrInvalidFields.Error(),
		HTTPStatusCode:   http.StatusBadRequest,
		StatusText:       "Invalid request",
		ValidationErrors: validationErrs,
	}
}

func ErrNotFound(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		ErrorText:      err.Error(),
		HTTPStatusCode: http.StatusNotFound,
		StatusText:     "Resource not found",
	}
}

func ErrInternal(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		ErrorText:      err.Error(),
		HTTPStatusCode: http.StatusInternalServerError,
		StatusText:     "Internal server error",
	}
}

func ErrUnauthenticated(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		ErrorText:      err.Error(),
		HTTPStatusCode: http.StatusUnauthorized,
		StatusText:     "Unauthorized",
	}
}

func ErrUnauthorized(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		ErrorText:      err.Error(),
		HTTPStatusCode: http.StatusForbidden,
		StatusText:     "Forbidden",
	}
}
