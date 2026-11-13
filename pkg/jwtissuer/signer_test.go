// Tests JWT signing, public verification and extension constraints.
// All signing material is generated locally for each test.
package jwtissuer

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig generates a throwaway P-256 key and returns a usable JWT config.
func testConfig(t *testing.T) Config {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	return Config{
		SigningKey:     string(pemKey),
		KeyID:          "core-jwt-test",
		Issuer:         "https://fulcrum.test",
		ParticipantTTL: 90 * time.Second,
	}
}

func TestSign_RoundTrip(t *testing.T) {
	cfg := testConfig(t)
	signer, err := New(cfg)
	require.NoError(t, err)

	frozen := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	signer.now = func() time.Time { return frozen }

	token, expiresAt, err := signer.Sign(Claims{Role: "participant",
		Claims:        jwt.Claims{Subject: "p-123", Audience: jwt.Audience{"recipient"}},
		ParticipantID: "p-123",
		ServiceID:     "svc-456",
		Relation:      "provider",
	})
	require.NoError(t, err)
	assert.Equal(t, frozen.Add(90*time.Second), expiresAt)

	// Verify against the public JWKS, as a recipient would.
	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.ES256})
	require.NoError(t, err)

	var out Claims
	require.NoError(t, parsed.Claims(signer.JWKS(), &out))

	assert.Equal(t, "https://fulcrum.test", out.Issuer)
	assert.Equal(t, "p-123", out.Subject)
	assert.Equal(t, jwt.Audience{"recipient"}, out.Audience)
	assert.Equal(t, "p-123", out.ParticipantID)
	assert.Equal(t, "svc-456", out.ServiceID)
	assert.Equal(t, "provider", out.Relation)
	assert.Equal(t, frozen.Unix(), out.IssuedAt.Time().Unix())
	assert.Equal(t, frozen.Unix(), out.NotBefore.Time().Unix())
	assert.Equal(t, expiresAt.Unix(), out.Expiry.Time().Unix())
	assert.NotEmpty(t, out.ID, "jti should be stamped")
}

func TestSign_KidInHeader(t *testing.T) {
	signer, err := New(testConfig(t))
	require.NoError(t, err)

	token, _, err := signer.Sign(Claims{Role: "participant", ParticipantID: "p-1"})
	require.NoError(t, err)

	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.ES256})
	require.NoError(t, err)
	require.Len(t, parsed.Headers, 1)
	assert.Equal(t, "core-jwt-test", parsed.Headers[0].KeyID)
	assert.Equal(t, string(jose.ES256), parsed.Headers[0].Algorithm)
}

func TestSign_FreshJtiEachCall(t *testing.T) {
	signer, err := New(testConfig(t))
	require.NoError(t, err)

	t1, _, err := signer.Sign(Claims{Role: "participant", ParticipantID: "p-1"})
	require.NoError(t, err)
	t2, _, err := signer.Sign(Claims{Role: "participant", ParticipantID: "p-1"})
	require.NoError(t, err)
	assert.NotEqual(t, t1, t2, "each token gets a unique jti")
}

