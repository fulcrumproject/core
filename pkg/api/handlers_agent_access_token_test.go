// Tests agent-only issuance through real authentication and authorization middleware.
// Untrusted bodies and agent configuration can never override operator grants.
package api

import (
	"context"
	"encoding/json"
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

func TestAgentAccessTokenRoutes(t *testing.T) {
	id, provider, other := properties.NewUUID(), properties.NewUUID(), properties.NewUUID()
	agentIdentity := func(agentID, pid *properties.UUID) *auth.Identity {
		return &auth.Identity{ID: properties.NewUUID(), Role: auth.RoleAgent, Scope: auth.IdentityScope{AgentID: agentID, ParticipantID: pid}}
	}
	for _, tc := range []struct {
		name, body                                       string
		identity                                         *auth.Identity
		status                                           int
		noGrant, disabled, missing, mismatch, auditError bool
	}{
		{name: "no body", identity: agentIdentity(&id, &provider), status: 200},
		{name: "all overrides ignored", identity: agentIdentity(&id, &provider), body: `{"app":"other","agentId":"other","roles":["admin"],"aud":"evil","exp":9999999999,"iss":"evil","signingKey":"evil"}`, status: 200},
		{name: "malformed body ignored", identity: agentIdentity(&id, &provider), body: `not JSON`, status: 200},
		{name: "no grant despite agent config", identity: agentIdentity(&id, &provider), status: 403, noGrant: true},
		{name: "other agent with same provider", identity: agentIdentity(&other, &provider), status: 403, noGrant: true},
		{name: "participant", identity: &auth.Identity{Role: auth.RoleParticipant, Scope: auth.IdentityScope{ParticipantID: &provider, AgentID: &id}}, status: 403},
		{name: "admin", identity: &auth.Identity{Role: auth.RoleAdmin, Scope: auth.IdentityScope{AgentID: &id}}, status: 403},
		{name: "missing credentials", status: 401},
		{name: "missing agent scope", identity: agentIdentity(nil, &provider), status: 403},
		{name: "zero agent scope", identity: agentIdentity(new(properties.UUID), &provider), status: 403},
		{name: "missing participant scope", identity: agentIdentity(&id, nil), status: 403},
		{name: "mismatched provider", identity: agentIdentity(&id, &other), status: 403},
		{name: "mismatched loaded agent", identity: agentIdentity(&id, &provider), status: 403, mismatch: true},
		{name: "missing agent", identity: agentIdentity(&id, &provider), status: 404, missing: true},
		{name: "audit error", identity: agentIdentity(&id, &provider), status: 500, auditError: true},
		{name: "disabled signer", identity: agentIdentity(&id, &provider), status: 503, disabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			aq, store, events := domain.NewMockAgentQuerier(t), domain.NewMockStore(t), domain.NewMockEventRepository(t)
			signer := cmpTestSigner(t)
			grants := map[string]sso.ApplicationGrant{}
			if !tc.noGrant && !tc.disabled {
				grants[id.String()] = sso.ApplicationGrant{Audience: "fae", Roles: []string{"pve"}}
			}
			if tc.disabled {
				signer = nil
			}
			policy, err := sso.NewApplicationTokens(signer, sso.ApplicationConfig{Grants: grants})
			require.NoError(t, err)
			if tc.identity != nil && tc.identity.Role == auth.RoleAgent && tc.identity.Scope.AgentID != nil && *tc.identity.Scope.AgentID != (properties.UUID{}) && tc.identity.Scope.ParticipantID != nil {
				agent := &domain.Agent{BaseEntity: domain.BaseEntity{ID: *tc.identity.Scope.AgentID}, ProviderID: provider, Status: domain.AgentDisconnected,
					Configuration: &properties.JSON{"audience": "evil", "roles": []any{"admin"}}}
				if tc.mismatch {
					agent.ID = other
				}
				var loadErr error
				if tc.missing {
					agent = nil
					loadErr = domain.NewNotFoundErrorf("agent missing")
				}
				aq.EXPECT().Get(mock.Anything, *tc.identity.Scope.AgentID).Return(agent, loadErr).Once()
			}
			if tc.status == 200 || tc.auditError {
				store.EXPECT().EventRepo().Return(events).Once()
				var auditErr error
				if tc.auditError {
					auditErr = errors.New("audit unavailable")
				}
				events.EXPECT().Create(mock.Anything, mock.Anything).Run(func(ctx context.Context, e *domain.Event) {
					require.Equal(t, domain.EventTypeAgentAccessTokenIssued, e.Type)
					require.Equal(t, id, *e.EntityID)
					require.Equal(t, id, *e.AgentID)
					require.Equal(t, provider, *e.ProviderID)
					require.Equal(t, tc.identity.ID.String(), e.InitiatorID)
					require.Equal(t, properties.JSON{"audience": "fae", "roles": []string{"pve"}}, e.Payload)
				}).Return(auditErr).Once()
			}
			h := NewAgentAccessTokenHandler(aq, domain.NewAgentAccessTokenCommander(store), authz.NewRuleBasedAuthorizer(authz.Rules), policy)
			r := chi.NewRouter()
			r.Use(middlewares.Auth(cmpAuthenticator{tc.identity}))
			r.Route("/api/v1/agents", func(r chi.Router) {
				NewAgentHandler(aq, domain.NewMockAgentCommander(t), authz.NewRuleBasedAuthorizer(authz.Rules)).Routes()(r)
				h.Routes()(r)
			})
			request := httptest.NewRequest(http.MethodPost, "/api/v1/agents/me/access-token?app=evil&aud=evil", strings.NewReader(tc.body))
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
			var result AgentAccessTokenRes
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
			var fields map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fields))
			require.Len(t, fields, 2)
			parsed, err := jwt.ParseSigned(result.Token, []jose.SignatureAlgorithm{jose.ES256})
			require.NoError(t, err)
			var claims sso.ApplicationClaims
			require.NoError(t, parsed.Claims(signer.JWKS(), &claims))
			require.NoError(t, claims.Validate(jwt.Expected{Issuer: "https://core.example", AnyAudience: jwt.Audience{"fae"}, Time: time.Now()}))
			require.Equal(t, id.String(), claims.App)
			require.NotEqual(t, tc.identity.ID.String(), claims.App)
			require.Equal(t, []string{"pve"}, claims.Roles)
			require.Equal(t, int64(300), int64(*claims.Expiry-*claims.IssuedAt))
			require.Equal(t, result.ExpiresAt.Unix(), claims.Expiry.Time().Unix())
		})
	}
}
