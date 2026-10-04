package storage

import (
	"context"

	"github.com/MatejGadjo/jadro/domain"
	"github.com/MatejGadjo/jadro/internal/storage/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

func (c *Client) UpsertPushSubscription(ctx context.Context, sub domain.PushSubscription) error {
	return c.q.UpsertPushSubscription(ctx, sqlc.UpsertPushSubscriptionParams{
		Endpoint: sub.Endpoint,
		P256dh:   sub.P256dh,
		Auth:     sub.Auth,
	})
}

func (c *Client) ListPushSubscriptions(ctx context.Context) ([]domain.PushSubscription, error) {
	rows, err := c.q.ListPushSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PushSubscription, len(rows))
	for i, r := range rows {
		out[i] = domain.PushSubscription{
			Endpoint: r.Endpoint,
			P256dh:   r.P256dh,
			Auth:     r.Auth,
		}
	}
	return out, nil
}

func (c *Client) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error {
	return c.q.DeletePushSubscriptionByEndpoint(ctx, endpoint)
}

// InsertPushSend records that clusterID was pushed (clusterID <= 0 → NULL, a
// digest-fallback send).
func (c *Client) InsertPushSend(ctx context.Context, clusterID int64) error {
	id := pgtype.Int8{}
	if clusterID > 0 {
		id = pgtype.Int8{Int64: clusterID, Valid: true}
	}
	return c.q.InsertPushSend(ctx, id)
}

// ClusterSentRecently reports whether clusterID was pushed within the last
// withinHours hours (used to avoid pushing the same story two days running).
func (c *Client) ClusterSentRecently(ctx context.Context, clusterID int64, withinHours int) (bool, error) {
	return c.q.ClusterSentRecently(ctx, sqlc.ClusterSentRecentlyParams{
		ClusterID: pgtype.Int8{Int64: clusterID, Valid: true},
		Column2:   int32(withinHours),
	})
}
