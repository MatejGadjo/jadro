package jobs

import (
	"context"

	"github.com/alitto/pond/v2"
	"go.uber.org/fx"
)

// Pipeline decouples ingestion, embedding generation, and cluster assignment stages.
type Pipeline struct {
	ctx      context.Context
	pool     pond.Pool
	embedder Embedder
	assigner ClusterAssigner
}

// NewPipeline wires the pipeline dependencies and registers a lifecycle hook to recover stale embeddings on startup.
func NewPipeline(lc fx.Lifecycle, pool pond.Pool, embedder Embedder, assigner ClusterAssigner) *Pipeline {
	p := &Pipeline{
		ctx:      context.Background(),
		pool:     pool,
		embedder: embedder,
		assigner: assigner,
	}
	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			p.embedder.RecoverStaleEmbeddings(p.ctx)
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
func (p *Pipeline) EmbedArticle(articleID int64) error {
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
