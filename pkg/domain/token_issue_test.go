// Tests issuance identity and service context independently of HTTP/signing.
// Persisted records and native authorization must agree with authenticated scope.
package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTokenPrincipal(t *testing.T) {
	provider, consumer, stranger, agentID, otherAgent, serviceID, zero := properties.NewUUID(), properties.NewUUID(), properties.NewUUID(), properties.NewUUID(), properties.NewUUID(), properties.NewUUID(), properties.UUID{}
	participant := func(pid *properties.UUID) *auth.Identity {
		return &auth.Identity{ID: properties.NewUUID(), Role: auth.RoleParticipant, Scope: auth.IdentityScope{ParticipantID: pid}}
	}
	agent := func(pid, aid *properties.UUID) *auth.Identity {
		return &auth.Identity{ID: properties.NewUUID(), Role: auth.RoleAgent, Scope: auth.IdentityScope{ParticipantID: pid, AgentID: aid}}
	}
	for _, tc := range []struct {
		name                                                                                                                                                                                    string
		identity                                                                                                                                                                                *auth.Identity
		withService                                                                                                                                                                             bool
		relation                                                                                                                                                                                string
		denied                                                                                                                                                                                  bool
		missingParticipant, missingAgent, wrongParticipant, wrongAgent, wrongProvider, foreignService, wrongAssignment, inconsistentService, staleScope, missingService, scopeError, readDenied bool
	}{
		{name: "participant identity", identity: participant(&provider)},
		{name: "consumer identity", identity: participant(&consumer)},
		{name: "agent identity ignores credential record ID", identity: agent(&provider, &agentID)},
		{name: "provider service", identity: participant(&provider), withService: true, relation: "provider"},
		{name: "consumer service", identity: participant(&consumer), withService: true, relation: "consumer"},
		{name: "assigned agent service", identity: agent(&provider, &agentID), withService: true, relation: "provider"},
		{name: "missing participant scope", identity: participant(nil), denied: true},
		{name: "zero participant scope", identity: participant(&zero), denied: true},
		{name: "participant with agent scope", identity: &auth.Identity{Role: auth.RoleParticipant, Scope: auth.IdentityScope{ParticipantID: &provider, AgentID: &agentID}}, denied: true},
		{name: "agent missing scope", identity: agent(&provider, nil), denied: true},
		{name: "agent zero scope", identity: agent(&provider, &zero), denied: true},
		{name: "admin cannot impersonate participant", identity: &auth.Identity{Role: auth.RoleAdmin, Scope: auth.IdentityScope{ParticipantID: &provider}}, denied: true},
		{name: "unknown role", identity: &auth.Identity{Role: "other", Scope: auth.IdentityScope{ParticipantID: &provider, AgentID: &agentID}}, denied: true},
		{name: "missing persisted participant", identity: participant(&provider), denied: true, missingParticipant: true},
		{name: "wrong persisted participant", identity: participant(&provider), denied: true, wrongParticipant: true},
		{name: "missing persisted agent", identity: agent(&provider, &agentID), denied: true, missingAgent: true},
		{name: "wrong persisted agent", identity: agent(&provider, &agentID), denied: true, wrongAgent: true},
		{name: "agent provider differs from scope", identity: agent(&consumer, &agentID), denied: true, wrongProvider: true},
		{name: "foreign participant service", identity: participant(&stranger), withService: true, denied: true},
		{name: "foreign agent service", identity: agent(&provider, &agentID), withService: true, denied: true, foreignService: true},
		{name: "other assigned agent same provider", identity: agent(&provider, &agentID), withService: true, denied: true, wrongAssignment: true},
		{name: "assigned agent inconsistent service provider", identity: agent(&provider, &agentID), withService: true, denied: true, inconsistentService: true},
		{name: "stale auth scope cannot grant ownership", identity: participant(&stranger), withService: true, denied: true, staleScope: true},
		{name: "native authorization cannot be bypassed", identity: participant(&provider), withService: true, denied: true, readDenied: true},
		{name: "service scope loading fails", identity: participant(&provider), withService: true, denied: true, scopeError: true},
		{name: "missing loaded service", identity: participant(&provider), withService: true, denied: true, missingService: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, pq, aq, sq := NewMockStore(t), NewMockParticipantRepository(t), NewMockAgentRepository(t), NewMockServiceRepository(t)
			store.EXPECT().ParticipantRepo().Return(pq).Maybe()
			store.EXPECT().AgentRepo().Return(aq).Maybe()
			store.EXPECT().ServiceRepo().Return(sq).Maybe()
			pid := provider
			if tc.identity.Scope.ParticipantID != nil {
				pid = *tc.identity.Scope.ParticipantID
			}
			persisted := &Participant{BaseEntity: BaseEntity{ID: pid}}
			if tc.wrongParticipant {
				persisted.ID = stranger
			}
			if tc.missingParticipant {
				persisted = nil
			}
			pq.EXPECT().Get(mock.Anything, pid).Return(persisted, nil).Maybe()
			persistedAgent := &Agent{BaseEntity: BaseEntity{ID: agentID}, ProviderID: pid}
			if tc.wrongAgent {
				persistedAgent.ID = otherAgent
			}
			if tc.wrongProvider {
				persistedAgent.ProviderID = provider
			}
			if tc.missingAgent {
				persistedAgent = nil
			}
			aq.EXPECT().Get(mock.Anything, agentID).Return(persistedAgent, nil).Maybe()
			svc := &Service{BaseEntity: BaseEntity{ID: serviceID}, ProviderID: provider, ConsumerID: consumer, AgentID: agentID}
			if tc.foreignService {
				svc.ProviderID = stranger
				svc.ConsumerID = stranger
				svc.AgentID = otherAgent
			}
			if tc.wrongAssignment {
				svc.AgentID = otherAgent
			}
			if tc.inconsistentService {
				svc.ProviderID = stranger
			}
			scope := &authz.DefaultObjectScope{ProviderID: &svc.ProviderID, ConsumerID: &svc.ConsumerID, AgentID: &svc.AgentID}
			if tc.staleScope {
				scope = &authz.DefaultObjectScope{ProviderID: &stranger}
			}
			var scopeErr error
			if tc.scopeError {
				scopeErr = errors.New("scope query unavailable")
			}
			sq.EXPECT().AuthScope(mock.Anything, serviceID).Return(scope, scopeErr).Maybe()
			if tc.missingService {
				svc = nil
			}
			sq.EXPECT().Get(mock.Anything, serviceID).Return(svc, nil).Maybe()
			rules := authz.Rules
			if tc.readDenied {
				rules = nil
			}
			commander := NewTokenIssueCommander(store, authz.NewRuleBasedAuthorizer(rules))
			var requested *properties.UUID
			if tc.withService {
				requested = &serviceID
			}
			got, err := commander.Resolve(auth.WithIdentity(context.Background(), tc.identity), requested)
			if tc.denied {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, pid, got.ParticipantID)
			require.Equal(t, tc.identity.Role, got.Role)
			if tc.identity.Role == auth.RoleAgent {
				require.Equal(t, agentID, *got.AgentID)
				require.NotEqual(t, tc.identity.ID, *got.AgentID)
			} else {
				require.Nil(t, got.AgentID)
			}
			require.Equal(t, tc.relation, got.Relation)
			if tc.withService {
				require.Equal(t, serviceID, got.Service.ID)
			} else {
				require.Nil(t, got.Service)
			}
		})
	}
}

