package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrResponse_Render(t *testing.T) {
	tests := []struct {
		name           string
		errResponse    *ErrRes
		expectedStatus int
	}{
		{
			name:           "Bad Request",
			errResponse:    &ErrRes{Status: http.StatusBadRequest},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Not Found",
			errResponse:    &ErrRes{Status: http.StatusNotFound},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Internal Server Error",
			errResponse:    &ErrRes{Status: http.StatusInternalServerError},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)

			err := tt.errResponse.Render(w, r)
			assert.NoError(t, err, "Render() should not return an error")
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code should match expected value")
		})
	}
}

func TestErrConstructors(t *testing.T) {
	tests := []struct {
		name       string
		renderer   render.Renderer
		wantStatus int
		wantErr    error
	}{
		{"invalid request", ErrInvalidRequest(errors.New("bad")), http.StatusBadRequest, errors.New("bad")},
		{"not found", ErrNotFound(errors.New("gone")), http.StatusNotFound, errors.New("gone")},
		{"internal", ErrInternal(errors.New("boom")), http.StatusInternalServerError, errors.New("boom")},
		{"unauthenticated", ErrUnauthenticated(errors.New("nope")), http.StatusUnauthorized, errors.New("nope")},
		{"unauthorized", ErrUnauthorized(errors.New("denied")), http.StatusForbidden, errors.New("denied")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, ok := tt.renderer.(*ErrRes)
			require.True(t, ok, "expected *ErrRes")
			assert.Equal(t, tt.wantStatus, res.Status)
			assert.Equal(t, tt.wantErr.Error(), res.Message)
			assert.Empty(t, res.Errors)
		})
	}
}
