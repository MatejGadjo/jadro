package jobs

import (
	"context"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/ranking"
)

// RescoreJob recalculates scores for all clusters active in the last 48 hours.
// Velocity (articles_last_6h) is updated first, then all scores are written in
// a single unnest statement so the sort order changes atomically.
type RescoreJob struct {
	store RescoreStore
	log   *logger.Logger
}

func NewRescoreJob(store RescoreStore, log *logger.Logger) *RescoreJob {
	return &RescoreJob{store: store, log: log}
}

func (j *RescoreJob) Run(ctx context.Context) error {
	if err := j.store.UpdateClusterVelocityBatch(ctx); err != nil {
		return err
	}

	clusters, err := j.store.ListActiveClusters(ctx)
	if err != nil {
		return err
	}

	if len(clusters) == 0 {
		j.log.Infow("rescore job completed", "clusters_rescored", 0)
		return nil
	}

	ids := make([]int64, len(clusters))
	scores := make([]float64, len(clusters))
	for i, c := range clusters {
		ids[i] = c.ID
		scores[i] = ranking.ComputeClusterScore(c.ArticleCount, &c.LastSeenAt, &c.CreatedAt, c.SourceCount, c.ArticlesLast6h)
	}

	if err := j.store.UpdateClusterScoresBatch(ctx, ids, scores); err != nil {
		return err
	}

	j.log.Infow("rescore job completed", "clusters_rescored", len(ids))
	return nil
}
