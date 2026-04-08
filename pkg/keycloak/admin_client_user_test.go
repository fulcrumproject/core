package keycloak

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Create ---

func TestCreate_Success(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("POST /admin/realms/"+testRealm+"/users", func(w http.ResponseWriter, r *http.Request) {
		var body UserRepresentation
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "john", body.Username)
		assert.Equal(t, "john@example.com", body.Email)
		require.Len(t, body.Credentials, 1)
		assert.Equal(t, "password", body.Credentials[0].Type)
		assert.Equal(t, "secret123", body.Credentials[0].Value)
		assert.False(t, body.Credentials[0].Temporary)

		w.Header().Set("Location", "/admin/realms/"+testRealm+"/users/user-123")
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/roles", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{
			{ID: "r1", Name: "admin"},
			{ID: "r2", Name: "participant"},
		})
	})

	mux.HandleFunc("POST /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		var roles []domain.KeycloakRole
		json.NewDecoder(r.Body).Decode(&roles)
		assert.Len(t, roles, 1)
		assert.Equal(t, "admin", roles[0].Name)
		w.WriteHeader(http.StatusNoContent)
	})

	client := setupTestClient(t, mux)
	user, err := client.Create(context.Background(), domain.CreateKeycloakUserParams{
		Username:  "john",
		Email:     "john@example.com",
		FirstName: "John",
		LastName:  "Doe",
		Password:  "secret123",
		Enabled:   true,
		Role:      "admin",
	})

	require.NoError(t, err)
	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "john", user.Username)
	assert.Equal(t, []string{"admin"}, user.Roles)
}

func TestCreate_Conflict(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("POST /admin/realms/"+testRealm+"/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("User exists"))
	})

	client := setupTestClient(t, mux)
	_, err := client.Create(context.Background(), domain.CreateKeycloakUserParams{
		Username: "john",
		Role:     "admin",
	})

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

func TestCreate_MissingLocationHeader(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("POST /admin/realms/"+testRealm+"/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	client := setupTestClient(t, mux)
	_, err := client.Create(context.Background(), domain.CreateKeycloakUserParams{
		Username: "john",
		Role:     "admin",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing Location header")
}

// --- Update ---

func TestUpdate_MergesFields(t *testing.T) {
	newEmail := "new@example.com"
	putBodyCh := make(chan UserRepresentation, 1)

	mux := newMux()

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, UserRepresentation{
			ID:        "user-123",
			Username:  "john",
			Email:     "old@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Enabled:   helpers.BoolPtr(true),
			Attributes: map[string][]string{
				"participant_id": {"p-1"},
			},
		})
	})

	mux.HandleFunc("PUT /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		var body UserRepresentation
		json.NewDecoder(r.Body).Decode(&body)
		putBodyCh <- body
		w.WriteHeader(http.StatusNoContent)
	})

	// Role-mappings endpoint needed for return value (no role change, so fetches current roles)
	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{{ID: "r1", Name: "admin"}})
	})

	client := setupTestClient(t, mux)
	user, err := client.Update(context.Background(), "user-123", domain.UpdateKeycloakUserParams{
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

	// Verify returned user is built from known data
	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "new@example.com", user.Email)
	assert.Equal(t, []string{"admin"}, user.Roles)
}

func TestUpdate_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	_, err := client.Update(context.Background(), "", domain.UpdateKeycloakUserParams{})

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

func TestUpdate_NotFound(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/bad-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	client := setupTestClient(t, mux)
	_, err := client.Update(context.Background(), "bad-id", domain.UpdateKeycloakUserParams{})

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.NotFoundError{})
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	client := setupTestClient(t, mux)
	err := client.Delete(context.Background(), "user-123")
	require.NoError(t, err)
}

func TestDelete_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	err := client.Delete(context.Background(), "")

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

func TestDelete_NotFound(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/bad-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	client := setupTestClient(t, mux)
	err := client.Delete(context.Background(), "bad-id")

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
	err := client.setPassword(context.Background(), "user-123", "secret123", false)
	require.NoError(t, err)
}

func TestSetPassword_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	err := client.setPassword(context.Background(), "", "pass", false)

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

func TestGet_FiltersSystemRoles(t *testing.T) {
	mux := newMux()

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, UserRepresentation{
			ID:       "user-123",
			Username: "john",
			Email:    "john@example.com",
			Enabled:  helpers.BoolPtr(true),
		})
	})

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{
			{ID: "r1", Name: "admin"},
			{ID: "r2", Name: "default-roles-myrealm"},
			{ID: "r3", Name: "offline_access"},
		})
	})

	client := setupTestClient(t, mux)
	user, err := client.Get(context.Background(), "user-123")

	require.NoError(t, err)
	assert.Equal(t, []string{"admin"}, user.Roles, "only app roles should be returned, system roles must be filtered out")
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
		assert.Equal(t, "john", r.URL.Query().Get("firstName"))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("2"))
	})

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "john", r.URL.Query().Get("firstName"))
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
		FirstName: "john",
		Page:      1,
		PageSize:  10,
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
	roles, err := client.getRealmRoles(context.Background())

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
	err := client.assignRealmRoles(context.Background(), "user-123", []domain.KeycloakRole{
		{ID: "r1", Name: "admin"},
	})
	require.NoError(t, err)
}

