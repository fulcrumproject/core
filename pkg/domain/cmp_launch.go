// CMP launch commands record successful service-scoped token issuance.
// The audit contains trusted scope and target, never tokens or opaque ext data.
package domain

import (
	"context"

	"github.com/fulcrumproject/core/pkg/properties"
)

const EventTypeServiceCmpLaunched EventType = "service.cmp_launch"

// CmpLaunchCommander records launch events independently of service mutation.
// It is shared by applications mounting Core's CMP launch handler.
type CmpLaunchCommander struct {
	store Store
}

func NewCmpLaunchCommander(store Store) *CmpLaunchCommander {
	return &CmpLaunchCommander{store: store}
}

// Record must succeed before the signed token is returned to the caller.
func (c *CmpLaunchCommander) Record(ctx context.Context, svc *Service, relation, audience string) error {
	event, err := NewEvent(EventTypeServiceCmpLaunched, WithService(svc), WithInitiatorCtx(ctx))
	if err != nil {
		return err
	}
	event.Payload = properties.JSON{"relation": relation, "audience": audience}
	return c.store.EventRepo().Create(ctx, event)
}
