package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/response"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/render"
)

func ErrDomain(err error) render.Renderer {
	slog.Error("API domain error", "error", err)
	var validationErr schema.ValidationError
	if errors.As(err, &validationErr) {
		return ErrValidation(validationErr)
	}
	if errors.As(err, &domain.InvalidInputError{}) {
		return ErrInvalidRequest(err)
	}
	if errors.As(err, &domain.NotFoundError{}) {
		return response.NewErrRes(http.StatusNotFound, err)
	}
	if errors.As(err, &domain.UnauthorizedError{}) {
		return ErrUnauthorized(err)
	}
	if errors.As(err, &domain.ConflictError{}) {
		return ErrConflict(err)
	}
	return ErrInternal(err)
}

func ErrConflict(err error) render.Renderer {
	return response.NewErrRes(http.StatusConflict, err)
}

func ErrInvalidRequest(err error) render.Renderer {
	return response.NewErrRes(http.StatusBadRequest, err)
}

func ErrNotFound() render.Renderer {
	return &response.ErrRes{
		HTTPStatusCode: http.StatusNotFound,
		Status:         http.StatusNotFound,
		Message:        "resource not found",
	}
}

func ErrInternal(err error) render.Renderer {
	return response.NewErrRes(http.StatusInternalServerError, err)
}

func ErrUnauthenticated() render.Renderer {
	return &response.ErrRes{
		HTTPStatusCode: http.StatusUnauthorized,
		Status:         http.StatusUnauthorized,
		Message:        "authentication required",
	}
}

func ErrUnauthorized(err error) render.Renderer {
	return response.NewErrRes(http.StatusForbidden, err)
}

func ErrValidation(err schema.ValidationError) render.Renderer {
	details := make([]response.ErrDetail, 0, len(err.Errors))
	for _, d := range err.Errors {
		// schema.newValidationErrorDetail already gates template+data on
		// non-empty data, so copy the fields straight across.
		details = append(details, response.ErrDetail{
			Path:     d.Path,
			Message:  d.Message,
			Template: d.Template,
			Data:     d.Data,
		})
	}
	return &response.ErrRes{
		Err:            err,
		HTTPStatusCode: http.StatusBadRequest,
		Status:         http.StatusBadRequest,
		Message:        "validation failed",
		Errors:         details,
	}
}
