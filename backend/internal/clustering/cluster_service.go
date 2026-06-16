package clustering

import (
	"context"
	"errors"
	"fmt"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/jackc/pgx/v5"
)

// ClusteringService assigns articles with ready embeddings to clusters.
type ClusteringService struct {
	store ClusteringStore
	log   *logger.Logger
}

// NewClusteringService wires the clustering service dependencies.
func NewClusteringService(store ClusteringStore, log *logger.Logger) *ClusteringService {
	return &ClusteringService{store: store, log: log}
}

// AssignArticle assigns an article with a ready embedding to the best-matching
// cluster, using ANN neighbors to find candidates. It creates a new cluster when
// no existing cluster is similar enough, and is a no-op if the article is already
// clustered (e.g. a deduplicated URL).
func (s *ClusteringService) AssignArticle(ctx context.Context, articleID int64) error {
	article, err := s.store.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("article not found: %d", articleID)
		}
		return err
	}
	if article.EmbeddingStatus != "ready" {
		return ErrMissingEmbedding
	}

	embedding, err := s.store.GetArticleEmbedding(ctx, articleID)
	if err != nil {
		return err
	}
	if len(embedding) == 0 {
		return ErrMissingEmbedding
	}

	// Skip if already clustered (e.g., deduplicated URL).
	if _, lookupErr := s.store.GetClusterIDForArticle(ctx, article.ID); lookupErr == nil {
		return nil
	} else if !errors.Is(lookupErr, pgx.ErrNoRows) {
		return lookupErr
	}

	bestClusterID, bestSim, err := s.bestClusterByEmbedding(ctx, embedding)
	if err != nil {
		return err
	}

	if bestClusterID == 0 || bestSim < SimilarityThreshold {
		clusterID, err := s.createClusterForArticle(ctx, article)
		if err != nil {
			return err
		}
		if err := s.store.AttachArticleToCluster(ctx, clusterID, article.ID); err != nil {
			return err
		}
		if err := s.store.InsertClusterVector(ctx, clusterID, embedding, 1); err != nil {
			return err
		}
		if err := s.updateClusterScorePreserveUpdatedAt(ctx, clusterID); err != nil {
			s.log.Warnw("failed to update new cluster score", "cluster_id", clusterID, "error", err)
		}
		return nil
	}

	attached, err := s.store.AttachAndUpdateVector(ctx, bestClusterID, article.ID, article.SourceID, lastSeenAtForArticle(article), embedding)
	if err != nil {
		return err
	}
	if !attached {
		return nil
	}

	if err := s.updateClusterScorePreserveUpdatedAt(ctx, bestClusterID); err != nil {
		s.log.Warnw("failed to update cluster score", "cluster_id", bestClusterID, "error", err)
	}
	if err := s.refreshClusterDisplayTitle(ctx, bestClusterID); err != nil {
		s.log.Warnw("failed to refresh cluster display title", "cluster_id", bestClusterID, "error", err)
	}
	return nil
}
