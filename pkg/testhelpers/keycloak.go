package testhelpers

import (
	"fmt"

	"resty.dev/v3"
)

type Realm struct {
	URL      string
	Name     string
	ClientID string
	Secret   string
}

func (r Realm) GetToken(username, password string) (string, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", r.URL, r.Name)
	var out struct {
		AccessToken string `json:"access_token"`
	}
	resp, err := resty.New().R().SetFormData(map[string]string{
		"grant_type":    "password",
		"client_id":     r.ClientID,
		"client_secret": r.Secret,
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