func TestSign_RejectsReservedExtKey(t *testing.T) {
	signer, err := New(testConfig(t))
	require.NoError(t, err)

	_, _, err = signer.Sign(Claims{Role: "participant",
		ParticipantID: "p-1",
		Ext:           map[string]any{"aud": "evil-recipient"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reserved claim")
}

func TestSign_RejectsOversizedExt(t *testing.T) {
	signer, err := New(testConfig(t))
	require.NoError(t, err)

	_, _, err = signer.Sign(Claims{Role: "participant",
		ParticipantID: "p-1",
		Ext:           map[string]any{"blob": strings.Repeat("x", maxExtBytes+1)},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestSign_ExtPassedThrough(t *testing.T) {
	signer, err := New(testConfig(t))
	require.NoError(t, err)

	token, _, err := signer.Sign(Claims{Role: "participant",
		ParticipantID: "p-1",
		Relation:      "consumer",
		Ext:           map[string]any{"operatorEmail": "buyer@acme.com", "mpOrderRef": "ORD-42"},
	})
	require.NoError(t, err)

	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.ES256})
	require.NoError(t, err)
	var out Claims
	require.NoError(t, parsed.Claims(signer.JWKS(), &out))
	assert.Equal(t, "buyer@acme.com", out.Ext["operatorEmail"])
	assert.Equal(t, "ORD-42", out.Ext["mpOrderRef"])
}

func TestNew_RejectsBadKey(t *testing.T) {
	_, err := New(Config{
		SigningKey:     "not a pem key",
		KeyID:          "k1",
		Issuer:         "iss",
		ParticipantTTL: time.Minute,
	})
	require.Error(t, err)
}

func TestParseECPrivateKey_PKCS8(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	got, err := parseECPrivateKey(string(pemKey))
	require.NoError(t, err)
	assert.True(t, key.Equal(got))
}

func TestConfiguration(t *testing.T) {
	for name, cfg := range map[string]Config{
		"empty": {}, "default TTL": {ParticipantTTL: DefaultParticipantTTL},
	} {
		t.Run(name, func(t *testing.T) {
			s, err := New(cfg)
			require.NoError(t, err)
			require.Nil(t, s)
			require.Empty(t, s.JWKS().Keys)
			_, _, err = s.Sign(Claims{Role: "participant"})
			require.Error(t, err)
		})
	}
	for name, cfg := range map[string]Config{
		"key only":       {SigningKey: "invalid"},
		"kid only":       {KeyID: "k1"},
		"issuer only":    {Issuer: "issuer"},
		"TTL only":       {ParticipantTTL: time.Minute},
		"negative TTL":   {ParticipantTTL: -time.Second},
		"blank settings": {SigningKey: " ", KeyID: " ", Issuer: " "},
	} {
		t.Run(name, func(t *testing.T) { _, err := New(cfg); require.Error(t, err) })
	}
	cfg := testConfig(t)
	cfg.ParticipantTTL = 0
	s, err := New(cfg)
	require.NoError(t, err)
	require.Equal(t, DefaultParticipantTTL, s.participantTTL)
	cfg.ParticipantTTL = -time.Second
	_, err = New(cfg)
	require.Error(t, err)
}

func TestKeyValidation(t *testing.T) {
	for _, curve := range []elliptic.Curve{elliptic.P384(), elliptic.P521()} {
		key, err := ecdsa.GenerateKey(curve, rand.Reader)
		require.NoError(t, err)
		for _, pkcs8 := range []bool{false, true} {
			var der []byte
			typ := "EC PRIVATE KEY"
			if pkcs8 {
				der, err = x509.MarshalPKCS8PrivateKey(key)
				typ = "PRIVATE KEY"
			} else {
				der, err = x509.MarshalECPrivateKey(key)
			}
			require.NoError(t, err)
			cfg := testConfig(t)
			cfg.SigningKey = string(pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}))
			_, err = New(cfg)
			require.ErrorContains(t, err, "P-256")
		}
	}
	cfg := testConfig(t)
	for _, bad := range []string{"not PEM", "-----BEGIN PRIVATE KEY-----\nAQID\n-----END PRIVATE KEY-----", cfg.SigningKey + cfg.SigningKey} {
		cfg.SigningKey = bad
		_, err := New(cfg)
		require.Error(t, err)
	}
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err)
	cfg.SigningKey = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	_, err = New(cfg)
	require.Error(t, err)
}

func TestPublicKeysAndSystemClaims(t *testing.T) {
	s, err := New(testConfig(t))
	require.NoError(t, err)
	keyJSON, err := json.Marshal(s.JWKS())
	require.NoError(t, err)
	var keys struct {
		Keys []map[string]any `json:"keys"`
	}
	require.NoError(t, json.Unmarshal(keyJSON, &keys))
	require.Len(t, keys.Keys, 1)
	require.Len(t, keys.Keys[0], 7)
	for _, field := range []string{"d", "p", "q", "k"} {
		require.NotContains(t, keys.Keys[0], field)
	}
	require.Equal(t, "EC", keys.Keys[0]["kty"])
	require.Equal(t, "P-256", keys.Keys[0]["crv"])
	require.Equal(t, "ES256", keys.Keys[0]["alg"])
	frozen := time.Now().Truncate(time.Second)
	s.now = func() time.Time { return frozen }
	token, _, err := s.Sign(Claims{Role: "participant", Claims: jwt.Claims{Issuer: "evil", ID: "evil", Expiry: jwt.NewNumericDate(frozen.Add(time.Hour))}})
	require.NoError(t, err)
	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.ES256})
	require.NoError(t, err)
	var claims Claims
	require.NoError(t, parsed.Claims(s.JWKS(), &claims))
	require.NotEqual(t, "evil", claims.Issuer)
	require.NotEqual(t, "evil", claims.ID)
	require.Equal(t, frozen.Add(DefaultParticipantTTL), claims.Expiry.Time())
	require.NoError(t, claims.Validate(jwt.Expected{Issuer: claims.Issuer, Time: frozen}))
	require.Error(t, claims.ValidateWithLeeway(jwt.Expected{Time: frozen.Add(DefaultParticipantTTL + time.Second)}, 0))
	_, err = jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.HS256})
	require.Error(t, err)
	// Returned keys cannot mutate subsequent discovery responses.
	keysCopy := s.JWKS()
	keysCopy.Keys[0].Key.(*ecdsa.PublicKey).X.SetInt64(0)
	require.NoError(t, parsed.Claims(s.JWKS(), &claims))
}

