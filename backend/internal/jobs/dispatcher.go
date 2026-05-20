package jobs

import (
	"sync"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
)

// Job represents a unit of work to execute in the dispatcher.
type Job struct {
	Name string
	Run  func() error
}

// Dispatcher is a minimal in-process job queue with a fixed worker pool.
type Dispatcher struct {
	log   *logger.Logger
	queue chan Job
	stop  chan struct{}
	wg    sync.WaitGroup
}

// NewDispatcher constructs a dispatcher with a buffered queue.
func NewDispatcher(log *logger.Logger) *Dispatcher {
	return &Dispatcher{
		log:   log,
		queue: make(chan Job, 2048),
		stop:  make(chan struct{}),
	}
}

// Start spins up workerCount workers to process queued jobs.
func (d *Dispatcher) Start(workerCount int) {
	if workerCount < 1 {
		workerCount = 1
	}
	for i := 0; i < workerCount; i++ {
		workerID := i
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			for {
				select {
				case job, ok := <-d.queue:
				if !ok {
					return
				}
				if job.Run == nil {
					continue
				}
				if err := job.Run(); err != nil {
					d.log.Warnw("job failed", "job", job.Name, "worker_id", workerID, "error", err)
				}
				case <-d.stop:
					return
				}
			}
		}()
	}
}

// Enqueue schedules a job for execution.
func (d *Dispatcher) Enqueue(job Job) {
	d.queue <- job
}

// Stop signals workers to exit and waits for them to finish.
func (d *Dispatcher) Stop() {
	close(d.stop)
	close(d.queue)
	d.wg.Wait()
}
