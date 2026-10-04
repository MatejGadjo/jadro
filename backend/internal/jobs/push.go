package jobs

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/MatejGadjo/jadro/internal/logger"
)

// PushSender is the daily Web Push send, implemented by push.Service.
type PushSender interface {
	SendDaily(ctx context.Context) error
}

// PushJob fires the once-daily notification send on the in-process cron. Like
// the other jobs it carries an overlap guard so a slow run is never doubled up.
type PushJob struct {
	sender  PushSender
	log     *logger.Logger
	running atomic.Bool
}

func NewPushJob(sender PushSender, log *logger.Logger) *PushJob {
	return &PushJob{sender: sender, log: log}
}

// RunOnce fires a single daily send in a new goroutine. Returns false if a run
// is already in progress.
func (j *PushJob) RunOnce() bool {
	if !j.running.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		defer j.running.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := j.sender.SendDaily(ctx); err != nil {
			j.log.Errorw("push: daily send failed", "error", err)
		}
	}()
	return true
}
