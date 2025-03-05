package api

import (
	"errors"
	"log"
	"net/http"

	"fulcrumproject.org/core/internal/domain"
	"github.com/go-chi/render"
)

// ErrResponse represents an error response
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	ErrorText  string `json:"error,omitempty"` // application-level error message
}

func ErrDomain(err error) render.Renderer {
	log.Printf("API domain error %v", err)
	if errors.As(err, &domain.InvalidInputError{}) {
		return ErrInvalidRequest(err)
	}
	if errors.As(err, &domain.NotFoundError{}) {
		return ErrNotFound()
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

func ErrUnauthorized() render.Renderer {
	return &ErrResponse{
		HTTPStatusCode: http.StatusUnauthorized,
		StatusText:     "Unauthorized",
		ErrorText:      "Authentication required",
	}
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}
