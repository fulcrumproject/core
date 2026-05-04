package e2e

import (
	"fmt"

	"resty.dev/v3"
)

const keycloakTokenURL = "http://localhost:8080/realms/fulcrum/protocol/openid-connect/token"

func GetToken(username, password string) (string, error) {
	var out struct {
		AccessToken string `json:"access_token"`
	}
	resp, err := resty.New().R().SetFormData(map[string]string{
		"grant_type": "password",
		"client_id":  "fulcrum-ui",
		"username":   username,
		"password":   password,
	}).SetResult(&out).Post(keycloakTokenURL)

	if err != nil {
		return "", fmt.Errorf("keycloak token: %w", err)
	}
	if resp.IsError() {
		return "", fmt.Errorf("keycloak token: %s", resp.String())
	}

	return out.AccessToken, nil
}
