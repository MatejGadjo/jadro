package postgres

import (
	"context"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InsertArticle inserts a new article or returns the existing article ID if the URL already exists.
// The snippet must already be normalized (no HTML).
// It returns the article ID and whether a new row was inserted.
func InsertArticle(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, article storage.Article) (int64, bool, error) {
	category := article.Category
	if category == "" {
		category = article.Topic
	}

	categoryMethod := article.CategoryMethod
	if categoryMethod == "" {
		categoryMethod = "fallback"
	}

	categorySignals := article.CategorySignals
	if len(categorySignals) == 0 {
		categorySignals = []byte("{}")
	}

	categoryConfidence := article.CategoryConfidence
	if categoryConfidence < 0 || categoryConfidence > 1 {
		categoryConfidence = 0.50
	}

	query := `
		WITH inserted AS (
			INSERT INTO articles (
				source_id,
				feed_id,
				feed_url,
				source_domain,
				url,
				image_url,
				title,
				snippet,
				normalized_title,
				normalized_snippet,
				rss_categories,
				category,
				category_confidence,
				category_method,
				category_signals,
				topic,
				published_at,
				fetched_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
			ON CONFLICT (url) DO NOTHING
			RETURNING id
		),
		updated AS (
			UPDATE articles
			SET image_url = $6
			WHERE url = $5
			  AND image_url IS NULL
			  AND $6 IS NOT NULL
			RETURNING id
		)
		SELECT id, true AS is_new
		FROM inserted
		UNION ALL
		SELECT id, false AS is_new
		FROM updated
		UNION ALL
		SELECT id, false AS is_new
		FROM articles
		WHERE url = $5
		LIMIT 1
	`

	var id int64
	var isNew bool
	err := pool.QueryRow(ctx, query,
		article.SourceID,
		article.FeedID,
		article.FeedURL,
		article.SourceDomain,
		article.URL,
		article.ImageURL,
		article.Title,
		article.Snippet,
		article.NormalizedTitle,
		article.NormalizedSnippet,
		article.RSSCategories,
		category,
		categoryConfidence,
		categoryMethod,
		categorySignals,
		article.Topic,
		article.PublishedAt,
		article.FetchedAt,
	).Scan(&id, &isNew)

	if err != nil {
		log.Errorw("failed to insert article", "url", article.URL, "source_id", article.SourceID, "error", err)
		return 0, false, err
	}

	return id, isNew, nil
}

// ClearArticleImageURL sets image_url to NULL for the given article.
// Called when a frontend onError event confirms the stored URL returns a broken image.
func ClearArticleImageURL(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, id int64) error {
	query := `UPDATE articles SET image_url = NULL WHERE id = $1`

	result, err := pool.Exec(ctx, query, id)
	if err != nil {
		log.Errorw("failed to clear article image url", "article_id", id, "error", err)
		return err
	}

	if result.RowsAffected() == 0 {
		log.Warnw("no article found to clear image", "article_id", id)
		return pgx.ErrNoRows
	}

	return nil
}

// GetArticleByID retrieves an article by its ID.
// Returns nil, nil if the article is not found.
func GetArticleByID(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, id int64) (*storage.Article, error) {
	query := `
		SELECT id, source_id, feed_id, feed_url, source_domain, url, image_url, title, snippet, normalized_title, normalized_snippet, rss_categories, category, category_confidence, category_method, category_signals, topic, published_at, fetched_at, embedding_status
		FROM articles
		WHERE id = $1
	`

	var a storage.Article
	err := pool.QueryRow(ctx, query, id).Scan(
		&a.ID,
		&a.SourceID,
		&a.FeedID,
		&a.FeedURL,
		&a.SourceDomain,
		&a.URL,
		&a.ImageURL,
		&a.Title,
		&a.Snippet,
		&a.NormalizedTitle,
		&a.NormalizedSnippet,
		&a.RSSCategories,
		&a.Category,
		&a.CategoryConfidence,
		&a.CategoryMethod,
		&a.CategorySignals,
		&a.Topic,
		&a.PublishedAt,
		&a.FetchedAt,
		&a.EmbeddingStatus,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			log.Warnw("article not found", "article_id", id)
			return nil, nil
		}
		log.Errorw("failed to get article by id", "article_id", id, "error", err)
		return nil, err
	}

	return &a, nil
}
