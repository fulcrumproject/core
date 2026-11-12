// Config defines the optional server-owned CMP signing settings.
// A completely unconfigured signer leaves existing authentication unchanged.
package sso

import (
	"fmt"
	"strings"
	"time"
)

const DefaultTokenTTL = 90 * time.Second

// Config is reusable by Core applications that expose CMP launch tokens.
// SigningKey must never be logged or returned in an API response.
type Config struct {
	SigningKey string        `json:"signingKey" env:"CMP_SSO_SIGNING_KEY"`
	KeyID      string        `json:"keyId" env:"CMP_SSO_KEY_ID"`
	Issuer     string        `json:"issuer" env:"CMP_SSO_ISSUER"`
	TokenTTL   time.Duration `json:"tokenTtl" env:"CMP_SSO_TOKEN_TTL"`
}

// Enabled reports whether any key/identity setting requests signing.
func (c Config) Enabled() bool {
	return c.SigningKey != "" || c.KeyID != "" || c.Issuer != ""
}

// Validate rejects partial configuration. Zero TTL uses the 90-second default.
func (c Config) Validate() error {
	if c.TokenTTL < 0 {
		return fmt.Errorf("FULCRUM_CMP_SSO_TOKEN_TTL must be > 0 (or omitted for 90s)")
	}
	if !c.Enabled() {
		if c.TokenTTL != 0 && c.TokenTTL != DefaultTokenTTL {
			return fmt.Errorf("CMP SSO token TTL configured without signing settings")
		}
		return nil
	}
	if strings.TrimSpace(c.SigningKey) == "" || strings.TrimSpace(c.KeyID) == "" || strings.TrimSpace(c.Issuer) == "" {
		return fmt.Errorf("FULCRUM_CMP_SSO_SIGNING_KEY, _KEY_ID and _ISSUER are all required")
	}
	return nil
}
