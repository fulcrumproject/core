package keycloak

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fulcrumproject/core/pkg/domain"
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

