// Tests the narrow issuance request surface and its static token route.
// Generic requests cannot select protected claims or interfere with token CRUD.
package api

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/jwtissuer"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func issueTestSigner(t *testing.T) *jwtissuer.Signer {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	signer, err := jwtissuer.New(jwtissuer.Config{SigningKey: string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})), KeyID: "test-key", Issuer: "https://core.example"})
	require.NoError(t, err)
	return signer
}

func TestIssueTokenRequest(t *testing.T) {
	for _, body := range []string{
		``, `null`, `{`, `[]`, `{}`, `{"audience":" "}`, `{"audience":3}`, `{"audience":"recipient"} {}`, `{"audience":"recipient"} junk`,
		`{"audience":"recipient","serviceId":"not-uuid"}`, `{"audience":"recipient","serviceId":"00000000-0000-0000-0000-000000000000"}`,
		`{"audience":"recipient","ext":[]}`, `{"audience":"recipient","ext":"bad"}`, `{"audience":"recipient","ext":{"blob":"` + strings.Repeat("x", 4096) + `"}}`,
	} {
		t.Run(body[:min(len(body), 70)], func(t *testing.T) {
			w := httptest.NewRecorder()
			NewTokenIssueHandler(nil, nil).Issue(w, httptest.NewRequest(http.MethodPost, "/tokens/issue", strings.NewReader(body)))
			require.Equal(t, 400, w.Code, w.Body.String())
			require.NotContains(t, w.Body.String(), `"token":`)
		})
	}
	for _, field := range []string{"iss", "sub", "aud", "iat", "nbf", "exp", "jti", "role", "participant_id", "agent_id", "service_id", "relation", "ext"} {
		w := httptest.NewRecorder()
		body := `{"audience":"recipient","ext":{"` + field + `":"forged"}}`
		NewTokenIssueHandler(nil, nil).Issue(w, httptest.NewRequest(http.MethodPost, "/tokens/issue", strings.NewReader(body)))
		require.Equal(t, 400, w.Code, w.Body.String())
	}
	for _, field := range []string{"role", "participantId", "agentId", "relation", "iss", "sub", "aud", "exp", "jti", "signingKey", "keyId", "issuer", "tokenTtl", "redirectUrl", "app", "roles"} {
		w := httptest.NewRecorder()
		body := `{"audience":"recipient","` + field + `":"forged"}`
		NewTokenIssueHandler(nil, nil).Issue(w, httptest.NewRequest(http.MethodPost, "/tokens/issue", strings.NewReader(body)))
		require.Equal(t, 400, w.Code, w.Body.String())
	}
}

