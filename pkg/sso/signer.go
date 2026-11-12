// Signer shares one ES256/P-256 key across CMP and application token profiles.
// It publishes only public verification material and bounds browser extensions.
package sso

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

// maxExtBytes caps the serialized size of the marketplace-asserted ext block.
// ext is opaque passthrough; the cap stops a caller from bloating the token.
const maxExtBytes = 4096

// reservedClaims are claim names ext may never carry. ext is nested under its
// own "ext" key so it structurally cannot overwrite a top-level system claim,
// but rejecting these names too keeps the contract explicit and guards against
// a future refactor that flattens ext into the payload.
var reservedClaims = map[string]struct{}{
	"iss": {}, "sub": {}, "aud": {}, "exp": {}, "nbf": {}, "iat": {}, "jti": {},
	"participant_id": {}, "service_id": {}, "relation": {}, "ext": {},
}

// Signer holds the shared ES256 signing primitive and public verification JWKS.
// It is built once at boot and is safe for concurrent use by both profiles.
type Signer struct {
	signer   jose.Signer
	keys     jose.JSONWebKeySet
	issuer   string
	tokenTTL time.Duration
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
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = DefaultTokenTTL
	}
	priv, err := parseECPrivateKey(cfg.SigningKey)
	if err != nil {
		return nil, fmt.Errorf("parse CMP SSO signing key: %w", err)
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
		signer:   sig,
		keys:     jose.JSONWebKeySet{Keys: []jose.JSONWebKey{pub}},
		issuer:   cfg.Issuer,
		tokenTTL: cfg.TokenTTL,
		now:      time.Now,
	}, nil
}

// Sign stamps the system claims (iss from config; iat/nbf = now; exp = now+TTL;
// a fresh uuid jti) onto the caller's claims and serializes the signed token.
// It returns the token and its absolute expiry. ext is validated before signing.
func (s *Signer) Sign(claims LaunchClaims) (token string, expiresAt time.Time, err error) {
	if s == nil {
		return "", time.Time{}, fmt.Errorf("CMP SSO signing is not configured")
	}
	if err := ValidateExt(claims.Ext); err != nil {
		return "", time.Time{}, err
	}

	now := s.now()
	expiresAt = now.Add(s.tokenTTL)

	claims.Issuer = s.issuer
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.NotBefore = jwt.NewNumericDate(now)
	claims.Expiry = jwt.NewNumericDate(expiresAt)
	claims.ID = uuid.NewString()

	token, err = jwt.Signed(s.signer).Claims(claims).Serialize()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign launch token: %w", err)
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

// JWKS returns the public key set a CMP fetches (once, cached by kid) to verify
// launch tokens. Public keys only — the private signing key never leaves Signer.
// It publishes the single key from the current config; overlap rotation (serving
// a previous public key alongside the new one) is not yet implemented, so a key
// swap invalidates in-flight tokens at restart (bounded by the short TokenTTL).
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

// LaunchClaims is the payload of a CMP launch token: the RFC 7519 registered
// claims (embedded, flattened on marshal) plus Fulcrum's private claims. Sign
// stamps iss/iat/nbf/exp/jti; the caller supplies the rest. ext is opaque,
// marketplace-asserted data that Fulcrum signs but never interprets.
type LaunchClaims struct {
	jwt.Claims

	ParticipantID string         `json:"participant_id"`
	ServiceID     string         `json:"service_id"`
	Relation      string         `json:"relation"`
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
