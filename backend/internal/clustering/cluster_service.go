package clustering

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/MatejGadjo/jadro/internal/logger"
	"github.com/jackc/pgx/v5"
)

// ClusteringService assigns articles with ready embeddings to clusters.
type ClusteringService struct {
	store ClusteringStore
	log   *logger.Logger
	// assignMu serializes the find-or-create critical section in AssignArticle.
	// Cluster assignment runs concurrently on the shared worker pool; without
	// this lock, a burst of same-event articles each read "no matching cluster"
	// before any of them inserts one, so each spawns a rival cluster and one
	// story fragments across clusters. Correct for the single-instance
	// deployment; a multi-instance setup would need a DB-level lock instead.
	assignMu sync.Mutex
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

	// Run the find-or-create decision under assignMu. The lock is acquired and
	// released inside this closure via defer, so a panic in any of the DB calls
	// can never leak the mutex. A leaked lock here is catastrophic: it deadlocks
	// every future cluster assignment and, because they share the worker pool,
	// starves embedding too — the exact freeze observed in production. The
	// recover also surfaces the panic in logs instead of the pool swallowing it.
	//
	// Semantics preserved: the lock spans the ANN read through the vector insert
	// (so a concurrent assignment sees a just-created cluster) and is released
	// before the cosmetic score/title work below.
	var createdClusterID, bestClusterID int64
	err = func() (err error) {
		s.assignMu.Lock()
		defer s.assignMu.Unlock()
		defer func() {
			if r := recover(); r != nil {
				s.log.Errorw("panic in cluster assignment critical section",
					"article_id", article.ID, "recover", r, "stack", string(debug.Stack()))
				err = fmt.Errorf("cluster assignment panic for article %d: %v", article.ID, r)
			}
		}()

		best, bestSim, err := s.bestClusterByEmbedding(ctx, embedding)
		if err != nil {
			return err
		}
		if best == 0 || bestSim < SimilarityThreshold {
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
			createdClusterID = clusterID
			return nil
		}
		bestClusterID = best
		return nil
	}()
	if err != nil {
		return err
	}

	if createdClusterID != 0 {
		if err := s.updateClusterScorePreserveUpdatedAt(ctx, createdClusterID); err != nil {
			s.log.Warnw("failed to update new cluster score", "cluster_id", createdClusterID, "error", err)
		}
		return nil
	}

	// Existing-cluster path: attaching does not race the find-or-create
	// invariant, so it runs outside the lock.
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
