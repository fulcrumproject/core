package keycloak

import (
	"context"
	"crypto/tls"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/helpers"
	"resty.dev/v3"
)

// AdminClient implements domain.KeycloakAdminClient using the Keycloak Admin REST API.
type AdminClient struct {
	config      *Config
	client      *resty.Client
	tokenClient *resty.Client
	token       string
	tokenExpiry time.Time
	tokenMu     sync.Mutex
}

// NewAdminClient creates a new AdminClient configured with the given Keycloak settings.
func NewAdminClient(cfg *Config) *AdminClient {
	client := resty.New().SetBaseURL(cfg.GetAdminUrl()).SetHeader("Content-Type", "application/json").SetError(&keycloakErrorBody{}).SetAllowMethodDeletePayload(true)
	tokenClient := resty.New()

	if cfg.InsecureSkipVerify {
		tlsConfig := &tls.Config{InsecureSkipVerify: true}
		client.SetTLSClientConfig(tlsConfig)
		tokenClient.SetTLSClientConfig(tlsConfig)
	}

	if cfg.RestyDebug {
		client.SetDebug(true)
	}

	ac := &AdminClient{
		config:      cfg,
		client:      client,
		tokenClient: tokenClient,
	}

	client.AddRequestMiddleware(func(c *resty.Client, r *resty.Request) error {
		token, err := ac.ensureToken(r.Context())
		if err != nil {
			return err
		}
		r.SetAuthToken(token)
		return nil
	})

	return ac

}

func (a *AdminClient) ensureToken(ctx context.Context) (string, error) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()

	if a.token != "" && time.Now().Add(30*time.Second).Before(a.tokenExpiry) {
		return a.token, nil
	}

	formData := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     a.config.ClientID,
		"client_secret": a.config.ClientSecret,
	}

	var tokenRes AdminToken
	res, err := a.tokenClient.R().SetContext(ctx).SetHeader("Content-Type", "application/x-www-form-urlencoded").SetFormData(formData).SetResult(&tokenRes).Post(a.config.GetTokenUrl())
	if err != nil {
		return "", err
	}

	if res.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("token request failed (status %d): %s", res.StatusCode(), res.String())
	}

	a.token = tokenRes.AccessToken
	a.tokenExpiry = time.Now().Add(time.Duration(tokenRes.ExpiresIn) * time.Second)

	return a.token, nil

}

// AdminToken represents an OAuth2 token response from Keycloak.
type AdminToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type keycloakErrorBody struct {
	ErrorMessage string `json:"errorMessage"`
}

// CredentialRepresentation is the Keycloak API representation of a user credential.
type CredentialRepresentation struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}

// UserRepresentation is the Keycloak API representation of a user.
type UserRepresentation struct {
	ID            string                     `json:"id,omitempty"`
	Username      string                     `json:"username,omitempty"`
	Email         string                     `json:"email,omitempty"`
	EmailVerified *bool                      `json:"emailVerified,omitempty"`
	FirstName     string                     `json:"firstName,omitempty"`
	LastName      string                     `json:"lastName,omitempty"`
	Enabled       *bool                      `json:"enabled,omitempty"`
	Attributes    map[string][]string        `json:"attributes,omitempty"`
	Credentials []CredentialRepresentation `json:"credentials,omitempty"`
}

func keycloakError(resp *resty.Response, action string) error {
	msg := ""
	if body, ok := resp.Error().(*keycloakErrorBody); ok && body.ErrorMessage != "" {
		msg = body.ErrorMessage
	}

	switch resp.StatusCode() {
	case http.StatusNotFound:
		if msg != "" {
			return domain.NewNotFoundErrorf("%s", msg)
		}
		return domain.NewNotFoundErrorf("keycloak user not found")
	case http.StatusConflict:
		if msg != "" {
			return domain.NewInvalidInputErrorf("%s", msg)
		}
		return domain.NewInvalidInputErrorf("keycloak user conflict")
	default:
		if msg != "" {
			return fmt.Errorf("keycloak %s failed (status %d): %s", action, resp.StatusCode(), msg)
		}
		return fmt.Errorf("keycloak %s failed (status %d): %s", action, resp.StatusCode(), resp.String())
	}
}

