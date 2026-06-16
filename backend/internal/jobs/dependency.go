package jobs

import (
	"context"
	"time"

	"github.com/MatejGadjo/NewsAggregator/domain"
)

// Embedder generates and persists article embeddings. Implemented by clustering.EmbeddingService.
type Embedder interface {
	GenerateAndStore(ctx context.Context, articleID int64) error
	RecoverStaleEmbeddings(ctx context.Context)
}

// ClusterAssigner assigns an article with a ready embedding to a cluster.
// Implemented by clustering.ClusteringService.
type ClusterAssigner interface {
	AssignArticle(ctx context.Context, articleID int64) error
}

type RescoreStore interface {
	UpdateClusterVelocityBatch(ctx context.Context) error
	ListActiveClusters(ctx context.Context) ([]domain.Cluster, error)
	UpdateClusterScoresBatch(ctx context.Context, ids []int64, scores []float64) error
}

type CleanupStore interface {
	DeleteOldArticles(ctx context.Context, cutoff time.Time) (int64, error)
	DeleteOrphanedClusters(ctx context.Context) (int64, error)
}
