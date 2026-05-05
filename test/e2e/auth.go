//go:build e2e

package e2e

import (
	"fmt"

	"resty.dev/v3"
)

func GetToken(username, password string) (string, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", keycloakURL, keycloakRealm)
	var out struct {
		AccessToken string `json:"access_token"`
	}
	resp, err := resty.New().R().SetFormData(map[string]string{
		"grant_type":    "password",
		"client_id":     oauthClientID,
		"client_secret": oauthSecret,
		"username":      username,
		"password":      password,
	}).SetResult(&out).Post(tokenURL)

	if err != nil {
		return "", fmt.Errorf("keycloak token: %w", err)
	}
	if resp.IsError() {
		return "", fmt.Errorf("keycloak token: %s", resp.String())
	}

	return out.AccessToken, nil
}
