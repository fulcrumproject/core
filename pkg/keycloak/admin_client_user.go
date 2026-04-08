package keycloak

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"strconv"
	"strings"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/helpers"
)

// Create creates a new user in Keycloak, assigns the role, and returns the hydrated user.
func (a *AdminClient) Create(ctx context.Context, params domain.CreateKeycloakUserParams) (*domain.KeycloakUser, error) {
	attrs := map[string][]string{}
	if params.ParticipantID != "" {
		attrs["participant_id"] = []string{params.ParticipantID}
	}
	if params.AgentID != "" {
		attrs["agent_id"] = []string{params.AgentID}
	}

	enabled := helpers.BoolPtr(params.Enabled)
	emailVerified := helpers.BoolPtr(params.EmailVerified)
	body := UserRepresentation{
		Username:      params.Username,
		Email:         params.Email,
		EmailVerified: emailVerified,
		FirstName:     params.FirstName,
		LastName:      params.LastName,
		Enabled:       enabled,
		Attributes:    attrs,
	}
	if params.Password != "" {
		body.Credentials = []CredentialRepresentation{
			{Type: "password", Value: params.Password, Temporary: false},
		}
	}
	resp, err := a.client.R().
		SetContext(ctx).
		SetBody(body).
		Post("/users")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusCreated {
		return nil, keycloakError(resp, "create user")
	}
	location := resp.Header().Get("Location")
	if location == "" {
		return nil, fmt.Errorf("keycloak create user: missing Location header")
	}
	parts := strings.Split(location, "/")
	userID := parts[len(parts)-1]

	// Keycloak's POST /users endpoint ignores realmRoles in the request body.
	// Roles must be assigned via a separate role-mappings call.
	if err := a.assignRole(ctx, userID, string(params.Role)); err != nil {
		a.compensatingDelete(ctx, userID)
		return nil, err
	}

	body.ID = userID
	return buildKeycloakUser(body, []auth.Role{params.Role}), nil
}

// Update updates an existing user in Keycloak and returns the updated user.
func (a *AdminClient) Update(ctx context.Context, id string, params domain.UpdateKeycloakUserParams) (*domain.KeycloakUser, error) {
	if id == "" {
		return nil, domain.NewInvalidInputErrorf("keycloak user id is required")
	}

	var body UserRepresentation

	respUser, err := a.client.R().SetContext(ctx).SetPathParam("id", id).
		SetResult(&body).
		Get("/users/{id}")
	if err != nil {
		return nil, err
	}

	if respUser.StatusCode() != http.StatusOK {
		return nil, keycloakError(respUser, "get user for update")
	}

	if params.Email != nil {
		body.Email = *params.Email
	}

	if params.FirstName != nil {
		body.FirstName = *params.FirstName
	}

	if params.LastName != nil {
		body.LastName = *params.LastName
	}

	if params.Enabled != nil {
		body.Enabled = params.Enabled
	}

	if params.ParticipantID != nil || params.AgentID != nil {
		if body.Attributes == nil {
			body.Attributes = make(map[string][]string)
		}
		setAttribute(body.Attributes, "participant_id", params.ParticipantID)
		setAttribute(body.Attributes, "agent_id", params.AgentID)
	}

	resp, err := a.client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetBody(body).
		Put("/users/{id}")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return nil, keycloakError(resp, "update user")
	}

	// Determine final roles for the return value
	var roles []auth.Role
	if params.Role != nil {
		if err := a.setRole(ctx, id, string(*params.Role)); err != nil {
			return nil, err
		}
		roles = []auth.Role{*params.Role}
	} else {
		// Role unchanged: fetch current app-managed roles
		currentRoles, err := a.getUserRealmRoles(ctx, id)
		if err != nil {
			return nil, err
		}
		roles = filterAppManagedRoles(currentRoles)
	}

	if params.Password != nil {
		if err := a.setPassword(ctx, id, *params.Password, false); err != nil {
			return nil, err
		}
	}

	return buildKeycloakUser(body, roles), nil
}