// Create creates a new user in Keycloak and returns the user ID.
func (a *AdminClient) Create(ctx context.Context, params domain.CreateKeycloakUserParams) (string, error) {
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
		return "", err
	}
	if resp.StatusCode() != http.StatusCreated {
		return "", keycloakError(resp, "create user")
	}
	location := resp.Header().Get("Location")
	if location == "" {
		return "", fmt.Errorf("keycloak create user: missing Location header")
	}
	parts := strings.Split(location, "/")
	return parts[len(parts)-1], nil
}

// Update updates an existing user in Keycloak and returns the updated user.
func (a *AdminClient) Update(ctx context.Context, id string, params domain.UpdateKeycloakUserParams) error {
	if id == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}

	var body UserRepresentation

	respUser, err := a.client.R().SetContext(ctx).SetPathParam("id", id).
		SetResult(&body).
		Get("/users/{id}")
	if err != nil {
		return err
	}

	if respUser.StatusCode() != http.StatusOK {
		return keycloakError(respUser, "get user for update")
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
		if params.ParticipantID != nil {
			if *params.ParticipantID == "" {
				body.Attributes["participant_id"] = []string{}
			} else {
				body.Attributes["participant_id"] = []string{*params.ParticipantID}
			}
		}
		if params.AgentID != nil {
			if *params.AgentID == "" {
				body.Attributes["agent_id"] = []string{}
			} else {
				body.Attributes["agent_id"] = []string{*params.AgentID}
			}
		}
	}

	resp, err := a.client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetBody(body).
		Put("/users/{id}")
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return keycloakError(resp, "update user")
	}
	return nil
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

// SetPassword sets or resets a user's password in Keycloak.
func (a *AdminClient) SetPassword(ctx context.Context, id string, password string, temporary bool) error {
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

// GetRealmRoles returns all realm-level roles from Keycloak.
func (a *AdminClient) GetRealmRoles(ctx context.Context) ([]domain.KeycloakRole, error) {
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

// AssignRealmRoles assigns realm roles to a user in Keycloak.
func (a *AdminClient) AssignRealmRoles(ctx context.Context, id string, roles []domain.KeycloakRole) error {
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

// RemoveRealmRoles removes realm roles from a user in Keycloak.
func (a *AdminClient) RemoveRealmRoles(ctx context.Context, id string, roles []domain.KeycloakRole) error {
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

// SetRole replaces the user's current app-managed role with the given role.
// It leaves Keycloak built-in roles (e.g. default-roles-*, offline_access) untouched.
func (a *AdminClient) SetRole(ctx context.Context, userID string, role string) error {
	if userID == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	if role == "" {
		return domain.NewInvalidInputErrorf("role is required")
	}

	realmRoles, err := a.GetRealmRoles(ctx)
	if err != nil {
		return err
	}
	var targetRole *domain.KeycloakRole
	for _, r := range realmRoles {
		if r.Name == role {
			targetRole = &r
			break
		}
	}
	if targetRole == nil {
		return domain.NewInvalidInputErrorf("realm role %s not found in Keycloak", role)
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
		if err := a.RemoveRealmRoles(ctx, userID, toRemove); err != nil {
			return err
		}
	}

	return a.AssignRealmRoles(ctx, userID, []domain.KeycloakRole{*targetRole})
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
	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		if auth.Role(r.Name).Validate() == nil {
			roleNames = append(roleNames, r.Name)
		}
	}

	var participantID string
	if vals, ok := user.Attributes["participant_id"]; ok && len(vals) > 0 {
		participantID = vals[0]
	}
	var agentID string
	if vals, ok := user.Attributes["agent_id"]; ok && len(vals) > 0 {
		agentID = vals[0]
	}

	enabled := false
	if user.Enabled != nil {
		enabled = *user.Enabled
	}

	emailVerified := false
	if user.EmailVerified != nil {
		emailVerified = *user.EmailVerified
	}

	return &domain.KeycloakUser{
		ID:            user.ID,
		Username:      user.Username,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Email:         user.Email,
		EmailVerified: emailVerified,
		Enabled:       enabled,
		Roles:         roleNames,
		ParticipantID: participantID,
		AgentID:       agentID,
	}, nil
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
