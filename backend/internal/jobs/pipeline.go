package jobs

import (
	"context"
	"fmt"
	"errors"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/clustering"
	"github.com/MatejGadjo/NewsAggregator/internal/ingestion/normalize"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/storage"
	"github.com/MatejGadjo/NewsAggregator/internal/storage/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxSnippetTokensForEmbedding = 256
)

// Pipeline decouples ingestion, embedding generation, and cluster assignment stages.
type Pipeline struct {
	ctx          context.Context
	pool         *pgxpool.Pool
	log          *logger.Logger
	embeddingGen *clustering.EmbeddingGenerator
	dispatcher   *Dispatcher
}

// NewPipeline wires the pipeline dependencies.
func NewPipeline(pool *pgxpool.Pool, log *logger.Logger, embeddingGen *clustering.EmbeddingGenerator, dispatcher *Dispatcher) *Pipeline {
	return &Pipeline{
		ctx:          context.Background(),
		pool:         pool,
		log:          log,
		embeddingGen: embeddingGen,
		dispatcher:   dispatcher,
	}
}

// RecoverStaleEmbeddings resets any processing or pending articles to failed on startup.
// Called once after the dispatcher starts to clear state orphaned by a previous crash.
func (p *Pipeline) RecoverStaleEmbeddings() {
	processing, pending, err := postgres.MarkStaleEmbeddingsAsFailed(p.ctx, p.pool, p.log)
	if err != nil {
		p.log.Errorw("stale embedding recovery failed", "error", err)
		return
	}
	p.log.Infow("stale embedding recovery complete", "processing_reset", processing, "pending_reset", pending)
}

// EnqueueEmbed schedules an embedding job for an article.
func (p *Pipeline) EnqueueEmbed(articleID int64) {
	p.dispatcher.Enqueue(Job{
		Name: "embed_article",
		Run: func() error {
			return p.EmbedArticle(articleID)
		},
	})
}

// EnqueueAssignCluster schedules a clustering job for an article.
func (p *Pipeline) EnqueueAssignCluster(articleID int64) {
	p.dispatcher.Enqueue(Job{
		Name: "assign_cluster",
		Run: func() error {
			return p.AssignCluster(articleID)
		},
	})
}

// EmbedArticle generates and stores an embedding for the given article.
func (p *Pipeline) EmbedArticle(articleID int64) error {
	article, err := postgres.GetArticleByID(p.ctx, p.pool, p.log, articleID)
	if err != nil {
		return err
	}
	if article == nil {
		return fmt.Errorf("article not found: %d", articleID)
	}

	if err := postgres.UpdateArticleEmbeddingStatus(p.ctx, p.pool, p.log, articleID, "processing"); err != nil {
		return err
	}

	if p.embeddingGen == nil || !p.embeddingGen.IsInitialized() {
		_ = postgres.UpdateArticleEmbeddingStatus(p.ctx, p.pool, p.log, articleID, "failed")
		return fmt.Errorf("embedding generator unavailable for article %d", articleID)
	}

	embeddingText, err := buildEmbeddingText(article, p.embeddingGen)
	if err != nil {
		p.log.Warnw("skipping embedding generation: tokenizer unavailable",
			"article_id", articleID,
			"url", article.URL,
			"error", err)
		_ = postgres.UpdateArticleEmbeddingStatus(p.ctx, p.pool, p.log, articleID, "failed")
		return fmt.Errorf("embedding tokenizer unavailable for article %d: %w", articleID, err)
	}
	if embeddingText == "" {
		p.log.Warnw("skipping embedding generation: empty embedding text",
			"article_id", articleID,
			"url", article.URL,
			"title_len", len(article.Title),
			"normalized_title_len", len(article.NormalizedTitle),
			"snippet_len", len(article.NormalizedSnippet))
		_ = postgres.UpdateArticleEmbeddingStatus(p.ctx, p.pool, p.log, articleID, "failed")
		return fmt.Errorf("empty embedding text for article %d", articleID)
	}

	embCtx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	embedding, err := func() ([]float32, error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("embedding generation panic: %v", r)
			}
		}()
		return p.embeddingGen.GenerateEmbedding(embCtx, embeddingText)
	}()
	if err != nil {
		_ = postgres.UpdateArticleEmbeddingStatus(p.ctx, p.pool, p.log, articleID, "failed")
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("embedding generation timeout after 5s")
		}
		return err
	}

	if len(embedding) == 0 {
		_ = postgres.UpdateArticleEmbeddingStatus(p.ctx, p.pool, p.log, articleID, "failed")
		return fmt.Errorf("empty embedding result for article %d", articleID)
	}

	// Atomically update both embedding and status in a single SQL statement
	if err := postgres.UpdateArticleEmbeddingAndStatus(p.ctx, p.pool, p.log, articleID, embedding, "ready"); err != nil {
		_ = postgres.UpdateArticleEmbeddingStatus(p.ctx, p.pool, p.log, articleID, "failed")
		return err
	}

	p.EnqueueAssignCluster(articleID)
	return nil
}

// AssignCluster assigns an article with a ready embedding to a cluster.
func (p *Pipeline) AssignCluster(articleID int64) error {
	article, err := postgres.GetArticleByID(p.ctx, p.pool, p.log, articleID)
	if err != nil {
		return err
	}
	if article == nil {
		return fmt.Errorf("article not found: %d", articleID)
	}
	if article.EmbeddingStatus != "ready" {
		return fmt.Errorf("embedding status not ready: article %d status %s", articleID, article.EmbeddingStatus)
	}

	embedding, err := postgres.GetArticleEmbedding(p.ctx, p.pool, p.log, articleID)
	if err != nil {
		return err
	}
	if len(embedding) == 0 {
		return clustering.ErrMissingEmbedding
	}

	_, _, err = clustering.AssignArticleToClusterIncremental(p.ctx, p.pool, p.log, *article, embedding)
	return err
}

func buildEmbeddingText(article *storage.Article, embeddingGen *clustering.EmbeddingGenerator) (string, error) {
	if embeddingGen == nil {
		return "", fmt.Errorf("embedding generator not initialized")
	}
	if _, err := embeddingGen.TokenCount("tokenizer_check"); err != nil {
		return "", err
	}

	embeddingText := article.NormalizedTitle
	if embeddingText == "" {
		normalizedTitleOnly, _ := normalize.NormalizeArticle(article.Title, "")
		embeddingText = normalizedTitleOnly
	}
	if embeddingText == "" {
		embeddingText = article.Title
	}

	if article.NormalizedSnippet != "" {
		snippetForEmbedding := article.NormalizedSnippet
		maxTokens := maxSnippetTokensForEmbedding
		if embeddingText != "" {
			titleTokens, err := embeddingGen.TokenCount(embeddingText)
			if err != nil {
				return "", err
			}
			if titleTokens < maxSnippetTokensForEmbedding {
				remaining := maxSnippetTokensForEmbedding - titleTokens
				if remaining < maxTokens {
					maxTokens = remaining
				}
			}
		}
		if maxTokens > 0 {
			truncated, err := embeddingGen.TruncateTextToTokens(snippetForEmbedding, maxTokens)
			if err != nil {
				return "", err
			}
			snippetForEmbedding = truncated
		} else {
			snippetForEmbedding = ""
		}

		if embeddingText == "" {
			embeddingText = snippetForEmbedding
		} else {
			embeddingText = embeddingText + " " + snippetForEmbedding
		}
	}

	return embeddingText, nil
}
