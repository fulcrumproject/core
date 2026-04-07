package keycloak

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRealm        = "test-realm"
	testClientID     = "test-client"
	testClientSecret = "test-secret"
)

// setupTestClient creates an AdminClient backed by the given handler.
// The handler receives all requests (both token and admin API).
func setupTestClient(t *testing.T, handler http.Handler) *AdminClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg := &Config{
		KeycloakURL:  server.URL,
		Realm:        testRealm,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	}
	return NewAdminClient(cfg)
}

// tokenHandler responds to token requests with a valid access token.
func tokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AdminToken{AccessToken: "test-token", ExpiresIn: 300})
}

// newMux creates a mux with the token endpoint pre-registered.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/"+testRealm+"/protocol/openid-connect/token", tokenHandler)
	return mux
}

// jsonResponse writes a JSON response with the correct Content-Type header.
func jsonResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- CreateUser ---

func TestCreateUser_Success(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("POST /admin/realms/"+testRealm+"/users", func(w http.ResponseWriter, r *http.Request) {
		var body UserRepresentation
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "john", body.Username)
		assert.Equal(t, "john@example.com", body.Email)

		w.Header().Set("Location", "/admin/realms/"+testRealm+"/users/user-123")
		w.WriteHeader(http.StatusCreated)
	})

	client := setupTestClient(t, mux)
	id, err := client.CreateUser(context.Background(), &domain.KeycloakUser{
		Username:  "john",
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Enabled:   true,
	})

	require.NoError(t, err)
	assert.Equal(t, "user-123", id)
}

func TestCreateUser_Conflict(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("POST /admin/realms/"+testRealm+"/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("User exists"))
	})

	client := setupTestClient(t, mux)
	_, err := client.CreateUser(context.Background(), &domain.KeycloakUser{
		Username: "john",
	})

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

func TestCreateUser_MissingLocationHeader(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("POST /admin/realms/"+testRealm+"/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	client := setupTestClient(t, mux)
	_, err := client.CreateUser(context.Background(), &domain.KeycloakUser{
		Username: "john",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing Location header")
}

// --- UpdateUser ---

func TestUpdateUser_MergesFields(t *testing.T) {
	newEmail := "new@example.com"
	putBodyCh := make(chan UserRepresentation, 1)

	mux := newMux()

	// GET returns the user — simulates Keycloak state that reflects the PUT
	currentEmail := "old@example.com"
	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, UserRepresentation{
			ID:        "user-123",
			Username:  "john",
			Email:     currentEmail,
			FirstName: "John",
			LastName:  "Doe",
			Enabled:   helpers.BoolPtr(true),
			Attributes: map[string][]string{
				"participant_id": {"p-1"},
			},
		})
	})

	// PUT receives the merged body and updates the state for the subsequent GET
	mux.HandleFunc("PUT /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		var body UserRepresentation
		json.NewDecoder(r.Body).Decode(&body)
		currentEmail = body.Email
		putBodyCh <- body
		w.WriteHeader(http.StatusNoContent)
	})

	// GET for role mappings (called by a.Get at the end)
	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{{ID: "r1", Name: "admin"}})
	})

	client := setupTestClient(t, mux)
	result, err := client.UpdateUser(context.Background(), "user-123", domain.UpdateKeycloakUserParams{
		Email: &newEmail,
	})

	require.NoError(t, err)

	// Verify PUT body merged: new email, but kept original firstName/lastName/enabled/attributes
	putBody := <-putBodyCh
	assert.Equal(t, "new@example.com", putBody.Email)
	assert.Equal(t, "John", putBody.FirstName)
	assert.Equal(t, "Doe", putBody.LastName)
	require.NotNil(t, putBody.Enabled)
	assert.True(t, *putBody.Enabled)
	assert.Equal(t, []string{"p-1"}, putBody.Attributes["participant_id"])

	// Verify returned domain object
	assert.Equal(t, "user-123", result.ID)
	assert.Equal(t, "new@example.com", result.Email)
	assert.Equal(t, []string{"admin"}, result.Roles)
}

