package keycloak

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AdminClient struct {
	Config      *Config
	Client      *http.Client
	Token       string
	TokenExpiry time.Time
}

func NewAdminClient(cfg *Config) *AdminClient {
	if cfg.InsecureSkipVerify {
		return &AdminClient{
			Config: cfg,
			Client: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			},
		}
	}

	return &AdminClient{
		Config: cfg,
		Client: &http.Client{},
	}

}

func (a *AdminClient) getToken(ctx context.Context) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", a.Config.ClientID)
	data.Set("client_secret", a.Config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", a.Config.GetTokenUrl(), strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenRes AdminToken

	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return "", err
	}

	a.Token = tokenRes.AccessToken
	a.TokenExpiry = time.Now().Add(time.Duration(tokenRes.ExpiresIn) * time.Second)

	return tokenRes.AccessToken, nil

}

type AdminToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (a *AdminClient) doRequest(ctx context.Context, method string, path string, body io.Reader) (*http.Response, error) {
	token, err := a.getToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, a.Config.GetAdminUrl()+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res, err := a.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