func TestExtBoundaries(t *testing.T) {
	for key := range reservedClaims {
		require.Error(t, ValidateExt(map[string]any{key: "x"}))
	}
	require.NoError(t, ValidateExt(map[string]any{"x": strings.Repeat("x", maxExtBytes-8)}))
	require.Error(t, ValidateExt(map[string]any{"x": strings.Repeat("x", maxExtBytes-7)}))
	require.Error(t, ValidateExt(map[string]any{"x": make(chan int)}))
	require.NoError(t, ValidateExt(map[string]any{"nested": map[string]any{"iss": "opaque"}}))
}

func TestConcurrentSigning(t *testing.T) {
	s, err := New(testConfig(t))
	require.NoError(t, err)
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, _, err := s.Sign(Claims{Role: "participant"})
			if !assert.NoError(t, err) {
				return
			}
			parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.ES256})
			if assert.NoError(t, err) {
				assert.NoError(t, parsed.Claims(s.JWKS(), &Claims{Role: "participant"}))
			}
		}()
	}
	wg.Wait()
}

func TestRolePolicyAndClaimPresence(t *testing.T) {
	for _, custom := range []bool{false, true} {
		cfg := testConfig(t)
		if custom {
			cfg.ParticipantTTL = 45 * time.Second
			cfg.AgentTTL = 2 * time.Minute
		}
		s, err := New(cfg)
		require.NoError(t, err)
		for _, role := range []string{"participant", "agent"} {
			expectedTTL := DefaultParticipantTTL
			if role == "agent" {
				expectedTTL = DefaultAgentTTL
			}
			if custom {
				expectedTTL = cfg.ParticipantTTL
				if role == "agent" {
					expectedTTL = cfg.AgentTTL
				}
			}
			claims := Claims{Claims: jwt.Claims{Subject: "credential-id-is-not-the-subject", Audience: jwt.Audience{"recipient"}}, Role: role, ParticipantID: "provider-id"}
			subject := "provider-id"
			if role == "agent" {
				claims.AgentID = "agent-id"
				subject = "agent-id"
			}
			token, expires, err := s.Sign(claims)
			require.NoError(t, err)
			parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.ES256})
			require.NoError(t, err)
			var raw map[string]any
			require.NoError(t, parsed.Claims(s.JWKS(), &raw))
			require.Equal(t, subject, raw["sub"])
			require.Equal(t, role, raw["role"])
			require.Equal(t, "provider-id", raw["participant_id"])
			require.Equal(t, expectedTTL.Seconds(), raw["exp"].(float64)-raw["iat"].(float64))
			require.Equal(t, raw["iat"], raw["nbf"])
			require.Equal(t, float64(expires.Unix()), raw["exp"])
			for _, key := range []string{"service_id", "relation", "ext", "app", "roles", "redirectUrl"} {
				require.NotContains(t, raw, key)
			}
			if role == "agent" {
				require.Equal(t, "agent-id", raw["agent_id"])
			} else {
				require.NotContains(t, raw, "agent_id")
			}
		}
	}
	s, err := New(testConfig(t))
	require.NoError(t, err)
	for _, role := range []string{"admin", "other", ""} {
		_, _, err = s.Sign(Claims{Role: role})
		require.Error(t, err)
	}
	for _, ttl := range []time.Duration{-time.Second, time.Millisecond, time.Second + time.Nanosecond} {
		for _, agent := range []bool{false, true} {
			cfg := testConfig(t)
			if agent {
				cfg.AgentTTL = ttl
			} else {
				cfg.ParticipantTTL = ttl
			}
			_, err := New(cfg)
			require.Error(t, err)
		}
	}
}
