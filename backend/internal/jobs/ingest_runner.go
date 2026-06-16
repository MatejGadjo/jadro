package jobs

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
)

// IngestFunc is called by IngestRunner to fetch and process new articles.
// Returns the number of new articles stored.
type IngestFunc func(ctx context.Context) (int, error)

// IngestRunner executes a single ingest+rescore cycle on demand.
// It is triggered on a schedule by the in-process cron (see cron.go).
type IngestRunner struct {
	log          *logger.Logger
	ingest       IngestFunc
	rescore      *RescoreJob
	running      atomic.Bool
	lastIngestAt atomic.Value // stores time.Time
}

// LastIngestAt returns the time of the last completed ingest run, or nil if none has run yet.
func (r *IngestRunner) LastIngestAt() *time.Time {
	v := r.lastIngestAt.Load()
	if v == nil {
		return nil
	}
	t := v.(time.Time)
	return &t
}

func NewIngestRunner(log *logger.Logger, ingest IngestFunc, rescore *RescoreJob) *IngestRunner {
	return &IngestRunner{
		log:     log,
		ingest:  ingest,
		rescore: rescore,
	}
}

// RunOnce fires a single ingest+rescore cycle in a new goroutine.
// Returns false if a run is already in progress (caller should respond 409).
// after, if non-nil, is called with a background context once the run completes successfully.
func (r *IngestRunner) RunOnce(after func(context.Context)) bool {
	if !r.running.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		defer r.running.Store(false)
		r.run()
		if after != nil {
			after(context.Background())
		}
	}()
	return true
}

func (r *IngestRunner) run() {
	ctx := context.Background()
	r.log.Infow("ingest run started")

	count, err := r.ingest(ctx)
	if err != nil {
		r.log.Errorw("ingest run failed", "error", err)
		return
	}

	r.lastIngestAt.Store(time.Now())

	if err := r.rescore.Run(ctx); err != nil {
		r.log.Errorw("rescore failed", "error", err)
	}

	r.log.Infow("ingest run completed", "new_articles", count)
}
