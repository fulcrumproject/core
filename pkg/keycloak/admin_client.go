package keycloak

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fulcrumproject/core/pkg/domain"
	"resty.dev/v3"
)

type AdminClient struct {
	Config      *Config
	Client      *resty.Client
	tokenClient *resty.Client
	Token       string
	TokenExpiry time.Time
}

func NewAdminClient(cfg *Config) *AdminClient {
	client := resty.New().SetBaseURL(cfg.GetAdminUrl()).SetHeader("Content-Type", "application/json")
	tokenClient := resty.New()

	if cfg.InsecureSkipVerify {
		tlsConfig := &tls.Config{InsecureSkipVerify: true}
		client.SetTLSClientConfig(tlsConfig)
		tokenClient.SetTLSClientConfig(tlsConfig)
	}

	ac := &AdminClient{
		Config:      cfg,
		Client:      client,
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
	if a.Token != "" && time.Now().Add(30*time.Second).Before(a.TokenExpiry) {
		return a.Token, nil
	}

	formData := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     a.Config.ClientID,
		"client_secret": a.Config.ClientSecret,
	}

	var tokenRes AdminToken
	res, err := a.tokenClient.R().SetContext(ctx).SetHeader("Content-Type", "application/x-www-form-urlencoded").SetFormData(formData).SetResult(&tokenRes).Post(a.Config.GetTokenUrl())
	if err != nil {
		return "", err
	}

	if res.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("token request failed (status %d): %s", res.StatusCode(), res.String())
	}

	a.Token = tokenRes.AccessToken
	a.TokenExpiry = time.Now().Add(time.Duration(tokenRes.ExpiresIn) * time.Second)

	return a.Token, nil

}

type AdminToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type CredentialRepresentation struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}

type RoleRepresentation struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserRepresentation struct {
	ID          string                     `json:"id,omitempty"`
	Username    string                     `json:"username,omitempty"`
	Email       string                     `json:"email,omitempty"`
	FirstName   string                     `json:"firstName,omitempty"`
	LastName    string                     `json:"lastName,omitempty"`
	Enabled     *bool                      `json:"enabled,omitempty"`
	Attributes  map[string][]string        `json:"attributes,omitempty"`
	RealmRoles  []string                   `json:"realmRoles,omitempty"`
	Credentials []CredentialRepresentation `json:"credentials,omitempty"`
}

func keycloakError(resp *resty.Response, action string) error {
	switch resp.StatusCode() {
	case http.StatusNotFound:
		return domain.NewNotFoundErrorf("keycloak user not found")
	case http.StatusConflict:
		return domain.NewInvalidInputErrorf("keycloak user conflict: %s", resp.String())
	default:
		return fmt.Errorf("keycloak %s failed (status %d): %s", action, resp.StatusCode(), resp.String())
	}
}