func TestUpdateUser_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	_, err := client.UpdateUser(context.Background(), "", domain.UpdateKeycloakUserParams{})

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

func TestUpdateUser_NotFound(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/bad-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	client := setupTestClient(t, mux)
	_, err := client.UpdateUser(context.Background(), "bad-id", domain.UpdateKeycloakUserParams{})

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.NotFoundError{})
}

// --- DeleteUser ---

func TestDeleteUser_Success(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	client := setupTestClient(t, mux)
	err := client.DeleteUser(context.Background(), "user-123")
	require.NoError(t, err)
}

func TestDeleteUser_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	err := client.DeleteUser(context.Background(), "")

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

func TestDeleteUser_NotFound(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/bad-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	client := setupTestClient(t, mux)
	err := client.DeleteUser(context.Background(), "bad-id")

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.NotFoundError{})
}

// --- SetPassword ---

func TestSetPassword_Success(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("PUT /admin/realms/"+testRealm+"/users/user-123/reset-password", func(w http.ResponseWriter, r *http.Request) {
		var cred CredentialRepresentation
		json.NewDecoder(r.Body).Decode(&cred)
		assert.Equal(t, "password", cred.Type)
		assert.Equal(t, "secret123", cred.Value)
		assert.False(t, cred.Temporary)
		w.WriteHeader(http.StatusNoContent)
	})

	client := setupTestClient(t, mux)
	err := client.SetPassword(context.Background(), "user-123", "secret123", false)
	require.NoError(t, err)
}

func TestSetPassword_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	err := client.SetPassword(context.Background(), "", "pass", false)

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

// --- Get ---

func TestGet_Success(t *testing.T) {
	mux := newMux()

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, UserRepresentation{
			ID:        "user-123",
			Username:  "john",
			Email:     "john@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Enabled:   helpers.BoolPtr(true),
			Attributes: map[string][]string{
				"participant_id": {"p-1"},
				"agent_id":       {"a-1"},
			},
		})
	})

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{
			{ID: "r1", Name: "admin"},
			{ID: "r2", Name: "participant"},
		})
	})

	client := setupTestClient(t, mux)
	user, err := client.Get(context.Background(), "user-123")

	require.NoError(t, err)
	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "john", user.Username)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.True(t, user.Enabled)
	assert.Equal(t, []string{"admin", "participant"}, user.Roles)
	assert.Equal(t, "p-1", user.ParticipantID)
	assert.Equal(t, "a-1", user.AgentID)
}

func TestGet_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	_, err := client.Get(context.Background(), "")

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

func TestGet_NotFound(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/bad-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	client := setupTestClient(t, mux)
	_, err := client.Get(context.Background(), "bad-id")

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.NotFoundError{})
}

// --- List ---

func TestList_Success(t *testing.T) {
	mux := newMux()

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/count", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "john", r.URL.Query().Get("search"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("2"))
	})

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "john", r.URL.Query().Get("search"))
		assert.Equal(t, "10", r.URL.Query().Get("max"))
		assert.Equal(t, "0", r.URL.Query().Get("first"))
		assert.Equal(t, "true", r.URL.Query().Get("briefRepresentation"))

		jsonResponse(w, []UserRepresentation{
			{ID: "u1", Username: "john1", Email: "j1@test.com", FirstName: "John", LastName: "One"},
			{ID: "u2", Username: "john2", Email: "j2@test.com", FirstName: "John", LastName: "Two"},
		})
	})

	client := setupTestClient(t, mux)
	result, err := client.List(context.Background(), domain.KeycloakUserListParams{
		Search:   "john",
		Page:     1,
		PageSize: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(2), result.TotalItems)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "u1", result.Items[0].ID)
	assert.Equal(t, "john1", result.Items[0].Username)
	assert.Equal(t, "u2", result.Items[1].ID)
	assert.Equal(t, 1, result.TotalPages)
	assert.Equal(t, 1, result.CurrentPage)
	assert.False(t, result.HasNext)
	assert.False(t, result.HasPrev)
}

