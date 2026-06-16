package jobs

import (
	"context"
	"fmt"

	"github.com/MatejGadjo/NewsAggregator/internal/config"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/fx"
)

// AfterIngestFunc runs after a successful ingest cycle (e.g. cache warming).
type AfterIngestFunc func(context.Context)

// Cron drives the in-process ingest and cleanup schedules. It replaces the
// former standalone scheduler binary, which triggered these jobs over HTTP.
type Cron struct {
	log         *logger.Logger
	cron        *cron.Cron
	ingest      *IngestRunner
	cleanup     *CleanupJob
	afterIngest AfterIngestFunc
	ingestCron  string
	cleanupCron string
}

func NewCron(
	lc fx.Lifecycle,
	cfg config.JobsConfig,
	log *logger.Logger,
	ingest *IngestRunner,
	cleanup *CleanupJob,
	afterIngest AfterIngestFunc,
) *Cron {
	c := &Cron{
		log:         log,
		cron:        cron.New(),
		ingest:      ingest,
		cleanup:     cleanup,
		afterIngest: afterIngest,
		ingestCron:  cfg.IngestCron,
		cleanupCron: cfg.CleanupCron,
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error { return c.start() },
		OnStop: func(_ context.Context) error {
			<-c.cron.Stop().Done()
			return nil
		},
	})

	return c
}

func (c *Cron) start() error {
	if _, err := c.cron.AddFunc(c.ingestCron, c.runIngest); err != nil {
		return fmt.Errorf("register ingest cron %q: %w", c.ingestCron, err)
	}
	if _, err := c.cron.AddFunc(c.cleanupCron, c.runCleanup); err != nil {
		return fmt.Errorf("register cleanup cron %q: %w", c.cleanupCron, err)
	}
	c.cron.Start()
	c.log.Infow("job scheduler started", "ingest_cron", c.ingestCron, "cleanup_cron", c.cleanupCron)
	return nil
}

func (c *Cron) runIngest() {
	if !c.ingest.RunOnce(c.afterIngest) {
		c.log.Warnw("ingest skipped: previous run still in progress")
	}
}

func (c *Cron) runCleanup() {
	if !c.cleanup.RunOnce() {
		c.log.Warnw("cleanup skipped: previous run still in progress")
	}
}
