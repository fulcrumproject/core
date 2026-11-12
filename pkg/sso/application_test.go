// Tests operator-owned application grants and the separate JWT profile.
// Synthetic keys and pinned time keep these checks independent of infrastructure.
package sso

import (
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestApplicationTokens(t *testing.T) {
	s, err := New(testConfig(t))
	require.NoError(t, err)
	now := time.Now().Truncate(time.Second)
	s.now = func() time.Time { return now }
	id := uuid.New()
	grant := ApplicationGrant{Audience: "fae-demo", Roles: []string{"pve"}}
	cfg := ApplicationConfig{Grants: map[string]ApplicationGrant{id.String(): grant}}
	app, err := NewApplicationTokens(s, cfg)
	require.NoError(t, err)
	// Mutating caller-owned configuration cannot change the boot-time policy.
	cfg.Grants[id.String()] = ApplicationGrant{Audience: "evil", Roles: []string{"admin"}}
	grant.Roles[0] = "evil"
	got, ok := app.Grant(id)
	require.True(t, ok)
	require.Equal(t, "fae-demo", got.Audience)
	require.Equal(t, []string{"pve"}, got.Roles)
	got.Roles[0] = "evil"
	token, expires, err := app.Issue(id)
	require.NoError(t, err)
	require.Equal(t, now.Add(5*time.Minute), expires)
	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.ES256})
	require.NoError(t, err)
	require.Equal(t, s.JWKS().Keys[0].KeyID, parsed.Headers[0].KeyID)
	var claims map[string]any
	require.NoError(t, parsed.Claims(s.JWKS(), &claims))
	require.Len(t, claims, 6)
	require.Equal(t, id.String(), claims["app"])
	require.Equal(t, []any{"pve"}, claims["roles"])
	require.Equal(t, "fae-demo", claims["aud"])
	require.Equal(t, s.issuer, claims["iss"])
	require.Equal(t, float64(now.Unix()), claims["iat"])
	require.Equal(t, float64(expires.Unix()), claims["exp"])
	_, _, err = app.Issue(uuid.New())
	require.Error(t, err)
	launch, _, err := s.Sign(LaunchClaims{})
	require.NoError(t, err)
	parsed, err = jwt.ParseSigned(launch, []jose.SignatureAlgorithm{jose.ES256})
	require.NoError(t, err)
	var browser LaunchClaims
	require.NoError(t, parsed.Claims(s.JWKS(), &browser))
	require.Equal(t, int64(90), int64(*browser.Expiry-*browser.IssuedAt))
	require.NotContains(t, claims, "service_id")
}

func TestApplicationConfiguration(t *testing.T) {
	s, err := New(testConfig(t))
	require.NoError(t, err)
	id := uuid.New().String()
	good := map[string]ApplicationGrant{id: {Audience: "fae", Roles: []string{"pve"}}}
	for name, cfg := range map[string]ApplicationConfig{
		"bad UUID":         {Grants: map[string]ApplicationGrant{"bad": {Audience: "fae", Roles: []string{"pve"}}}},
		"nil UUID":         {Grants: map[string]ApplicationGrant{uuid.Nil.String(): {Audience: "fae", Roles: []string{"pve"}}}},
		"missing audience": {Grants: map[string]ApplicationGrant{id: {Roles: []string{"pve"}}}},
		"missing roles":    {Grants: map[string]ApplicationGrant{id: {Audience: "fae"}}},
		"blank role":       {Grants: map[string]ApplicationGrant{id: {Audience: "fae", Roles: []string{" "}}}},
		"blank audience":   {Grants: map[string]ApplicationGrant{id: {Audience: " ", Roles: []string{"pve"}}}},
		"negative TTL":     {TokenTTL: -time.Second},
		"subsecond TTL":    {TokenTTL: time.Millisecond},
		"excessive TTL":    {TokenTTL: time.Hour},
	} {
		t.Run(name, func(t *testing.T) { _, err := NewApplicationTokens(s, cfg); require.Error(t, err) })
	}
	for _, cfg := range []ApplicationConfig{{}, {TokenTTL: DefaultApplicationTTL}} {
		app, err := NewApplicationTokens(nil, cfg)
		require.NoError(t, err)
		require.False(t, app.Enabled())
		_, _, err = app.Issue(uuid.New())
		require.Error(t, err)
	}
	for _, cfg := range []ApplicationConfig{{Grants: good}, {TokenTTL: time.Minute}} {
		_, err := NewApplicationTokens(nil, cfg)
		require.Error(t, err)
	}
	app, err := NewApplicationTokens(s, ApplicationConfig{TokenTTL: time.Minute, Grants: good})
	require.NoError(t, err)
	_, expires, err := app.Issue(uuid.MustParse(id))
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(time.Minute), expires, time.Second)
}