func (a *AdminClient) CreateUser(ctx context.Context, user domain.KeycloakUserCreateRequest) (string, error) {
	enabled := user.Enabled
	body := UserRepresentation{
		Username:   user.Username,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Enabled:    &enabled,
		Attributes: user.Attributes,
	}
	resp, err := a.Client.R().
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

func (a *AdminClient) UpdateUser(ctx context.Context, id string, user domain.KeycloakUserUpdateRequest) (*domain.KeycloakUser, error) {
	if id == "" {
		return nil, domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	body := UserRepresentation{
		Email:      stringPtrValue(user.Email),
		FirstName:  stringPtrValue(user.FirstName),
		LastName:   stringPtrValue(user.LastName),
		Enabled:    user.Enabled,
		Attributes: user.Attributes,
	}
	resp, err := a.Client.R().
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
	return a.Get(ctx, id)
}

func stringPtrValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (a *AdminClient) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	resp, err := a.Client.R().
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

func (a *AdminClient) SetPassword(ctx context.Context, id string, password string, temporary bool) error {
	if id == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	cred := CredentialRepresentation{
		Type:      "password",
		Value:     password,
		Temporary: temporary,
	}
	resp, err := a.Client.R().
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

func (a *AdminClient) GetRealmRoles(ctx context.Context) ([]domain.KeycloakRole, error) {
	var roles []RoleRepresentation
	resp, err := a.Client.R().
		SetContext(ctx).
		SetResult(&roles).
		Get("/roles")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, keycloakError(resp, "get realm roles")
	}
	result := make([]domain.KeycloakRole, 0, len(roles))
	for _, r := range roles {
		result = append(result, domain.KeycloakRole{ID: r.ID, Name: r.Name})
	}
	return result, nil
}

func toRoleRepresentations(roles []domain.KeycloakRole) []RoleRepresentation {
	reps := make([]RoleRepresentation, 0, len(roles))
	for _, r := range roles {
		reps = append(reps, RoleRepresentation{ID: r.ID, Name: r.Name})
	}
	return reps
}

func (a *AdminClient) AssignRealmRoles(ctx context.Context, id string, roles []domain.KeycloakRole) error {
	if id == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	resp, err := a.Client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetBody(toRoleRepresentations(roles)).
		Post("/users/{id}/role-mappings/realm")
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return keycloakError(resp, "assign realm roles")
	}
	return nil
}

func (a *AdminClient) RemoveRealmRoles(ctx context.Context, id string, roles []domain.KeycloakRole) error {
	if id == "" {
		return domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	resp, err := a.Client.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetBody(toRoleRepresentations(roles)).
		Delete("/users/{id}/role-mappings/realm")
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return keycloakError(resp, "remove realm roles")
	}
	return nil
}

func (a *AdminClient) GetUserRealmRoles(ctx context.Context, id string) ([]RoleRepresentation, error) {
	if id == "" {
		return nil, domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	var roles []RoleRepresentation
	resp, err := a.Client.R().
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

func (a *AdminClient) Get(ctx context.Context, id string) (*domain.KeycloakUser, error) {
	if id == "" {
		return nil, domain.NewInvalidInputErrorf("keycloak user id is required")
	}
	var user UserRepresentation
	resp, err := a.Client.R().
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

	roles, err := a.GetUserRealmRoles(ctx, id)
	if err != nil {
		return nil, err
	}
	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		roleNames = append(roleNames, r.Name)
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

	return &domain.KeycloakUser{
		ID:            user.ID,
		Username:      user.Username,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Email:         user.Email,
		Enabled:       enabled,
		Roles:         roleNames,
		ParticipantID: participantID,
		AgentID:       agentID,
	}, nil
}

func (a *AdminClient) List(ctx context.Context, params domain.KeycloakUserListParams) (domain.KeycloakUserPaginatedRes, error) {
	first := (params.Page - 1) * params.PageSize

	listParams := map[string]string{
		"search":              params.Search,
		"max":                 strconv.Itoa(params.PageSize),
		"first":               strconv.Itoa(first),
		"briefRepresentation": "true",
	}
	countParams := map[string]string{
		"search": params.Search,
	}

	var userCount int
	respCount, err := a.Client.R().SetQueryParams(countParams).SetResult(&userCount).Get("/users/count")
	if err != nil {
		return domain.KeycloakUserPaginatedRes{}, err
	}
	if respCount.StatusCode() != http.StatusOK {
		return domain.KeycloakUserPaginatedRes{}, fmt.Errorf("keycloak admin API error (status %d): %s", respCount.StatusCode(), respCount.String())
	}

	var users []UserRepresentation
	resp, err := a.Client.R().SetQueryParams(listParams).SetResult(&users).Get("/users")
	if err != nil {
		return domain.KeycloakUserPaginatedRes{}, err
	}
	if resp.StatusCode() != http.StatusOK {
		return domain.KeycloakUserPaginatedRes{}, fmt.Errorf("keycloak admin API error (status %d): %s", resp.StatusCode(), resp.String())
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

	return domain.KeycloakUserPaginatedRes{
		Items:      keycloakUsers,
		TotalItems: userCount,
	}, nil
}
