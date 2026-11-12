// Agent access token commands persist successful application-token issuance.
// Audit records contain trusted agent scope and grants, never signed tokens.
package domain

import (
	"context"

	"github.com/fulcrumproject/core/pkg/properties"
)

const EventTypeAgentAccessTokenIssued EventType = "agent.access_token_issued"

type AgentAccessTokenCommander struct{ store Store }

func NewAgentAccessTokenCommander(store Store) *AgentAccessTokenCommander {
	return &AgentAccessTokenCommander{store: store}
}

// Record must succeed before the token can leave the issuance handler.
func (c *AgentAccessTokenCommander) Record(ctx context.Context, agent *Agent, audience string, roles []string) error {
	event, err := NewEvent(EventTypeAgentAccessTokenIssued, WithAgent(agent), WithInitiatorCtx(ctx))
	if err != nil {
		return err
	}
	event.Payload = properties.JSON{"audience": audience, "roles": append([]string(nil), roles...)}
	return c.store.EventRepo().Create(ctx, event)
}
