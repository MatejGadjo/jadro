package jobs

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/MatejGadjo/jadro/internal/logger"
	"github.com/alitto/pond/v2"
	"go.uber.org/fx"
)

const (
	// stallCheckInterval is how often the watchdog samples embedding progress.
	stallCheckInterval = time.Minute
	// stallThreshold is how long embeds may be in-flight with zero completions
	// before we treat the pipeline as stalled and dump goroutine stacks. Sized
	// well above the 5s per-embed timeout so it only trips on a real hang.
	stallThreshold = 3 * time.Minute
)

// Pipeline decouples ingestion, embedding generation, and cluster assignment stages.
type Pipeline struct {
	ctx      context.Context
	pool     pond.Pool
	embedder Embedder
	assigner ClusterAssigner
	log      *logger.Logger

	// Watchdog counters (atomic). A deadlock inside embedding leaves inFlight > 0
	// while lastActivityNano stops advancing — the signature we detect.
	inFlight         int64
	lastActivityNano int64
	stallReported    int32
}

// NewPipeline wires the pipeline dependencies and registers a lifecycle hook to
// recover stale embeddings on startup and start the stall watchdog.
func NewPipeline(lc fx.Lifecycle, pool pond.Pool, embedder Embedder, assigner ClusterAssigner, log *logger.Logger) *Pipeline {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pipeline{
		ctx:              ctx,
		pool:             pool,
		embedder:         embedder,
		assigner:         assigner,
		log:              log,
		lastActivityNano: time.Now().UnixNano(),
	}
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			p.embedder.RecoverStaleEmbeddings(p.ctx)
			go p.watchStalls()
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
	return p
}

// EnqueueEmbed schedules an embedding job for an article.
func (p *Pipeline) EnqueueEmbed(articleID int64) {
	p.pool.Submit(func() {
		if err := p.EmbedArticle(articleID); err != nil {
			// error already logged inside EmbedArticle via embedder
			_ = err
		}
	})
}

// EnqueueAssignCluster schedules a clustering job for an article.
func (p *Pipeline) EnqueueAssignCluster(articleID int64) {
	p.pool.Submit(func() {
		if err := p.AssignCluster(articleID); err != nil {
			_ = err
		}
	})
}

// EmbedArticle generates and stores an embedding, then enqueues cluster assignment.
// It brackets the embed with watchdog counters so a hang inside GenerateAndStore
// (e.g. a deadlocked ONNX mutex) is observable: inFlight stays raised while
// lastActivityNano never advances.
func (p *Pipeline) EmbedArticle(articleID int64) error {
	atomic.AddInt64(&p.inFlight, 1)
	defer func() {
		atomic.AddInt64(&p.inFlight, -1)
		atomic.StoreInt64(&p.lastActivityNano, time.Now().UnixNano())
		atomic.StoreInt32(&p.stallReported, 0)
	}()

	if err := p.embedder.GenerateAndStore(p.ctx, articleID); err != nil {
		return err
	}
	p.EnqueueAssignCluster(articleID)
	return nil
}

// AssignCluster assigns an article with a ready embedding to a cluster.
func (p *Pipeline) AssignCluster(articleID int64) error {
	return p.assigner.AssignArticle(p.ctx, articleID)
}

// watchStalls periodically checks whether embedding work is in-flight yet no
// embed has completed within stallThreshold. When that happens it logs an error
// with a full goroutine dump (once per stall episode), so a silent deadlock
// self-documents instead of requiring a live SIGQUIT.
func (p *Pipeline) watchStalls() {
	ticker := time.NewTicker(stallCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			inFlight := atomic.LoadInt64(&p.inFlight)
			idle := time.Since(time.Unix(0, atomic.LoadInt64(&p.lastActivityNano)))
			if inFlight <= 0 || idle < stallThreshold {
				continue
			}
			if !atomic.CompareAndSwapInt32(&p.stallReported, 0, 1) {
				continue // already reported this episode
			}
			buf := make([]byte, 1<<20)
			n := runtime.Stack(buf, true)
			p.log.Errorw("embedding pipeline stall detected: workers in-flight but no completions",
				"in_flight", inFlight,
				"idle", idle.String(),
				"goroutine_dump", string(buf[:n]))
		}
	}
}
