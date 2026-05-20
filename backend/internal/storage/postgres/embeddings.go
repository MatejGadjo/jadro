package postgres

import (
	"context"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// UpdateArticleEmbeddingAndStatus atomically updates both the embedding and status in a single transaction
func UpdateArticleEmbeddingAndStatus(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, articleID int64, embedding []float32, status string) error {
	if len(embedding) == 0 {
		log.Warnw("attempted to update article with empty embedding", "article_id", articleID)
		return nil
	}

	// Convert []float32 to pgvector.Vector
	vec := pgvector.NewVector(embedding)

	query := `
		UPDATE articles
		SET embedding = $1, embedding_status = $2
		WHERE id = $3
	`

	result, err := pool.Exec(ctx, query, vec, status, articleID)
	if err != nil {
		log.Errorw("failed to update article embedding and status", "article_id", articleID, "status", status, "error", err)
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		log.Warnw("no article found to update embedding and status", "article_id", articleID, "status", status)
		return pgx.ErrNoRows
	}

	return nil
}

// beddingStatus updates the embedding_status column for an article.
func UpdateArticleEmbeddingStatus(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, articleID int64, status string) error {
	query := `
		UPDATE articles
		SET embedding_status = $1
		WHERE id = $2
	`

	result, err := pool.Exec(ctx, query, status, articleID)
	if err != nil {
		log.Errorw("failed to update article embedding status", "article_id", articleID, "status", status, "error", err)
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		log.Warnw("no article found to update embedding status", "article_id", articleID, "status", status)
		return pgx.ErrNoRows
	}

	return nil
}

// GetArticleEmbedding retrieves the embedding for an article
func GetArticleEmbedding(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, articleID int64) ([]float32, error) {
	query := `
		SELECT embedding
		FROM articles
		WHERE id = $1
		AND embedding IS NOT NULL
	`

	var vec pgvector.Vector
	err := pool.QueryRow(ctx, query, articleID).Scan(&vec)
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Warnw("article embedding not found", "article_id", articleID)
			return nil, nil
		}
		log.Errorw("failed to get article embedding", "article_id", articleID, "error", err)
		return nil, err
	}

	// Convert pgvector.Vector to []float32
	embedding := vec.Slice()

	return embedding, nil
}

// MarkStaleEmbeddingsAsFailed resets orphaned embedding_status rows on startup.
// Returns counts of processing and pending rows reset.
func MarkStaleEmbeddingsAsFailed(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger) (processing int64, pending int64, err error) {
	r1, err := pool.Exec(ctx, `UPDATE articles SET embedding_status = 'failed' WHERE embedding_status = 'processing'`)
	if err != nil {
		log.Errorw("failed to reset processing embeddings", "error", err)
		return 0, 0, err
	}
	r2, err := pool.Exec(ctx, `UPDATE articles SET embedding_status = 'failed' WHERE embedding_status = 'pending'`)
	if err != nil {
		log.Errorw("failed to reset pending embeddings", "error", err)
		return 0, 0, err
	}
	return r1.RowsAffected(), r2.RowsAffected(), nil
}

// GetArticleEmbeddingsByIDs retrieves embeddings for a batch of articles.
func GetArticleEmbeddingsByIDs(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, articleIDs []int64) (map[int64][]float32, error) {
	if len(articleIDs) == 0 {
		return map[int64][]float32{}, nil
	}

	query := `
		SELECT id, embedding
		FROM articles
		WHERE id = ANY($1)
		  AND embedding IS NOT NULL
	`

	rows, err := pool.Query(ctx, query, articleIDs)
	if err != nil {
		log.Errorw("failed to query article embeddings", "article_ids", len(articleIDs), "error", err)
		return nil, err
	}
	defer rows.Close()

	embeddings := make(map[int64][]float32, len(articleIDs))
	for rows.Next() {
		var (
			articleID int64
			vec       pgvector.Vector
		)
		if err := rows.Scan(&articleID, &vec); err != nil {
			log.Errorw("failed to scan article embedding row", "error", err)
			return nil, err
		}
		embeddings[articleID] = append([]float32(nil), vec.Slice()...)
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating article embeddings", "error", err)
		return nil, err
	}

	return embeddings, nil
}
