package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/schema"
	"github.com/go-chi/render"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderToMap(t *testing.T, r render.Renderer) map[string]any {
	t.Helper()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	require.NoError(t, render.Render(w, req, r))
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestErrDomain_ConflictTemplated(t *testing.T) {
	err := domain.NewConflictError("{field} '{value}' already exists", domain.MsgData{"field": "name", "value": "foo"})
	body := renderToMap(t, ErrDomain(err))

	assert.Equal(t, float64(409), body["status"])
	assert.Equal(t, "name 'foo' already exists", body["message"])
	assert.Equal(t, "{field} '{value}' already exists", body["template"])
	assert.Equal(t, "name", body["data"].(map[string]any)["field"])
}

func TestErrDomain_PlainInvalidInputOmitsTemplate(t *testing.T) {
	err := domain.NewInvalidInputError("config pool name cannot be empty", nil)
	body := renderToMap(t, ErrDomain(err))

	assert.Equal(t, float64(400), body["status"])
	assert.Equal(t, "config pool name cannot be empty", body["message"])
	_, hasTemplate := body["template"]
	assert.False(t, hasTemplate, "no data means no template")
	_, hasData := body["data"]
	assert.False(t, hasData, "no data must be omitted")
}

func TestErrDomain_InternalHasNumericStatus(t *testing.T) {
	body := renderToMap(t, ErrDomain(assertAnError{}))
	assert.Equal(t, float64(500), body["status"])
	assert.NotEmpty(t, body["message"])
	_, hasTemplate := body["template"]
	assert.False(t, hasTemplate, "plain error must omit template")
}

func TestErrValidation_MapsToErrorsArray(t *testing.T) {
	verr := schema.NewValidationError([]schema.ValidationErrorDetail{
		{Path: "name", Message: "too short"},
	})
	body := renderToMap(t, ErrDomain(verr))

	assert.Equal(t, float64(400), body["status"])
	assert.Equal(t, "validation failed", body["message"])
	errs := body["errors"].([]any)
	require.Len(t, errs, 1)
	first := errs[0].(map[string]any)
	assert.Equal(t, "name", first["path"])
	assert.Equal(t, "too short", first["message"])
}

func TestErrDomain_WrappedValidationKeepsErrorsArray(t *testing.T) {
	verr := schema.NewValidationError([]schema.ValidationErrorDetail{
		{Path: "name", Message: "too short"},
	})
	body := renderToMap(t, ErrDomain(fmt.Errorf("update failed: %w", verr)))

	assert.Equal(t, float64(400), body["status"])
	errs := body["errors"].([]any)
	require.Len(t, errs, 1)
	assert.Equal(t, "name", errs[0].(map[string]any)["path"])
}

type assertAnError struct{}

func (assertAnError) Error() string { return "boom" }
