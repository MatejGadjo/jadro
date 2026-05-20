package jobs

import (
	"context"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/ranking"
	"github.com/MatejGadjo/NewsAggregator/internal/storage/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RescoreJob recalculates scores for all clusters active in the last 48 hours.
// Velocity (articles_last_6h) is updated first, then all scores are written in
// a single unnest statement so the sort order changes atomically.
type RescoreJob struct {
	pool *pgxpool.Pool
	log  *logger.Logger
}

func NewRescoreJob(pool *pgxpool.Pool, log *logger.Logger) *RescoreJob {
	return &RescoreJob{pool: pool, log: log}
}

func (j *RescoreJob) Run(ctx context.Context) error {
	if err := postgres.UpdateClusterVelocityBatch(ctx, j.pool, j.log); err != nil {
		return err
	}

	clusters, err := postgres.ListActiveClusters(ctx, j.pool, j.log)
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
		newest := c.LastSeenAt
		created := c.CreatedAt
		ids[i] = c.ID
		scores[i] = ranking.ComputeClusterScore(c.ArticleCount, &newest, &created, c.SourceCount, c.ArticlesLast6h)
	}

	if err := postgres.UpdateClusterScoresBatch(ctx, j.pool, j.log, ids, scores); err != nil {
		return err
	}

	j.log.Infow("rescore job completed", "clusters_rescored", len(ids))
	return nil
}
