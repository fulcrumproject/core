// Config defines optional server-owned signing and caller-role lifetime policy.
// An entirely unconfigured issuer leaves all existing Core authentication intact.
package jwtissuer

import (
	"fmt"
	"strings"
	"time"
)

const (
	DefaultParticipantTTL = 90 * time.Second
	DefaultAgentTTL       = 5 * time.Minute
)

// Config is shared by Core applications mounting the generic issuance handler.
// SigningKey is private server material and must never be logged or returned.
type Config struct {
	SigningKey     string        `json:"signingKey" env:"JWT_SIGNING_KEY"`
	KeyID          string        `json:"keyId" env:"JWT_KEY_ID"`
	Issuer         string        `json:"issuer" env:"JWT_ISSUER"`
	ParticipantTTL time.Duration `json:"participantTtl" env:"JWT_PARTICIPANT_TTL"`
	AgentTTL       time.Duration `json:"agentTtl" env:"JWT_AGENT_TTL"`
}

func (c Config) Enabled() bool {
	return c.SigningKey != "" || c.KeyID != "" || c.Issuer != ""
}

// Validate rejects partial signing settings and non-positive/subsecond TTLs.
// Zero means use the role default; configured durations must be whole seconds.
func (c Config) Validate() error {
	for _, ttl := range []time.Duration{c.ParticipantTTL, c.AgentTTL} {
		if ttl < 0 || ttl%time.Second != 0 {
			return fmt.Errorf("JWT lifetimes must be positive whole seconds (or zero for defaults)")
		}
	}
	if !c.Enabled() {
		if (c.ParticipantTTL != 0 && c.ParticipantTTL != DefaultParticipantTTL) || (c.AgentTTL != 0 && c.AgentTTL != DefaultAgentTTL) {
			return fmt.Errorf("JWT lifetime policy requires signing settings")
		}
		return nil
	}
	if strings.TrimSpace(c.SigningKey) == "" || strings.TrimSpace(c.KeyID) == "" || strings.TrimSpace(c.Issuer) == "" {
		return fmt.Errorf("JWT signingKey, keyId and issuer are all required")
	}
	return nil
}
