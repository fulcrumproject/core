package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrResponse_Render(t *testing.T) {
	tests := []struct {
		name           string
		errResponse    *ErrResponse
		expectedStatus int
	}{
		{
			name: "Bad Request",
			errResponse: &ErrResponse{
				HTTPStatusCode: http.StatusBadRequest,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Not Found",
			errResponse: &ErrResponse{
				HTTPStatusCode: http.StatusNotFound,
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Internal Server Error",
			errResponse: &ErrResponse{
				HTTPStatusCode: http.StatusInternalServerError,
			},
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

func TestErrInvalidRequest(t *testing.T) {
	testErr := errors.New("test validation error")

	renderer := ErrInvalidRequest(testErr)
	errResp, ok := renderer.(*ErrResponse)
	require.True(t, ok, "Expected *ErrResponse type")

	assert.Equal(t, testErr, errResp.Err, "Err should match the input error")
	assert.Equal(t, testErr.Error(), errResp.ErrorText, "ErrorText should match error message")
	assert.Equal(t, http.StatusBadRequest, errResp.HTTPStatusCode, "HTTPStatusCode should be BadRequest")
	assert.Equal(t, "Invalid request", errResp.StatusText, "StatusText should be 'Invalid request'")
	assert.Nil(t, errResp.ValidationErrors, "ValidationErrors should be nil")
}

func TestMultiErrInvalidRequest(t *testing.T) {
	validationErrs := []ValidationError{
		{Path: "name", Message: "name is required"},
		{Path: "email", Message: "email is invalid"},
	}

	renderer := MultiErrInvalidRequest(validationErrs)
	errResp, ok := renderer.(*ErrResponse)
	require.True(t, ok, "Expected *ErrResponse type")

	assert.Equal(t, ErrInvalidFields, errResp.Err, "Err should be ErrInvalidFields")
	assert.Equal(t, ErrInvalidFields.Error(), errResp.ErrorText, "ErrorText should match ErrInvalidFields message")
	assert.Equal(t, http.StatusBadRequest, errResp.HTTPStatusCode, "HTTPStatusCode should be BadRequest")
	assert.Equal(t, "Invalid request", errResp.StatusText, "StatusText should be 'Invalid request'")
	assert.Len(t, errResp.ValidationErrors, 2, "Should have 2 validation errors")
	assert.Equal(t, "name", errResp.ValidationErrors[0].Path, "First validation error path should be 'name'")
	assert.Equal(t, "email is invalid", errResp.ValidationErrors[1].Message, "Second validation error message should match")
}

func TestErrNotFound(t *testing.T) {
	testErr := errors.New("resource not found")

	renderer := ErrNotFound(testErr)
	errResp, ok := renderer.(*ErrResponse)
	require.True(t, ok, "Expected *ErrResponse type")

	assert.Equal(t, testErr, errResp.Err, "Err should match the input error")
	assert.Equal(t, testErr.Error(), errResp.ErrorText, "ErrorText should match error message")
	assert.Equal(t, http.StatusNotFound, errResp.HTTPStatusCode, "HTTPStatusCode should be NotFound")
	assert.Equal(t, "Resource not found", errResp.StatusText, "StatusText should be 'Resource not found'")
}

func TestErrInternal(t *testing.T) {
	testErr := errors.New("database connection failed")

	renderer := ErrInternal(testErr)
	errResp, ok := renderer.(*ErrResponse)
	require.True(t, ok, "Expected *ErrResponse type")

	assert.Equal(t, testErr, errResp.Err, "Err should match the input error")
	assert.Equal(t, testErr.Error(), errResp.ErrorText, "ErrorText should match error message")
	assert.Equal(t, http.StatusInternalServerError, errResp.HTTPStatusCode, "HTTPStatusCode should be InternalServerError")
	assert.Equal(t, "Internal server error", errResp.StatusText, "StatusText should be 'Internal server error'")
}

func TestErrUnauthenticated(t *testing.T) {
	testErr := errors.New("invalid credentials")

	renderer := ErrUnauthenticated(testErr)
	errResp, ok := renderer.(*ErrResponse)
	require.True(t, ok, "Expected *ErrResponse type")

	assert.Equal(t, testErr, errResp.Err, "Err should match the input error")
	assert.Equal(t, testErr.Error(), errResp.ErrorText, "ErrorText should match error message")
	assert.Equal(t, http.StatusUnauthorized, errResp.HTTPStatusCode, "HTTPStatusCode should be Unauthorized")
	assert.Equal(t, "Unauthorized", errResp.StatusText, "StatusText should be 'Unauthorized'")
}

func TestErrUnauthorized(t *testing.T) {
	testErr := errors.New("insufficient permissions")

	renderer := ErrUnauthorized(testErr)
	errResp, ok := renderer.(*ErrResponse)
	require.True(t, ok, "Expected *ErrResponse type")

	assert.Equal(t, testErr, errResp.Err, "Err should match the input error")
	assert.Equal(t, testErr.Error(), errResp.ErrorText, "ErrorText should match error message")
	assert.Equal(t, http.StatusForbidden, errResp.HTTPStatusCode, "HTTPStatusCode should be Forbidden")
	assert.Equal(t, "Forbidden", errResp.StatusText, "StatusText should be 'Forbidden'")
}
