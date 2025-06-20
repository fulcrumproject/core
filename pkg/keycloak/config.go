package keycloak

import "fmt"

type Config struct {
	KeycloakURL    string `json:"keycloakUrl" env:"OAUTH_KEYCLOAK_URL"`
	Realm          string `json:"realm" env:"OAUTH_REALM"`
	ClientID       string `json:"clientId" env:"OAUTH_CLIENT_ID"`
	ClientSecret   string `json:"clientSecret" env:"OAUTH_CLIENT_SECRET"`
	JWKSCacheTTL   int    `json:"jwksCacheTtl" env:"OAUTH_JWKS_CACHE_TTL"`
	ValidateIssuer bool   `json:"validateIssuer" env:"OAUTH_VALIDATE_ISSUER"`
}

// GetJWKSURL returns the JWKS endpoint URL for the Keycloak realm
func (c *Config) GetJWKSURL() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid_connect/certs", c.KeycloakURL, c.Realm)
}

// GetIssuer returns the expected issuer for JWT tokens
func (c *Config) GetIssuer() string {
	return fmt.Sprintf("%s/realms/%s", c.KeycloakURL, c.Realm)
}
