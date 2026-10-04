package handlers

import (
	"context"

	"github.com/MatejGadjo/jadro/internal/service"
)

type FeedServicer interface {
	GetFeed(ctx context.Context, topic *string, page int) ([]byte, error)
	Sitemap(ctx context.Context) ([]byte, error)
	WarmCache(ctx context.Context)
}

type ClusterServicer interface {
	GetCluster(ctx context.Context, id int64) ([]byte, int, error)
	GetLatestArticles(ctx context.Context, limit int) ([]service.LatestArticleItem, error)
	ClearArticleImage(ctx context.Context, id int64) error
	WasEverAssigned(ctx context.Context, id int64) (bool, error)
}

type HealthServicer interface {
	Check(ctx context.Context) (service.HealthStatus, int)
}