func TestTokenIssueRoutes(t *testing.T) {
	pid, credentialID, tokenID := properties.NewUUID(), properties.NewUUID(), properties.NewUUID()
	identity := &auth.Identity{ID: credentialID, Role: auth.RoleParticipant, Scope: auth.IdentityScope{ParticipantID: &pid}}
	store, pq, events, tq := domain.NewMockStore(t), domain.NewMockParticipantRepository(t), domain.NewMockEventRepository(t), domain.NewMockTokenQuerier(t)
	store.EXPECT().ParticipantRepo().Return(pq).Once()
	pq.EXPECT().Get(mock.Anything, pid).Return(&domain.Participant{BaseEntity: domain.BaseEntity{ID: pid}}, nil).Once()
	store.EXPECT().EventRepo().Return(events).Once()
	events.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	tq.EXPECT().AuthScope(mock.Anything, tokenID).Return(&authz.DefaultObjectScope{ParticipantID: &pid}, nil).Once()
	tq.EXPECT().Get(mock.Anything, tokenID).Return(&domain.Token{BaseEntity: domain.BaseEntity{ID: tokenID}}, nil).Once()
	signer := issueTestSigner(t)
	authorizer := authz.NewRuleBasedAuthorizer(authz.Rules)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
		})
	})
	r.Route("/tokens", func(r chi.Router) {
		NewTokenHandler(tq, domain.NewMockTokenCommander(t), nil, authorizer).Routes()(r)
		NewTokenIssueHandler(domain.NewTokenIssueCommander(store, authorizer), signer).Routes()(r)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tokens/issue?sub=forged&exp=9999999999", strings.NewReader(`{"audience":"recipient","ext":{"order":"42","nested":{"role":"opaque"}}}`)))
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	var raw map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	require.Len(t, raw, 2)
	var result IssueTokenRes
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	parsed, err := jwt.ParseSigned(result.Token, []jose.SignatureAlgorithm{jose.ES256})
	require.NoError(t, err)
	var claims jwtissuer.Claims
	require.NoError(t, parsed.Claims(signer.JWKS(), &claims))
	require.NoError(t, claims.Validate(jwt.Expected{Issuer: "https://core.example", Subject: pid.String(), AnyAudience: jwt.Audience{"recipient"}, Time: time.Now()}))
	require.NotEqual(t, credentialID.String(), claims.Subject)
	require.Equal(t, "participant", claims.Role)
	require.Empty(t, claims.AgentID)
	require.Empty(t, claims.ServiceID)
	require.Empty(t, claims.Relation)
	require.Equal(t, "42", claims.Ext["order"])
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tokens/"+tokenID.String(), nil))
	require.Equal(t, 200, w.Code, w.Body.String())
}

func TestIssueTokenExtNumbers(t *testing.T) {
	for _, number := range []string{
		"9007199254740993", "-9007199254740993",
		"1234567890123456789", "-1234567890123456789",
		"0.12345678901234567890123456789",
		"-1.234567890123456789e+25", "1.234567890123456789e-25",
	} {
		t.Run(number, func(t *testing.T) {
			pid := properties.NewUUID()
			identity := &auth.Identity{ID: properties.NewUUID(), Role: auth.RoleParticipant, Scope: auth.IdentityScope{ParticipantID: &pid}}
			store, pq, events := domain.NewMockStore(t), domain.NewMockParticipantRepository(t), domain.NewMockEventRepository(t)
			store.EXPECT().ParticipantRepo().Return(pq).Once()
			pq.EXPECT().Get(mock.Anything, pid).Return(&domain.Participant{BaseEntity: domain.BaseEntity{ID: pid}}, nil).Once()
			store.EXPECT().EventRepo().Return(events).Once()
			events.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
			signer := issueTestSigner(t)
			body := `{"audience":"recipient","ext":{"number":` + number + `,"nested":[{"numbers":[` + number + `,` + number + `]}]}}`
			r := httptest.NewRequest(http.MethodPost, "/tokens/issue", strings.NewReader(body))
			r = r.WithContext(auth.WithIdentity(r.Context(), identity))
			w := httptest.NewRecorder()
			NewTokenIssueHandler(domain.NewTokenIssueCommander(store, nil), signer).Issue(w, r)
			require.Equal(t, 200, w.Code, w.Body.String())
			var result IssueTokenRes
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
			parsed, err := jose.ParseSigned(result.Token, []jose.SignatureAlgorithm{jose.ES256})
			require.NoError(t, err)
			payload, err := parsed.Verify(signer.JWKS())
			require.NoError(t, err)
			// Decode verified claims into json.Number, never float64, including
			// numbers inside nested objects and arrays in the actual JWT payload.
			var claims struct {
				Ext struct {
					Number any `json:"number"`
					Nested []struct {
						Numbers []any `json:"numbers"`
					} `json:"nested"`
				} `json:"ext"`
			}
			dec := json.NewDecoder(bytes.NewReader(payload))
			dec.UseNumber()
			require.NoError(t, dec.Decode(&claims))
			require.Len(t, claims.Ext.Nested, 1)
			require.Len(t, claims.Ext.Nested[0].Numbers, 2)
			want, ok := new(big.Rat).SetString(number)
			require.True(t, ok)
			for _, value := range append([]any{claims.Ext.Number}, claims.Ext.Nested[0].Numbers...) {
				got, ok := value.(json.Number)
				require.True(t, ok, "expected a JSON number, got %T", value)
				actual, ok := new(big.Rat).SetString(got.String())
				require.True(t, ok)
				require.Zero(t, actual.Cmp(want), "opaque number changed: want %s, got %s", number, got)
			}
		})
	}
}

func TestDisabledJWKS(t *testing.T) {
	w := httptest.NewRecorder()
	NewJwksHandler(nil).JWKS(w, httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil))
	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `{"keys":[]}`, w.Body.String())
}
