package response

import (
	"errors"
	"net/http"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/go-chi/render"
)

// ErrRes is the unified error response body shared by the API and middleware
// layers. pkg/api builds on this type via NewErrRes.
type ErrRes struct {
	Err            error `json:"-"`
	HTTPStatusCode int   `json:"-"`

	Status   int            `json:"status"`
	Message  string         `json:"message"`
	Template string         `json:"template,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
	Errors   []ErrDetail    `json:"errors,omitempty"`
}

// ErrDetail is a single field-level error.
type ErrDetail struct {
	Path     string         `json:"path"`
	Message  string         `json:"message"`
	Template string         `json:"template,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
}

func (e *ErrRes) Render(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(e.HTTPStatusCode)
	return nil
}

// NewErrRes builds a response, lifting template + data when err carries them.
func NewErrRes(status int, err error) *ErrRes {
	res := &ErrRes{
		Err:            err,
		HTTPStatusCode: status,
		Status:         status,
		Message:        err.Error(),
	}
	// Only expose the template when there is data to interpolate: a plain
	// message with no data would otherwise duplicate itself into template.
	var templated domain.Templated
	if errors.As(err, &templated) && len(templated.MessageData()) > 0 {
		res.Template = templated.MessageTemplate()
		res.Data = templated.MessageData()
	}
	return res
}

func ErrInvalidRequest(err error) render.Renderer {
	return NewErrRes(http.StatusBadRequest, err)
}

func ErrNotFound(err error) render.Renderer {
	return NewErrRes(http.StatusNotFound, err)
}

func ErrInternal(err error) render.Renderer {
	return NewErrRes(http.StatusInternalServerError, err)
}

func ErrUnauthenticated(err error) render.Renderer {
	return NewErrRes(http.StatusUnauthorized, err)
}

func ErrUnauthorized(err error) render.Renderer {
	return NewErrRes(http.StatusForbidden, err)
}