func TestTokenIssueAudit(t *testing.T) {
	pid, aid, sid := properties.NewUUID(), properties.NewUUID(), properties.NewUUID()
	for _, withService := range []bool{false, true} {
		for _, role := range []auth.Role{auth.RoleParticipant, auth.RoleAgent} {
			t.Run(string(role)+map[bool]string{false: " identity", true: " service"}[withService], func(t *testing.T) {
				store, events := NewMockStore(t), NewMockEventRepository(t)
				store.EXPECT().EventRepo().Return(events).Once()
				identity := &auth.Identity{ID: properties.NewUUID(), Role: role}
				p := &TokenPrincipal{Role: role, ParticipantID: pid}
				if role == auth.RoleAgent {
					p.AgentID = &aid
				}
				if withService {
					p.Service = &Service{BaseEntity: BaseEntity{ID: sid}, ProviderID: pid, ConsumerID: properties.NewUUID(), AgentID: aid}
					p.Relation = "provider"
				}
				expectedErr := errors.New("audit unavailable")
				events.EXPECT().Create(mock.Anything, mock.Anything).Run(func(ctx context.Context, e *Event) {
					require.Equal(t, EventTypeTokenIssued, e.Type)
					require.Equal(t, identity.ID.String(), e.InitiatorID)
					require.Equal(t, pid, *e.ParticipantID)
					require.Equal(t, string(role), e.Payload["role"])
					require.Equal(t, "recipient", e.Payload["audience"])
					expected := 2
					if withService {
						expected++
						require.Equal(t, sid, *e.EntityID)
						require.Equal(t, "provider", e.Payload["relation"])
					}
					require.Len(t, e.Payload, expected)
				}).Return(expectedErr).Once()
				require.ErrorIs(t, NewTokenIssueCommander(store, nil).Record(auth.WithIdentity(context.Background(), identity), p, "recipient"), expectedErr)
			})
		}
	}
}