func TestAssignRealmRoles_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	err := client.assignRealmRoles(context.Background(), "", []domain.KeycloakRole{{ID: "r1", Name: "admin"}})

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
	err := client.removeRealmRoles(context.Background(), "user-123", []domain.KeycloakRole{
		{ID: "r1", Name: "admin"},
	})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRemoveRealmRoles_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	err := client.removeRealmRoles(context.Background(), "", []domain.KeycloakRole{{ID: "r1", Name: "admin"}})

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

// --- SetRole ---

func TestSetRole_Success(t *testing.T) {
	mux := newMux()

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/roles", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{
			{ID: "r1", Name: "admin"},
			{ID: "r2", Name: "participant"},
			{ID: "r3", Name: "agent"},
		})
	})

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{
			{ID: "r1", Name: "admin"},
			{ID: "r99", Name: "default-roles-myrealm"},
		})
	})

	removeCalled := false
	mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		removeCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		var roles []domain.KeycloakRole
		json.NewDecoder(r.Body).Decode(&roles)
		assert.Len(t, roles, 1)
		assert.Equal(t, "participant", roles[0].Name)
		assert.Equal(t, "r2", roles[0].ID)
		w.WriteHeader(http.StatusNoContent)
	})

	client := setupTestClient(t, mux)
	err := client.setRole(context.Background(), "user-123", "participant")

	require.NoError(t, err)
	assert.True(t, removeCalled)
}

func TestSetRole_RoleNotFound(t *testing.T) {
	mux := newMux()
	mux.HandleFunc("GET /admin/realms/"+testRealm+"/roles", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{
			{ID: "r1", Name: "admin"},
		})
	})

	client := setupTestClient(t, mux)
	err := client.setRole(context.Background(), "user-123", "nonexistent")

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestSetRole_EmptyID(t *testing.T) {
	client := setupTestClient(t, newMux())
	err := client.setRole(context.Background(), "", "admin")

	require.Error(t, err)
	assert.ErrorAs(t, err, &domain.InvalidInputError{})
}

// --- Update with attributes ---

func TestUpdate_SetsAttributes(t *testing.T) {
	newParticipantID := "p-new"
	putBodyCh := make(chan UserRepresentation, 1)

	mux := newMux()

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, UserRepresentation{
			ID:       "user-123",
			Username: "john",
			Enabled:  helpers.BoolPtr(true),
			Attributes: map[string][]string{
				"participant_id": {"p-old"},
			},
		})
	})

	mux.HandleFunc("PUT /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		var body UserRepresentation
		json.NewDecoder(r.Body).Decode(&body)
		putBodyCh <- body
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{{ID: "r1", Name: "participant"}})
	})

	client := setupTestClient(t, mux)
	user, err := client.Update(context.Background(), "user-123", domain.UpdateKeycloakUserParams{
		ParticipantID: &newParticipantID,
	})

	require.NoError(t, err)

	putBody := <-putBodyCh
	assert.Equal(t, []string{"p-new"}, putBody.Attributes["participant_id"])
	assert.Equal(t, "p-new", user.ParticipantID)
}

func TestUpdate_ClearsAttributes(t *testing.T) {
	empty := ""
	putBodyCh := make(chan UserRepresentation, 1)

	mux := newMux()

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, UserRepresentation{
			ID:       "user-123",
			Username: "john",
			Enabled:  helpers.BoolPtr(true),
			Attributes: map[string][]string{
				"participant_id": {"p-1"},
				"agent_id":       {"a-1"},
			},
		})
	})

	mux.HandleFunc("PUT /admin/realms/"+testRealm+"/users/user-123", func(w http.ResponseWriter, r *http.Request) {
		var body UserRepresentation
		json.NewDecoder(r.Body).Decode(&body)
		putBodyCh <- body
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /admin/realms/"+testRealm+"/users/user-123/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []domain.KeycloakRole{{ID: "r1", Name: "admin"}})
	})

	client := setupTestClient(t, mux)
	user, err := client.Update(context.Background(), "user-123", domain.UpdateKeycloakUserParams{
		ParticipantID: &empty,
		AgentID:       &empty,
	})

	require.NoError(t, err)

	putBody := <-putBodyCh
	assert.Equal(t, []string{}, putBody.Attributes["participant_id"], "participant_id should be cleared")
	assert.Equal(t, []string{}, putBody.Attributes["agent_id"], "agent_id should be cleared")
	assert.Equal(t, "", user.ParticipantID)
	assert.Equal(t, "", user.AgentID)
}
