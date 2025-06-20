package keycloak

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_GetJWKSURL(t *testing.T) {
	config := &Config{
		KeycloakURL: "https://keycloak.example.com",
		Realm:       "test-realm",
	}

	expected := "https://keycloak.example.com/realms/test-realm/protocol/openid_connect/certs"
	actual := config.GetJWKSURL()

	assert.Equal(t, expected, actual, "JWKS URL should match expected value")
}

func TestConfig_GetIssuer(t *testing.T) {
	config := &Config{
		KeycloakURL: "https://keycloak.example.com",
		Realm:       "test-realm",
	}

	expected := "https://keycloak.example.com/realms/test-realm"
	actual := config.GetIssuer()

	assert.Equal(t, expected, actual, "Issuer should match expected value")
}