// Delete deletes a user from Keycloak.
func (a *AdminClient) Delete(ctx context.Context, id string) error {
	if id == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	resp, err := a.client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		Delete("/users/{id}")
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return keycloakError(resp, "delete user")
	}
	return nil
}

// setPassword sets or resets a user's password in Keycloak.
func (a *AdminClient) setPassword(ctx context.Context, id string, password string, temporary bool) error {
	if id == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	cred := CredentialRepresentation{
		Type:      "password",
		Value:     password,
		Temporary: temporary,
	}
	resp, err := a.client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetBody(cred).
		Put("/users/{id}/reset-password")
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return keycloakError(resp, "set password")
	}
	return nil
}

// getRealmRoles returns all realm-level roles from Keycloak.
func (a *AdminClient) getRealmRoles(ctx context.Context) ([]domain.KeycloakRole, error) {
	var roles []domain.KeycloakRole
	resp, err := a.client.R().
		SetContext(ctx).
		SetResult(&roles).
		Get("/roles")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, keycloakError(resp, "get realm roles")
	}
	return roles, nil
}

// assignRealmRoles assigns realm roles to a user in Keycloak.
func (a *AdminClient) assignRealmRoles(ctx context.Context, id string, roles []domain.KeycloakRole) error {
	if id == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	resp, err := a.client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetBody(roles).
		Post("/users/{id}/role-mappings/realm")
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return keycloakError(resp, "assign realm roles")
	}
	return nil
}

// removeRealmRoles removes realm roles from a user in Keycloak.
func (a *AdminClient) removeRealmRoles(ctx context.Context, id string, roles []domain.KeycloakRole) error {
	if id == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	resp, err := a.client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetBody(roles).
		Delete("/users/{id}/role-mappings/realm")
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return keycloakError(resp, "remove realm roles")
	}
	return nil
}

// setRole replaces the user's current app-managed role with the given role.
// It leaves Keycloak built-in roles (e.g. default-roles-*, offline_access) untouched.
func (a *AdminClient) setRole(ctx context.Context, userID string, role string) error {
	if userID == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	if role == "" {
		return domain.NewInvalidInputErrorf("role is required")
	}

	targetRole, err := a.findRealmRole(ctx, role)
	if err != nil {
		return err
	}

	currentRoles, err := a.getUserRealmRoles(ctx, userID)
	if err != nil {
		return err
	}

	var toRemove []domain.KeycloakRole
	for _, r := range currentRoles {
		if auth.Role(r.Name).Validate() == nil {
			toRemove = append(toRemove, r)
		}
	}
	if len(toRemove) > 0 {
		if err := a.removeRealmRoles(ctx, userID, toRemove); err != nil {
			return err
		}
	}

	return a.assignRealmRoles(ctx, userID, []domain.KeycloakRole{*targetRole})
}

// assignRole assigns a role to a newly created user (no existing roles to remove).
func (a *AdminClient) assignRole(ctx context.Context, userID string, role string) error {
	targetRole, err := a.findRealmRole(ctx, role)
	if err != nil {
		return err
	}
	return a.assignRealmRoles(ctx, userID, []domain.KeycloakRole{*targetRole})
}

