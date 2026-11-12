// Signer issues generic Core identity JWTs using one ES256/P-256 key.
// Public discovery and bounded opaque extensions are shared by both caller roles.
package jwtissuer

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
)

// maxExtBytes caps the serialized size of the caller-asserted ext block.
// ext is opaque passthrough; the cap stops a caller from bloating the token.
const maxExtBytes = 4096

// reservedClaims are claim names ext may never carry. ext is nested under its
// own "ext" key so it structurally cannot overwrite a top-level system claim,
// but rejecting these names too keeps the contract explicit and guards against
// a future refactor that flattens ext into the payload.
var reservedClaims = map[string]struct{}{
	"iss": {}, "sub": {}, "aud": {}, "exp": {}, "nbf": {}, "iat": {}, "jti": {},
	"role": {}, "participant_id": {}, "agent_id": {}, "service_id": {}, "relation": {}, "ext": {},
}

// Signer holds the shared ES256 signing primitive and public verification JWKS.
// It is built once at boot and is safe for concurrent use by both caller roles.
type Signer struct {
	signer         jose.Signer
	keys           jose.JSONWebKeySet
	issuer         string
	participantTTL time.Duration
	agentTTL       time.Duration
	// now is injectable so tests can pin iat/nbf/exp.
	now func() time.Time
}

// New builds a Signer from config: it parses the EC P-256 private key, wires an
// ES256 jose.Signer that stamps the kid into every protected header, and
// derives the matching public key for the JWKS.
func New(cfg Config) (*Signer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if !cfg.Enabled() {
		return nil, nil
	}
	if cfg.ParticipantTTL == 0 {
		cfg.ParticipantTTL = DefaultParticipantTTL
	}
	if cfg.AgentTTL == 0 {
		cfg.AgentTTL = DefaultAgentTTL
	}
	priv, err := parseECPrivateKey(cfg.SigningKey)
	if err != nil {
		return nil, fmt.Errorf("parse JWT signing key: %w", err)
	}

	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.ES256, Key: priv},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", cfg.KeyID),
	)
	if err != nil {
		return nil, fmt.Errorf("build jose signer: %w", err)
	}

	pub := jose.JSONWebKey{
		Key:       priv.Public(),
		KeyID:     cfg.KeyID,
		Algorithm: string(jose.ES256),
		Use:       "sig",
	}

	return &Signer{
		signer:         sig,
		keys:           jose.JSONWebKeySet{Keys: []jose.JSONWebKey{pub}},
		issuer:         cfg.Issuer,
		participantTTL: cfg.ParticipantTTL,
		agentTTL:       cfg.AgentTTL,
		now:            time.Now,
	}, nil
}

// Sign stamps registered claims and a domain subject using caller-role TTL policy.
// Its typed identity/context must come from Core authorization, not a request body.
func (s *Signer) Sign(claims Claims) (token string, expiresAt time.Time, err error) {
	if s == nil {
		return "", time.Time{}, fmt.Errorf("JWT signing is not configured")
	}
	if err := ValidateExt(claims.Ext); err != nil {
		return "", time.Time{}, err
	}

	ttl := s.participantTTL
	switch claims.Role {
	case "participant":
		claims.Subject = claims.ParticipantID
	case "agent":
		ttl = s.agentTTL
		claims.Subject = claims.AgentID
	default:
		return "", time.Time{}, fmt.Errorf("unsupported JWT principal role")
	}
	now := s.now().UTC().Truncate(time.Second)
	expiresAt = now.Add(ttl)

	claims.Issuer = s.issuer
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.NotBefore = jwt.NewNumericDate(now)
	claims.Expiry = jwt.NewNumericDate(expiresAt)
	claims.ID = uuid.NewString()

	token, err = jwt.Signed(s.signer).Claims(claims).Serialize()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign identity token: %w", err)
	}
	return token, expiresAt, nil
}

// parseECPrivateKey decodes a PEM-encoded EC P-256 private key, accepting both
// SEC1 ("EC PRIVATE KEY") and PKCS#8 ("PRIVATE KEY") encodings.
func parseECPrivateKey(pemKey string) (*ecdsa.PrivateKey, error) {
	block, rest := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	if strings.TrimSpace(string(rest)) != "" {
		return nil, fmt.Errorf("expected a single private key PEM block")
	}
	var ecKey *ecdsa.PrivateKey
	var err error
	switch block.Type {
	case "EC PRIVATE KEY":
		ecKey, err = x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		var ok bool
		ecKey, ok = key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS#8 key is %T, want *ecdsa.PrivateKey", key)
		}
	default:
		return nil, fmt.Errorf("unsupported PEM block type %q", block.Type)
	}
	if err != nil {
		return nil, err
	}
	if ecKey.Curve != elliptic.P256() || ecKey.D == nil || ecKey.D.Sign() <= 0 || ecKey.D.Cmp(elliptic.P256().Params().N) >= 0 || !elliptic.P256().IsOnCurve(ecKey.X, ecKey.Y) {
		return nil, fmt.Errorf("signing key must be a usable EC P-256 private key for ES256")
	}
	x, y := elliptic.P256().ScalarBaseMult(ecKey.D.Bytes())
	if x.Cmp(ecKey.X) != 0 || y.Cmp(ecKey.Y) != 0 {
		return nil, fmt.Errorf("signing key public and private material do not match")
	}
	return ecKey, nil
}

// JWKS exposes only the single current public verification key.
// Key replacement at restart invalidates tokens signed with the previous key.
func (s *Signer) JWKS() jose.JSONWebKeySet {
	if s == nil {
		return jose.JSONWebKeySet{Keys: []jose.JSONWebKey{}}
	}
	// Return a fresh public key so callers cannot mutate the signer's key set.
	pub := s.keys.Keys[0]
	key := pub.Key.(*ecdsa.PublicKey)
	pub.Key = &ecdsa.PublicKey{Curve: key.Curve, X: new(big.Int).Set(key.X), Y: new(big.Int).Set(key.Y)}
	return jose.JSONWebKeySet{Keys: []jose.JSONWebKey{pub}}
}

// Claims certifies Core identity and optional authorized service context.
// Ext is separately namespaced, caller-asserted data, never external privileges.
type Claims struct {
	jwt.Claims
	Role          string         `json:"role"`
	ParticipantID string         `json:"participant_id"`
	AgentID       string         `json:"agent_id,omitempty"`
	ServiceID     string         `json:"service_id,omitempty"`
	Relation      string         `json:"relation,omitempty"`
	Ext           map[string]any `json:"ext,omitempty"`
}

// ValidateExt enforces the size cap and rejects reserved claim names inside ext.
// Exported so the handler can reject bad ext as client input (400) before
// signing; Sign also calls it as a defense-in-depth guard.
func ValidateExt(ext map[string]any) error {
	if len(ext) == 0 {
		return nil
	}
	for k := range ext {
		if _, reserved := reservedClaims[k]; reserved {
			return fmt.Errorf("ext may not contain reserved claim %q", k)
		}
	}
	encoded, err := json.Marshal(ext)
	if err != nil {
		return fmt.Errorf("ext is not serializable: %w", err)
	}
	if len(encoded) > maxExtBytes {
		return fmt.Errorf("ext exceeds %d bytes", maxExtBytes)
	}
	return nil
}