func TestList_PaginationOffset(t *testing.T) {
	mux := newMux()

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/count", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("25"))
	})

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users", func(w http.ResponseWriter, r *http.Request) {
		// Page 3 with pageSize 10 should be first=20
		assert.Equal(t, "20", r.URL.Query().Get("first"))
		assert.Equal(t, "10", r.URL.Query().Get("max"))
		jsonResponse(w, []UserRepresentation{})
	})

	client := setupTestClient(t, mux)
	result, err := client.List(context.Background(), domain.KeycloakUserListParams{
		Page:     3,
		PageSize: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(25), result.TotalItems)
	assert.Equal(t, 3, result.TotalPages)
	assert.Equal(t, 3, result.CurrentPage)
	assert.False(t, result.HasNext)
	assert.True(t, result.HasPrev)
}

// --- GetRealmRoles ---

func TestGetRealmRoles_Success(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("GET /admin/realms/"+testRealm+"/roles", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{
			{ID: "r1", Name: "admin"},
			{ID: "r2", Name: "participant"},
			{ID: "r3", Name: "agent"},
		})
	})

	client := setupTestClient(t, mux)
	roles, err := client.GetRealmRoles(context.Background())

	require.NoError(t, err)
	assert.Len(t, roles, 3)
	assert.Equal(t, "admin", roles[0].Name)
	assert.Equal(t, "r1", roles[0].ID)
}

// --- AssignRealmRoles ---

func TestAssignRealmRoles_Success(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("POST /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		var roles []domain.KeycloakRole
		json.NewDecoder(r.Body).Decode(&roles)
		assert.Len(t, roles, 1)
		assert.Equal(t, "admin", roles[0].Name)
		w.WriteHeader(http.StatusNoContent)
	})

	client := setupTestClient(t, mux)
	err := client.AssignRealmRoles(context.Background(), "user-123", []domain.KeycloakRole{
		{ID: "r1", Name: "admin"},
	})
	require.NoError(t, err)
}

func TestAssignRealmRoles_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	err := client.AssignRealmRoles(context.Background(), "", []domain.KeycloakRole{{ID: "r1", Name: "admin"}})

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

// --- RemoveRealmRoles ---

func TestRemoveRealmRoles_Success(t *testing.T) {
	mux := newMux()
	called := false
	mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	client := setupTestClient(t, mux)
	err := client.RemoveRealmRoles(context.Background(), "user-123", []domain.KeycloakRole{
		{ID: "r1", Name: "admin"},
	})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRemoveRealmRoles_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	err := client.RemoveRealmRoles(context.Background(), "", []domain.KeycloakRole{{ID: "r1", Name: "admin"}})

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

// --- ensureToken ---

func TestEnsureToken_CachesToken(t *testing.T) {
	tokenCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/"+testRealm+"/protocol/openid-connect/token", func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AdminToken{AccessToken: "cached-token", ExpiresIn: 300})
	})

	mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/u1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/u2", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	client := setupTestClient(t, mux)
	ctx := context.Background()

	// Two calls should only request one token
	_ = client.DeleteUser(ctx, "u1")
	_ = client.DeleteUser(ctx, "u2")

	assert.Equal(t, 1, tokenCalls, "token should be cached across requests")
}

func TestEnsureToken_FailureReturnsError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/"+testRealm+"/protocol/openid-connect/token", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid credentials"))
	})

	client := setupTestClient(t, mux)
	err := client.DeleteUser(context.Background(), "user-123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token request failed")
}

// --- GetUserRealmRoles ---

func TestGetUserRealmRoles_Success(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{
			{ID: "r1", Name: "admin"},
		})
	})

	client := setupTestClient(t, mux)
	roles, err := client.GetUserRealmRoles(context.Background(), "user-123")

	require.NoError(t, err)
	assert.Len(t, roles, 1)
	assert.Equal(t, "admin", roles[0].Name)
}

func TestGetUserRealmRoles_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	_, err := client.GetUserRealmRoles(context.Background(), "")

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}
