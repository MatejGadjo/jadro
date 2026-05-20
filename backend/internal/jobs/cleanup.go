package jobs

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
)

// CleanupJob periodically deletes articles older than the retention window and
// any clusters that become empty as a result.
type CleanupJob struct {
	pool            *pgxpool.Pool
	log             *logger.Logger
	retentionDays   int
	cron            *cron.Cron
	running         atomic.Bool
	lastCleanupAt   atomic.Value // stores time.Time
}

func (j *CleanupJob) LastCleanupAt() *time.Time {
	v := j.lastCleanupAt.Load()
	if v == nil {
		return nil
	}
	t := v.(time.Time)
	return &t
}

func NewCleanupJob(pool *pgxpool.Pool, log *logger.Logger) *CleanupJob {
	return &CleanupJob{
		pool: pool,
		log:  log,
		cron: cron.New(),
	}
}

func (j *CleanupJob) Start(cronExpr string, retentionDays int) error {
	j.retentionDays = retentionDays
	if _, err := j.cron.AddFunc(cronExpr, j.run); err != nil {
		return err
	}
	j.cron.Start()
	return nil
}

func (j *CleanupJob) Stop() {
	j.cron.Stop()
}

func (j *CleanupJob) run() {
	if !j.running.CompareAndSwap(false, true) {
		j.log.Warnw("cleanup skipped: previous run still in progress")
		return
	}
	defer j.running.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	j.log.Infow("cleanup run started", "retention_days", j.retentionDays)

	interval := time.Duration(j.retentionDays) * 24 * time.Hour
	cutoff := time.Now().UTC().Add(-interval)

	tag, err := j.pool.Exec(ctx,
		`DELETE FROM articles WHERE COALESCE(published_at, fetched_at) < $1`,
		cutoff,
	)
	if err != nil {
		j.log.Errorw("cleanup failed: delete articles", "error", err)
		return
	}
	articlesDeleted := tag.RowsAffected()

	tag, err = j.pool.Exec(ctx,
		`DELETE FROM clusters WHERE id NOT IN (SELECT DISTINCT cluster_id FROM cluster_articles)`,
	)
	if err != nil {
		j.log.Errorw("cleanup failed: delete orphaned clusters", "error", err)
		return
	}
	clustersDeleted := tag.RowsAffected()

	j.lastCleanupAt.Store(time.Now())

	j.log.Infow("cleanup run completed",
		"articles_deleted", articlesDeleted,
		"clusters_deleted", clustersDeleted,
		"retention_days", j.retentionDays,
	)
}