// findRealmRole looks up a realm role by name from Keycloak.
func (a *AdminClient) findRealmRole(ctx context.Context, name string) (*domain.KeycloakRole, error) {
	realmRoles, err := a.getRealmRoles(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range realmRoles {
		if r.Name == name {
			return &r, nil
		}
	}
	return nil, domain.NewInvalidInputErrorf("realm role %s not found in Keycloak", name)
}

// filterAppManagedRoles extracts app-managed role names from a list of Keycloak roles,
// filtering out Keycloak built-in roles (e.g. default-roles-*, offline_access).
func filterAppManagedRoles(roles []domain.KeycloakRole) []auth.Role {
	var names []auth.Role
	for _, r := range roles {
		if auth.Role(r.Name).Validate() == nil {
			names = append(names, auth.Role(r.Name))
		}
	}
	return names
}

// setAttribute sets or clears a single attribute in a Keycloak attributes map.
func setAttribute(attrs map[string][]string, key string, val *string) {
	if val == nil {
		return
	}
	if *val == "" {
		attrs[key] = []string{}
	} else {
		attrs[key] = []string{*val}
	}
}

func (a *AdminClient) compensatingDelete(ctx context.Context, userID string) {
	if err := a.Delete(ctx, userID); err != nil {
		slog.Error("failed compensating delete of keycloak user", "userID", userID, "error", err)
	}
}

// buildKeycloakUser constructs a domain.KeycloakUser from a Keycloak representation and known roles.
func buildKeycloakUser(rep UserRepresentation, roles []auth.Role) *domain.KeycloakUser {
	var participantID string
	if vals, ok := rep.Attributes["participant_id"]; ok && len(vals) > 0 {
		participantID = vals[0]
	}
	var agentID string
	if vals, ok := rep.Attributes["agent_id"]; ok && len(vals) > 0 {
		agentID = vals[0]
	}
	enabled := false
	if rep.Enabled != nil {
		enabled = *rep.Enabled
	}
	emailVerified := false
	if rep.EmailVerified != nil {
		emailVerified = *rep.EmailVerified
	}
	return &domain.KeycloakUser{
		ID:            rep.ID,
		Username:      rep.Username,
		FirstName:     rep.FirstName,
		LastName:      rep.LastName,
		Email:         rep.Email,
		EmailVerified: emailVerified,
		Enabled:       enabled,
		Roles:         roles,
		ParticipantID: participantID,
		AgentID:       agentID,
	}
}

func (a *AdminClient) getUserRealmRoles(ctx context.Context, id string) ([]domain.KeycloakRole, error) {
	if id == "" {
		return nil, domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	var roles []domain.KeycloakRole
	resp, err := a.client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetResult(&roles).
		Get("/users/{id}/role-mappings/realm")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, keycloakError(resp, "get user realm roles")
	}
	return roles, nil
}

// Get retrieves a single keycloak user by ID, including their realm roles.
func (a *AdminClient) Get(ctx context.Context, id string) (*domain.KeycloakUser, error) {
	if id == "" {
		return nil, domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	var user UserRepresentation
	resp, err := a.client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetResult(&user).
		Get("/users/{id}")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, keycloakError(resp, "get user")
	}

	roles, err := a.getUserRealmRoles(ctx, id)
	if err != nil {
		return nil, err
	}

	return buildKeycloakUser(user, filterAppManagedRoles(roles)), nil
}

// List retrieves a paginated list of keycloak users.
func (a *AdminClient) List(ctx context.Context, params domain.KeycloakUserListParams) (*domain.PageRes[domain.KeycloakUserListItem], error) {
	first := (params.Page - 1) * params.PageSize

	countParams := make(map[string]string)
	if params.FirstName != "" {
		countParams["firstName"] = params.FirstName
	}

	if params.Email != "" {
		countParams["email"] = params.Email
	}

	if params.LastName != "" {
		countParams["lastName"] = params.LastName
	}

	listParams := map[string]string{
		"max":                 strconv.Itoa(params.PageSize),
		"first":               strconv.Itoa(first),
		"briefRepresentation": "true",
	}

	maps.Copy(listParams, countParams)

	var userCount int
	respCount, err := a.client.R().SetContext(ctx).SetQueryParams(countParams).SetResult(&userCount).Get("/users/count")
	if err != nil {
		return nil, err
	}
	if respCount.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("keycloak admin API error (status %d): %s", respCount.StatusCode(), respCount.String())
	}

	var users []UserRepresentation
	resp, err := a.client.R().SetContext(ctx).SetQueryParams(listParams).SetResult(&users).Get("/users")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("keycloak admin API error (status %d): %s", resp.StatusCode(), resp.String())
	}
	keycloakUsers := make([]domain.KeycloakUserListItem, 0, len(users))
	for _, user := range users {
		keycloakUsers = append(keycloakUsers, domain.KeycloakUserListItem{
			ID:        user.ID,
			Username:  user.Username,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Email:     user.Email,
		})
	}

	page := &domain.PageReq{Page: params.Page, PageSize: params.PageSize}
	return domain.NewPaginatedResult(keycloakUsers, int64(userCount), page), nil
}
