package clustering

import (
	"context"
	"time"

	"github.com/MatejGadjo/jadro/domain"
)

// EmbeddingStore is the persistence surface the EmbeddingService needs.
type EmbeddingStore interface {
	GetArticleByID(ctx context.Context, id int64) (domain.Article, error)
	UpdateArticleEmbeddingStatus(ctx context.Context, id int64, status domain.EmbeddingStatus) error
	UpdateArticleEmbeddingAndStatus(ctx context.Context, id int64, embedding []float32, status domain.EmbeddingStatus) error
	MarkProcessingEmbeddingsAsFailed(ctx context.Context) (int64, error)
	MarkPendingEmbeddingsAsFailed(ctx context.Context) (int64, error)
}

type ClusteringStore interface {
	// read
	GetArticleByID(ctx context.Context, id int64) (domain.Article, error)
	GetArticleEmbedding(ctx context.Context, id int64) ([]float32, error)
	GetClusterIDForArticle(ctx context.Context, articleID int64) (int64, error)
	FindSimilarClusterNeighbors(ctx context.Context, embedding []float32, hours, maxNeighbors, maxAgeHours int) ([]domain.ClusterNeighbor, error)
	FindNearestArticleSimilarityInClusters(ctx context.Context, embedding []float32, clusterIDs []int64) ([]domain.ClusterNeighbor, error)
	GetClusterByID(ctx context.Context, id int64) (domain.Cluster, error)
	GetRecentArticleCount(ctx context.Context, clusterID int64) (int64, error)
	GetArticlesInCluster(ctx context.Context, clusterID int64) ([]domain.Article, error)
	GetArticleEmbeddingsByIDs(ctx context.Context, ids []int64) ([]domain.ArticleEmbedding, error)

	// write
	CreateCluster(ctx context.Context, headline, summary, displayTitle, topic string, lastSeenAt time.Time) (int64, error)
	InsertClusterSource(ctx context.Context, clusterID, sourceID int64) error
	AttachArticleToCluster(ctx context.Context, clusterID, articleID int64) error
	InsertClusterVector(ctx context.Context, clusterID int64, vector []float32, articleCount int) error
	UpdateClusterScoreValue(ctx context.Context, id int64, score float64) error
	UpdateClusterDisplayTitle(ctx context.Context, id int64, displayTitle string) error

	// transactional
	AttachAndUpdateVector(ctx context.Context, clusterID, articleID, sourceID int64, lastSeenAt time.Time, embedding []float32) (bool, error)
}
