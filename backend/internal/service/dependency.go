package service

import (
	"context"
	"time"

	"github.com/MatejGadjo/jadro/domain"
)

type FeedStore interface {
	ListClustersForFeedAll(ctx context.Context, limit, offset int32) ([]domain.Cluster, error)
	ListClustersForFeedByTopic(ctx context.Context, topic string, limit, offset int32) ([]domain.Cluster, error)
	ListTopClustersByScore(ctx context.Context, limit int32) ([]domain.Cluster, error)
	ListTopCategoryClustersWithArticles(ctx context.Context, topics []string, limit int32) ([]domain.ClusterWithFeaturedArticle, error)
	ListClusterArticlesWithSources(ctx context.Context, clusterID int64) ([]domain.ClusterArticleSource, error)
}

type ClusterStore interface {
	GetClusterByID(ctx context.Context, id int64) (domain.Cluster, error)
	GetArticlesInCluster(ctx context.Context, clusterID int64) ([]domain.Article, error)
	GetSourceByID(ctx context.Context, id int64) (domain.Source, error)
	ListLatestArticles(ctx context.Context, limit int32) ([]domain.LatestArticle, error)
	GetArticleByID(ctx context.Context, id int64) (domain.Article, error)
	ClearArticleImageURL(ctx context.Context, id int64) error
	MaxClusterID(ctx context.Context) (int64, error)
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	DeleteByPattern(ctx context.Context, pattern string) error
}

type HealthStore interface {
	Ping(ctx context.Context) error
	ArticleCount(ctx context.Context) (int, error)
	ClusterCount(ctx context.Context) (int, error)
}

type HealthCache interface {
	Ping(ctx context.Context) error
}
