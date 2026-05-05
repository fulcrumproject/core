//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/fulcrumproject/core/pkg/api"
	"github.com/stretchr/testify/require"
)

func testEvent(t *testing.T, env *Env) {
	t.Run("admin lists events", func(t *testing.T) {
		page := mustList[api.EventRes](t, env.AdminClient, "/events")
		require.GreaterOrEqual(t, page.TotalItems, int64(1))
	})

	t.Run("subscriber leases events and acks", func(t *testing.T) {
		subscriberID := env.Seed.EventSub.SubscriberID
		instanceID := "e2e-instance-" + uniq()

		var lease api.EventLeaseRes
		resp, err := env.AdminClient.R().
			SetBody(api.EventLeaseReq{
				SubscriberID: subscriberID,
				InstanceID:   instanceID,
			}).
			SetResult(&lease).
			Post("/events/lease")
		require.NoError(t, err)
		require.Equalf(t, http.StatusOK, resp.StatusCode(), "lease: %s", resp.String())
		require.NotEmpty(t, lease.Events, "expected leased events from earlier subtests")

		// Pick the highest sequence we've seen.
		var maxSeq int64
		for _, e := range lease.Events {
			if e.SequenceNumber > maxSeq {
				maxSeq = e.SequenceNumber
			}
		}
		require.Greater(t, maxSeq, int64(0))

		var ack api.EventAckRes
		resp, err = env.AdminClient.R().
			SetBody(api.EventAckReq{
				SubscriberID:               subscriberID,
				InstanceID:                 instanceID,
				LastEventSequenceProcessed: maxSeq,
			}).
			SetResult(&ack).
			Post("/events/ack")
		require.NoError(t, err)
		require.Equalf(t, http.StatusOK, resp.StatusCode(), "ack: %s", resp.String())
		require.Equal(t, maxSeq, ack.LastEventSequenceProcessed)

		// Re-acking the same sequence must 409, not no-op: per-subscriber sequences are strictly increasing.
		resp, err = env.AdminClient.R().
			SetBody(api.EventAckReq{
				SubscriberID:               subscriberID,
				InstanceID:                 instanceID,
				LastEventSequenceProcessed: maxSeq,
			}).
			Post("/events/ack")
		require.NoError(t, err)
		require.Equalf(t, http.StatusConflict, resp.StatusCode(), "re-ack of same sequence must 409: %s", resp.String())
	})

	t.Run("lease without subscriberId is rejected", func(t *testing.T) {
		resp, err := env.AdminClient.R().
			SetBody(api.EventLeaseReq{InstanceID: "x"}).
			Post("/events/lease")
		require.NoError(t, err)
		require.Equalf(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", resp.String())
	})
}
