package clustering

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MatejGadjo/NewsAggregator/domain"
	"github.com/MatejGadjo/NewsAggregator/internal/ingestion/normalize"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/jackc/pgx/v5"
)

const (
	maxSnippetTokensForEmbedding = 256
	embeddingGenTimeout          = 5 * time.Second
)

// EmbeddingService generates and persists article embeddings.
type EmbeddingService struct {
	gen   *EmbeddingGenerator
	store EmbeddingStore
	log   *logger.Logger
}

// NewEmbeddingService wires the embedding service dependencies.
func NewEmbeddingService(gen *EmbeddingGenerator, store EmbeddingStore, log *logger.Logger) *EmbeddingService {
	return &EmbeddingService{gen: gen, store: store, log: log}
}

// RecoverStaleEmbeddings resets any processing or pending articles to failed.
// Called once on startup to clear state orphaned by a previous crash.
func (s *EmbeddingService) RecoverStaleEmbeddings(ctx context.Context) {
	processing, err := s.store.MarkProcessingEmbeddingsAsFailed(ctx)
	if err != nil {
		s.log.Errorw("stale embedding recovery failed (processing)", "error", err)
		return
	}
	pending, err := s.store.MarkPendingEmbeddingsAsFailed(ctx)
	if err != nil {
		s.log.Errorw("stale embedding recovery failed (pending)", "error", err)
		return
	}
	s.log.Infow("stale embedding recovery complete",
		"processing_reset", processing,
		"pending_reset", pending)
}

// GenerateAndStore generates an embedding for the given article and persists it.
func (s *EmbeddingService) GenerateAndStore(ctx context.Context, articleID int64) error {
	article, err := s.store.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("article not found: %d", articleID)
		}
		return err
	}

	if err := s.store.UpdateArticleEmbeddingStatus(ctx, articleID, domain.EmbeddingStatusProcessing); err != nil {
		return err
	}

	if s.gen == nil || !s.gen.IsInitialized() {
		_ = s.store.UpdateArticleEmbeddingStatus(ctx, articleID, domain.EmbeddingStatusFailed)
		return fmt.Errorf("embedding generator unavailable for article %d", articleID)
	}

	embeddingText, err := s.buildEmbeddingText(article)
	if err != nil {
		s.log.Warnw("skipping embedding generation: tokenizer unavailable",
			"article_id", articleID, "url", article.URL, "error", err)
		_ = s.store.UpdateArticleEmbeddingStatus(ctx, articleID, domain.EmbeddingStatusFailed)
		return fmt.Errorf("embedding tokenizer unavailable for article %d: %w", articleID, err)
	}
	if embeddingText == "" {
		s.log.Warnw("skipping embedding generation: empty embedding text",
			"article_id", articleID, "url", article.URL,
			"title_len", len(article.Title),
			"normalized_title_len", len(article.NormalizedTitle),
			"snippet_len", len(article.NormalizedSnippet))
		_ = s.store.UpdateArticleEmbeddingStatus(ctx, articleID, domain.EmbeddingStatusFailed)
		return fmt.Errorf("empty embedding text for article %d", articleID)
	}

	embCtx, cancel := context.WithTimeout(ctx, embeddingGenTimeout)
	defer cancel()

	embedding, err := func() (emb []float32, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("embedding generation panic: %v", r)
			}
		}()
		return s.gen.GenerateEmbedding(embCtx, embeddingText)
	}()
	if err != nil {
		_ = s.store.UpdateArticleEmbeddingStatus(ctx, articleID, domain.EmbeddingStatusFailed)
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("embedding generation timeout after 5s")
		}
		return err
	}

	if len(embedding) == 0 {
		_ = s.store.UpdateArticleEmbeddingStatus(ctx, articleID, domain.EmbeddingStatusFailed)
		return fmt.Errorf("empty embedding result for article %d", articleID)
	}

	if err := s.store.UpdateArticleEmbeddingAndStatus(ctx, articleID, embedding, domain.EmbeddingStatusReady); err != nil {
		_ = s.store.UpdateArticleEmbeddingStatus(ctx, articleID, domain.EmbeddingStatusFailed)
		return err
	}

	return nil
}

func (s *EmbeddingService) buildEmbeddingText(article domain.Article) (string, error) {
	if !s.gen.hasTokenizer() {
		return "", fmt.Errorf("embedding tokenizer not initialized")
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
			titleTokens, err := s.gen.tokenCount(embeddingText)
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
			truncated, err := s.gen.truncateTextToTokens(snippetForEmbedding, maxTokens)
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
