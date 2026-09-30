package jobs

import (
	"context"
	"fmt"

	"github.com/MatejGadjo/jadro/internal/config"
	"github.com/MatejGadjo/jadro/internal/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/fx"
)

// AfterIngestFunc runs after a successful ingest cycle (e.g. cache warming).
type AfterIngestFunc func(context.Context)

// Cron drives the in-process ingest, cleanup and daily-push schedules. It
// replaces the former standalone scheduler binary, which triggered these jobs
// over HTTP.
type Cron struct {
	log           *logger.Logger
	cron          *cron.Cron
	ingest        *IngestRunner
	cleanup       *CleanupJob
	push          *PushJob
	afterIngest   AfterIngestFunc
	ingestCron    string
	cleanupCron   string
	pushCron      string // empty = push disabled (no VAPID keys); not registered
	ingestOnStart bool
}

func NewCron(
	lc fx.Lifecycle,
	cfg config.JobsConfig,
	pushCfg config.PushConfig,
	log *logger.Logger,
	ingest *IngestRunner,
	cleanup *CleanupJob,
	push *PushJob,
	afterIngest AfterIngestFunc,
) *Cron {
	c := &Cron{
		log:           log,
		cron:          cron.New(),
		ingest:        ingest,
		cleanup:       cleanup,
		push:          push,
		afterIngest:   afterIngest,
		ingestCron:    cfg.IngestCron,
		cleanupCron:   cfg.CleanupCron,
		ingestOnStart: cfg.IngestOnStart,
	}
	if pushCfg.Enabled() {
		// Pin the daily send to MK local time via a per-entry CRON_TZ prefix so
		// the global scheduler (ingest/cleanup) keeps using server time.
		c.pushCron = withCronTZ(pushCfg.Cron, pushCfg.Timezone)
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
	if c.pushCron != "" {
		if _, err := c.cron.AddFunc(c.pushCron, c.runPush); err != nil {
			return fmt.Errorf("register push cron %q: %w", c.pushCron, err)
		}
	}
	c.cron.Start()
	c.log.Infow("job scheduler started",
		"ingest_cron", c.ingestCron, "cleanup_cron", c.cleanupCron, "push_cron", c.pushCron)
	if c.ingestOnStart {
		c.log.Infow("running initial ingest (INGEST_ON_START=true)")
		go c.runIngest()
	}
	return nil
}

// withCronTZ prefixes a cron spec with a CRON_TZ directive so robfig/cron
// evaluates that single entry in the given location. Empty/UTC tz → unchanged.
func withCronTZ(spec, tz string) string {
	if tz == "" || tz == "UTC" {
		return spec
	}
	return "CRON_TZ=" + tz + " " + spec
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

func (c *Cron) runPush() {
	if !c.push.RunOnce() {
		c.log.Warnw("push skipped: previous run still in progress")
	}
}
