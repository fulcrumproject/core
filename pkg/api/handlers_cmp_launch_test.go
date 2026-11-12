// Tests CMP launch authorization, trusted claims and failure responses.
// Requests exercise the real service route, decoding and authorization chain.
package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/middlewares"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/fulcrumproject/core/pkg/sso"
	"github.com/go-chi/chi/v5"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func cmpTestSigner(t *testing.T) *sso.Signer {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	signer, err := sso.New(sso.Config{SigningKey: string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})), KeyID: "test-key", Issuer: "https://core.example"})
	require.NoError(t, err)
	return signer
}

type cmpAuthenticator struct{ identity *auth.Identity }

func (a cmpAuthenticator) Authenticate(context.Context, string) (*auth.Identity, error) {
	return a.identity, nil
}
func (a cmpAuthenticator) Health(context.Context) error { return nil }

func TestCmpLaunchRoutes(t *testing.T) {
	provider, consumer, stranger := properties.NewUUID(), properties.NewUUID(), properties.NewUUID()
	svc := &domain.Service{BaseEntity: domain.BaseEntity{ID: properties.NewUUID()}, ProviderID: provider, ConsumerID: consumer, AgentID: properties.NewUUID()}
	participant := func(id *properties.UUID) *auth.Identity {
		return &auth.Identity{ID: properties.NewUUID(), Role: auth.RoleParticipant, Scope: auth.IdentityScope{ParticipantID: id}}
	}
	for _, tc := range []struct {
		name                                       string
		identity                                   *auth.Identity
		body                                       string
		status                                     int
		relation                                   string
		disabled, noTarget, auditError, staleScope bool
	}{
		{name: "provider", identity: participant(&provider), body: `{}`, status: 200, relation: "provider"},
		{name: "consumer opaque ext and untrusted top-level fields", identity: participant(&consumer), body: `{"ext":{"order":"42","nested":{"iss":"opaque"}},"aud":"evil","service_id":"evil","relation":"provider","participantId":"evil","redirectUrl":"https://evil.example"}`, status: 200, relation: "consumer"},
		{name: "other participant", identity: participant(&stranger), body: `{}`, status: 403},
		{name: "agent with provider scope", identity: &auth.Identity{Role: auth.RoleAgent, Scope: auth.IdentityScope{ParticipantID: &provider, AgentID: &svc.AgentID}}, body: `{}`, status: 403},
		{name: "admin", identity: &auth.Identity{Role: auth.RoleAdmin}, body: `{}`, status: 403},
		{name: "missing identity", body: `{}`, status: 401},
		{name: "participant without scope", identity: participant(nil), body: `{}`, status: 403},
		{name: "reserved ext", identity: participant(&provider), body: `{"ext":{"aud":"evil"}}`, status: 400},
		{name: "oversized ext", identity: participant(&provider), body: `{"ext":{"blob":"` + strings.Repeat("x", 4096) + `"}}`, status: 400},
		{name: "wrong ext type", identity: participant(&provider), body: `{"ext":"evil"}`, status: 400},
		{name: "empty body matches CEM", identity: participant(&provider), body: ``, status: 400},
		{name: "malformed JSON", identity: participant(&provider), body: `{`, status: 400},
		{name: "null body matches CEM", identity: participant(&provider), body: `null`, status: 200, relation: "provider"},
		{name: "disabled signer", identity: participant(&provider), body: `{}`, status: 503, disabled: true},
		{name: "no CMP target", identity: participant(&provider), body: `{}`, status: 422, noTarget: true},
		{name: "audit must succeed", identity: participant(&provider), body: `{}`, status: 500, auditError: true},
		{name: "ownership rechecked after middleware", identity: participant(&stranger), body: `{}`, status: 403, staleScope: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sq, aq := domain.NewMockServiceQuerier(t), domain.NewMockAgentQuerier(t)
			scope := &authz.DefaultObjectScope{ProviderID: &provider, ConsumerID: &consumer, AgentID: &svc.AgentID}
			if tc.staleScope {
				scope.ProviderID = &stranger
			}
			sq.EXPECT().AuthScope(mock.Anything, svc.ID).Return(scope, nil).Maybe()
			sq.EXPECT().Get(mock.Anything, svc.ID).Return(svc, nil).Maybe()
			store, events := domain.NewMockStore(t), domain.NewMockEventRepository(t)
			config := properties.JSON{"cmpLandingUrl": "https://cmp.example/landing", "cmpAudience": "cmp"}
			if tc.noTarget {
				config = properties.JSON{}
			}
			if tc.status == 200 || tc.noTarget || tc.auditError {
				aq.EXPECT().Get(mock.Anything, svc.AgentID).Return(&domain.Agent{Configuration: &config}, nil).Once()
			}
			if tc.status == 200 || tc.auditError {
				store.EXPECT().EventRepo().Return(events).Once()
				var err error
				if tc.auditError {
					err = errors.New("audit unavailable")
				}
				events.EXPECT().Create(mock.Anything, mock.Anything).Run(func(ctx context.Context, e *domain.Event) {
					require.Equal(t, domain.EventTypeServiceCmpLaunched, e.Type)
					require.Equal(t, svc.ID, *e.EntityID)
					require.Equal(t, provider, *e.ProviderID)
					require.Equal(t, consumer, *e.ConsumerID)
					require.Equal(t, svc.AgentID, *e.AgentID)
					require.Equal(t, tc.identity.ID.String(), e.InitiatorID)
					require.Len(t, e.Payload, 2)
					require.Equal(t, "cmp", e.Payload["audience"])
				}).Return(err).Once()
			}
			signer := cmpTestSigner(t)
			if tc.disabled {
				signer = nil
			}
			authorizer := authz.NewRuleBasedAuthorizer(authz.Rules)
			h := NewCmpLaunchHandler(sq, aq, domain.NewCmpLaunchCommander(store), authorizer, signer)
			r := chi.NewRouter()
			r.Get("/.well-known/jwks.json", NewSsoHandler(signer).JWKS)
			r.Route("/api/v1", func(r chi.Router) {
				r.Use(middlewares.Auth(cmpAuthenticator{tc.identity}))
				r.Route("/services", func(r chi.Router) {
					NewServiceHandler(sq, aq, nil, nil, authorizer).Routes()(r)
					h.Routes()(r)
				})
			})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/services/"+svc.ID.String()+"/cmp-launch-token", strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			if tc.identity != nil {
				request.Header.Set("Authorization", "Bearer synthetic")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)
			require.Equal(t, tc.status, w.Code, w.Body.String())
			if tc.status != 200 {
				require.NotContains(t, w.Body.String(), `"token":`)
				return
			}
			require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			var out CmpLaunchTokenRes
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
			require.Equal(t, "https://cmp.example/landing", out.RedirectURL)
			public := httptest.NewRecorder()
			r.ServeHTTP(public, httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil))
			require.Equal(t, 200, public.Code)
			var jwks jose.JSONWebKeySet
			require.NoError(t, json.Unmarshal(public.Body.Bytes(), &jwks))
			parsed, err := jwt.ParseSigned(out.Token, []jose.SignatureAlgorithm{jose.ES256})
			require.NoError(t, err)
			require.Equal(t, "test-key", parsed.Headers[0].KeyID)
			var claims sso.LaunchClaims
			require.NoError(t, parsed.Claims(jwks, &claims))
			require.NoError(t, claims.Validate(jwt.Expected{Issuer: "https://core.example", AnyAudience: jwt.Audience{"cmp"}, Subject: tc.identity.Scope.ParticipantID.String(), Time: time.Now()}))
			require.Equal(t, svc.ID.String(), claims.ServiceID)
			require.Equal(t, tc.identity.Scope.ParticipantID.String(), claims.ParticipantID)
			require.Equal(t, tc.relation, claims.Relation)
			require.Equal(t, int64(90), int64(*claims.Expiry-*claims.IssuedAt))
			require.Equal(t, out.ExpiresAt.Unix(), claims.Expiry.Time().Unix())
			if tc.relation == "consumer" {
				require.Equal(t, "42", claims.Ext["order"])
			}
		})
	}
}

func TestCmpTarget(t *testing.T) {
	for _, cfg := range []*properties.JSON{nil, {}, {"cmpLandingUrl": "https://cmp"}, {"cmpAudience": "cmp"}, {"cmpLandingUrl": 42, "cmpAudience": "cmp"}, {"cmpLandingUrl": "", "cmpAudience": "cmp"}} {
		_, _, ok := cmpTarget(cfg)
		require.False(t, ok)
	}
}

func TestDisabledJWKS(t *testing.T) {
	w := httptest.NewRecorder()
	NewSsoHandler(nil).JWKS(w, httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil))
	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `{"keys":[]}`, w.Body.String())
}
