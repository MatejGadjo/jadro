package push

import (
	"context"

	"github.com/MatejGadjo/jadro/domain"
)

// Store persists subscriptions and the daily send log.
type Store interface {
	ListPushSubscriptions(ctx context.Context) ([]domain.PushSubscription, error)
	DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error
	InsertPushSend(ctx context.Context, clusterID int64) error
	ClusterSentRecently(ctx context.Context, clusterID int64, withinHours int) (bool, error)
}

// FeedReader supplies the ranked clusters used to pick the day's story. Both
// methods already exist on the storage client (reused, no new SQL).
type FeedReader interface {
	ListTopClustersByScore(ctx context.Context, limit int32) ([]domain.Cluster, error)
	ListClusterArticlesWithSources(ctx context.Context, clusterID int64) ([]domain.ClusterArticleSource, error)
}

// Sender delivers one encrypted payload to one subscription. The real
// implementation wraps webpush.SendNotification; tests supply a fake. It returns
// the endpoint's HTTP status so the caller can prune dead subscriptions (404/410).
type Sender interface {
	Send(ctx context.Context, payload []byte, sub domain.PushSubscription) (status int, err error)
}
