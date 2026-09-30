package jobs

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/MatejGadjo/jadro/internal/config"
	"github.com/MatejGadjo/jadro/internal/logger"
)

// CleanupJob deletes articles older than the retention window and any clusters
// that become empty as a result. Triggered on a schedule by the in-process cron.
type CleanupJob struct {
	store         CleanupStore
	log           *logger.Logger
	retentionDays int
	running       atomic.Bool
	lastCleanupAt atomic.Value // stores time.Time
}

func (j *CleanupJob) LastCleanupAt() *time.Time {
	v := j.lastCleanupAt.Load()
	if v == nil {
		return nil
	}
	t := v.(time.Time)
	return &t
}

func NewCleanupJob(cfg config.JobsConfig, store CleanupStore, log *logger.Logger) *CleanupJob {
	return &CleanupJob{
		store:         store,
		log:           log,
		retentionDays: cfg.CleanupRetentionDays,
	}
}

// RunOnce fires a single cleanup cycle in a new goroutine.
// Returns false if a run is already in progress (caller should respond 409).
func (j *CleanupJob) RunOnce() bool {
	if !j.running.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		defer j.running.Store(false)
		j.doCleanup()
	}()
	return true
}

func (j *CleanupJob) doCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	j.log.Infow("cleanup run started", "retention_days", j.retentionDays)

	interval := time.Duration(j.retentionDays) * 24 * time.Hour
	cutoff := time.Now().UTC().Add(-interval)

	articlesDeleted, err := j.store.DeleteOldArticles(ctx, cutoff)
	if err != nil {
		j.log.Errorw("cleanup failed: delete articles", "error", err)
		return
	}

	clustersDeleted, err := j.store.DeleteOrphanedClusters(ctx)
	if err != nil {
		j.log.Errorw("cleanup failed: delete orphaned clusters", "error", err)
		return
	}

	j.lastCleanupAt.Store(time.Now())

	j.log.Infow("cleanup run completed",
		"articles_deleted", articlesDeleted,
		"clusters_deleted", clustersDeleted,
		"retention_days", j.retentionDays,
	)
}
