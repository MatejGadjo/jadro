package jobs

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/robfig/cron/v3"
)

// IngestFunc is called by the scheduler each tick to fetch and process new articles.
// Returns the number of new articles stored.
type IngestFunc func(ctx context.Context) (int, error)

// Scheduler periodically runs RSS ingestion followed by a cluster rescore.
type Scheduler struct {
	log             *logger.Logger
	ingest          IngestFunc
	rescore         *RescoreJob
	cron            *cron.Cron
	running         atomic.Bool
	lastIngestAt    atomic.Value // stores time.Time
	postIngestHook  func(ctx context.Context)
}

// SetPostIngestHook registers a function called after each successful ingest+rescore cycle.
func (s *Scheduler) SetPostIngestHook(fn func(ctx context.Context)) {
	s.postIngestHook = fn
}

// LastIngestAt returns the time of the last completed ingest run, or nil if none has run yet.
func (s *Scheduler) LastIngestAt() *time.Time {
	v := s.lastIngestAt.Load()
	if v == nil {
		return nil
	}
	t := v.(time.Time)
	return &t
}

func NewScheduler(log *logger.Logger, ingest IngestFunc, rescore *RescoreJob) *Scheduler {
	return &Scheduler{
		log:     log,
		ingest:  ingest,
		rescore: rescore,
		cron:    cron.New(),
	}
}

// Start registers the cron expression, starts the scheduler, and fires one run immediately.
func (s *Scheduler) Start(cronExpr string) error {
	if _, err := s.cron.AddFunc(cronExpr, s.run); err != nil {
		return err
	}
	s.cron.Start()
	go s.run()
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// RunOnce fires a single ingest+rescore cycle in a new goroutine.
// Returns false if a run is already in progress (caller should respond 409).
func (s *Scheduler) RunOnce() bool {
	if s.running.Load() {
		return false
	}
	go s.run()
	return true
}

func (s *Scheduler) run() {
	if !s.running.CompareAndSwap(false, true) {
		s.log.Warnw("ingest skipped: previous run still in progress")
		return
	}
	defer s.running.Store(false)

	ctx := context.Background()
	s.log.Infow("ingest run started")

	count, err := s.ingest(ctx)
	if err != nil {
		s.log.Errorw("ingest run failed", "error", err)
		return
	}

	s.lastIngestAt.Store(time.Now())

	if err := s.rescore.Run(ctx); err != nil {
		s.log.Errorw("rescore failed", "error", err)
	}

	if s.postIngestHook != nil {
		s.postIngestHook(ctx)
	}

	s.log.Infow("ingest run completed", "new_articles", count)
}
