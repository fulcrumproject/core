package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewKeycloakUserHandler(t *testing.T) {
	querier := domain.NewMockKeycloakUserQuerier(t)
	commander := domain.NewMockKeycloakUserCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewKeycloakUserHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

func TestKeycloakUserHandlerRoutes(t *testing.T) {
	querier := domain.NewMockKeycloakUserQuerier(t)
	commander := domain.NewMockKeycloakUserCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewKeycloakUserHandler(querier, commander, authz)

	routeFunc := handler.Routes()
	assert.NotNil(t, routeFunc)

	r := chi.NewRouter()
	routeFunc(r)

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		switch {
		case method == "GET" && route == "/":
		case method == "POST" && route == "/":
		case method == "GET" && route == "/{id}":
		case method == "GET" && route == "/{id}/":
		case method == "PATCH" && route == "/{id}":
		case method == "PATCH" && route == "/{id}/":
		case method == "DELETE" && route == "/{id}":
		case method == "DELETE" && route == "/{id}/":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}

	err := chi.Walk(r, walkFunc)
	assert.NoError(t, err)
}

func TestKeycloakUserToRes(t *testing.T) {
	user := &domain.KeycloakUser{
		ID:            "user-123",
		Username:      "john",
		Email:         "john@example.com",
		FirstName:     "John",
		LastName:      "Doe",
		Enabled:       true,
		Roles:         []auth.Role{auth.RoleAdmin},
		ParticipantID: "p-1",
		AgentID:       "a-1",
	}

	res := KeycloakUserToRes(user)

	assert.Equal(t, "user-123", res.ID)
	assert.Equal(t, "john", res.Username)
	assert.Equal(t, "john@example.com", res.Email)
	assert.Equal(t, "John", res.FirstName)
	assert.Equal(t, "Doe", res.LastName)
	assert.True(t, res.Enabled)
	assert.Equal(t, []auth.Role{auth.RoleAdmin}, res.Roles)
	assert.Equal(t, "p-1", res.ParticipantID)
	assert.Equal(t, "a-1", res.AgentID)
}

func TestKeycloakUserHandlerList(t *testing.T) {
	testCases := []struct {
		name           string
		url            string
		mockSetup      func(q *domain.MockKeycloakUserQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			url:  "/keycloak-users?page=1&pageSize=10&firstName=john",
			mockSetup: func(q *domain.MockKeycloakUserQuerier) {
				q.EXPECT().
					List(mock.Anything, mock.MatchedBy(func(p domain.KeycloakUserListParams) bool {
						return p.FirstName == "john" && p.Page == 1 && p.PageSize == 10
					})).
					Return(&domain.PageRes[domain.KeycloakUserListItem]{
						Items: []domain.KeycloakUserListItem{
							{ID: "u-1", Username: "john", Email: "john@test.com", FirstName: "John", LastName: "Doe"},
						},
						TotalItems:  1,
						TotalPages:  1,
						CurrentPage: 1,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "QuerierError",
			url:  "/keycloak-users?page=1&pageSize=10",
			mockSetup: func(q *domain.MockKeycloakUserQuerier) {
				q.EXPECT().List(mock.Anything, mock.Anything).Return(nil, fmt.Errorf("keycloak unavailable"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			querier := domain.NewMockKeycloakUserQuerier(t)
			tc.mockSetup(querier)

			h := &KeycloakUserHandler{querier: querier}
			handler := http.HandlerFunc(h.List)

			req := httptest.NewRequest("GET", tc.url, nil)
			req = req.WithContext(auth.WithIdentity(req.Context(), newMockAuthAdmin()))
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				items := response["items"].([]any)
				assert.Len(t, items, 1)
				first := items[0].(map[string]any)
				assert.Equal(t, "john", first["username"])
			}
		})
	}
}

func TestKeycloakUserHandlerCreate(t *testing.T) {
	testCases := []struct {
		name           string
		request        CreateKeycloakUserReq
		mockSetup      func(c *domain.MockKeycloakUserCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			request: CreateKeycloakUserReq{
				Username:  "newuser",
				Email:     "new@test.com",
				FirstName: "New",
				LastName:  "User",
				Password:  "secret",
				Enabled:   true,
				Role:      auth.RoleAdmin,
			},
			mockSetup: func(c *domain.MockKeycloakUserCommander) {
				c.EXPECT().
					Create(mock.Anything, mock.MatchedBy(func(p domain.CreateKeycloakUserParams) bool {
						return p.Username == "newuser" && p.Email == "new@test.com"
					})).
					Return(&domain.KeycloakUser{
						ID:        "created-id",
						Username:  "newuser",
						Email:     "new@test.com",
						FirstName: "New",
						LastName:  "User",
						Enabled:   true,
						Roles:     []auth.Role{auth.RoleAdmin},
					}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "CommanderError",
			request: CreateKeycloakUserReq{
				Username: "bad",
			},
			mockSetup: func(c *domain.MockKeycloakUserCommander) {
				c.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, domain.NewInvalidInputErrorf("username too short"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commander := domain.NewMockKeycloakUserCommander(t)
			tc.mockSetup(commander)

			bodyBytes, err := json.Marshal(tc.request)
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/keycloak-users", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(auth.WithIdentity(req.Context(), newMockAuthAdmin()))

			w := httptest.NewRecorder()
			h := &KeycloakUserHandler{commander: commander}
			middlewareHandler := middlewares.DecodeBody[CreateKeycloakUserReq]()(http.HandlerFunc(h.Create))
			middlewareHandler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusCreated {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "created-id", response["id"])
				assert.Equal(t, "newuser", response["username"])
			}
		})
	}
}

func TestKeycloakUserHandlerGet(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(q *domain.MockKeycloakUserQuerier)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "user-123",
			mockSetup: func(q *domain.MockKeycloakUserQuerier) {
				q.EXPECT().Get(mock.Anything, "user-123").Return(&domain.KeycloakUser{
					ID:        "user-123",
					Username:  "john",
					Email:     "john@test.com",
					FirstName: "John",
					LastName:  "Doe",
					Enabled:   true,
					Roles:     []auth.Role{auth.RoleAdmin},
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "NotFound",
			id:   "nonexistent",
			mockSetup: func(q *domain.MockKeycloakUserQuerier) {
				q.EXPECT().Get(mock.Anything, "nonexistent").Return(nil, domain.NewNotFoundErrorf("user not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			querier := domain.NewMockKeycloakUserQuerier(t)
			tc.mockSetup(querier)

			h := &KeycloakUserHandler{querier: querier}
			handler := http.HandlerFunc(h.Get)

			req := httptest.NewRequest("GET", "/keycloak-users/"+tc.id, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(auth.WithIdentity(req.Context(), newMockAuthAdmin()))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "user-123", response["id"])
				assert.Equal(t, "john", response["username"])
			}
		})
	}
}

func TestKeycloakUserHandlerUpdate(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		request        UpdateKeycloakUserReq
		mockSetup      func(c *domain.MockKeycloakUserCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "user-123",
			request: UpdateKeycloakUserReq{
				Email:     helpers.StringPtr("updated@test.com"),
				FirstName: helpers.StringPtr("Updated"),
			},
			mockSetup: func(c *domain.MockKeycloakUserCommander) {
				c.EXPECT().
					Update(mock.Anything, "user-123", mock.MatchedBy(func(p domain.UpdateKeycloakUserParams) bool {
						return *p.Email == "updated@test.com" && *p.FirstName == "Updated"
					})).
					Return(&domain.KeycloakUser{
						ID:        "user-123",
						Username:  "john",
						Email:     "updated@test.com",
						FirstName: "Updated",
						LastName:  "Doe",
						Enabled:   true,
						Roles:     []auth.Role{auth.RoleAdmin},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "CommanderError",
			id:   "user-123",
			request: UpdateKeycloakUserReq{
				Email: helpers.StringPtr("bad"),
			},
			mockSetup: func(c *domain.MockKeycloakUserCommander) {
				c.EXPECT().Update(mock.Anything, "user-123", mock.Anything).Return(nil, domain.NewInvalidInputErrorf("invalid email"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commander := domain.NewMockKeycloakUserCommander(t)
			tc.mockSetup(commander)

			bodyBytes, err := json.Marshal(tc.request)
			require.NoError(t, err)
			req := httptest.NewRequest("PATCH", "/keycloak-users/"+tc.id, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(auth.WithIdentity(req.Context(), newMockAuthAdmin()))

			w := httptest.NewRecorder()
			h := &KeycloakUserHandler{commander: commander}
			middlewareHandler := middlewares.DecodeBody[UpdateKeycloakUserReq]()(http.HandlerFunc(h.Update))
			middlewareHandler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectedStatus == http.StatusOK {
				var response map[string]any
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "user-123", response["id"])
				assert.Equal(t, "updated@test.com", response["email"])
			}
		})
	}
}

func TestKeycloakUserHandlerDelete(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		mockSetup      func(c *domain.MockKeycloakUserCommander)
		expectedStatus int
	}{
		{
			name: "Success",
			id:   "user-123",
			mockSetup: func(c *domain.MockKeycloakUserCommander) {
				c.EXPECT().Delete(mock.Anything, "user-123").Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "CommanderError",
			id:   "user-123",
			mockSetup: func(c *domain.MockKeycloakUserCommander) {
				c.EXPECT().Delete(mock.Anything, "user-123").Return(fmt.Errorf("keycloak unavailable"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commander := domain.NewMockKeycloakUserCommander(t)
			tc.mockSetup(commander)

			h := &KeycloakUserHandler{commander: commander}
			handler := http.HandlerFunc(h.Delete)

			req := httptest.NewRequest("DELETE", "/keycloak-users/"+tc.id, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.id)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = req.WithContext(auth.WithIdentity(req.Context(), newMockAuthAdmin()))

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
