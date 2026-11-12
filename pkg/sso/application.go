// Application tokens bind an agent to an operator-owned audience and module roles.
// They share the CMP signer and public key, but never its browser claims or TTL.
package sso

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
)

const DefaultApplicationTTL = 5 * time.Minute

// ApplicationConfig is server policy, never participant-writable agent configuration.
// Grants use Core's JSON configuration file; the builder does not support env maps.
type ApplicationConfig struct {
	TokenTTL time.Duration               `json:"tokenTtl" env:"APPLICATION_TOKEN_TTL"`
	Grants   map[string]ApplicationGrant `json:"grants"`
}

type ApplicationGrant struct {
	Audience string   `json:"audience"`
	Roles    []string `json:"roles"`
}

// ApplicationTokens is an immutable boot-time policy sharing the existing signer.
// Removing grants or changing TTL requires rebuilding it (normally a Core restart).
type ApplicationTokens struct {
	signer *Signer
	ttl    time.Duration
	grants map[uuid.UUID]ApplicationGrant
}

func NewApplicationTokens(signer *Signer, cfg ApplicationConfig) (*ApplicationTokens, error) {
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = DefaultApplicationTTL
	}
	if cfg.TokenTTL < time.Second || cfg.TokenTTL > 15*time.Minute {
		return nil, fmt.Errorf("application token TTL must be between 1s and 15m (omitted/zero defaults to 5m)")
	}
	if signer == nil && (len(cfg.Grants) != 0 || cfg.TokenTTL != DefaultApplicationTTL) {
		return nil, fmt.Errorf("application token policy requires CMP SSO signing settings")
	}
	p := &ApplicationTokens{signer: signer, ttl: cfg.TokenTTL, grants: make(map[uuid.UUID]ApplicationGrant)}
	for key, grant := range cfg.Grants {
		id, err := uuid.Parse(key)
		if err != nil || id == uuid.Nil || key != id.String() {
			return nil, fmt.Errorf("application grant key must be a canonical nonzero agent UUID")
		}
		if strings.TrimSpace(grant.Audience) == "" || len(grant.Roles) == 0 {
			return nil, fmt.Errorf("application grant for %s requires an audience and roles", id)
		}
		for _, role := range grant.Roles {
			if strings.TrimSpace(role) == "" {
				return nil, fmt.Errorf("application grant for %s contains an empty role", id)
			}
		}
		grant.Roles = append([]string(nil), grant.Roles...)
		p.grants[id] = grant
	}
	return p, nil
}

func (p *ApplicationTokens) Enabled() bool { return p != nil && p.signer != nil }

// Grant returns a copy; request data cannot create or modify the policy.
func (p *ApplicationTokens) Grant(id uuid.UUID) (ApplicationGrant, bool) {
	if p == nil {
		return ApplicationGrant{}, false
	}
	grant, ok := p.grants[id]
	grant.Roles = append([]string(nil), grant.Roles...)
	return grant, ok
}

// Issue accepts only the authenticated agent ID; every other claim is server-owned.
func (p *ApplicationTokens) Issue(id uuid.UUID) (string, time.Time, error) {
	if !p.Enabled() {
		return "", time.Time{}, fmt.Errorf("application signing is not configured")
	}
	grant, ok := p.Grant(id)
	if !ok {
		return "", time.Time{}, fmt.Errorf("agent has no application grant")
	}
	now := p.signer.now()
	expires := now.Add(p.ttl)
	claims := ApplicationClaims{
		Claims: jwt.Claims{Issuer: p.signer.issuer, Audience: jwt.Audience{grant.Audience}, IssuedAt: jwt.NewNumericDate(now), Expiry: jwt.NewNumericDate(expires)},
		App:    id.String(), Roles: grant.Roles,
	}
	token, err := jwt.Signed(p.signer.signer).Claims(claims).Serialize()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign application token: %w", err)
	}
	return token, expires, nil
}

// ApplicationClaims deliberately has no human, tenant, service or VDC identity.
// JWT wire fields retain the external application/module proposal's spelling.
type ApplicationClaims struct {
	jwt.Claims
	App   string   `json:"app"`
	Roles []string `json:"roles"`
}
