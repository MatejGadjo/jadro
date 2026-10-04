package storage

import (
	"context"

	"github.com/MatejGadjo/jadro/domain"
	"github.com/MatejGadjo/jadro/internal/storage/sqlc"
	pgvector "github.com/pgvector/pgvector-go"
)

func (c *Client) GetArticleEmbedding(ctx context.Context, id int64) ([]float32, error) {
	vec, err := c.q.GetArticleEmbedding(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	if vec == nil {
		return nil, nil
	}
	return vec.Slice(), nil
}

func (c *Client) GetArticleEmbeddingsByIDs(ctx context.Context, ids []int64) ([]domain.ArticleEmbedding, error) {
	rows, err := c.q.GetArticleEmbeddingsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ArticleEmbedding, len(rows))
	for i, r := range rows {
		var emb []float32
		if r.Embedding != nil {
			emb = r.Embedding.Slice()
		}
		out[i] = domain.ArticleEmbedding{
			ID:        int64(r.ID),
			Embedding: emb,
		}
	}
	return out, nil
}

func (c *Client) MarkPendingEmbeddingsAsFailed(ctx context.Context) (int64, error) {
	tag, err := c.q.MarkPendingEmbeddingsAsFailed(ctx)
	return tag.RowsAffected(), err
}

func (c *Client) MarkProcessingEmbeddingsAsFailed(ctx context.Context) (int64, error) {
	tag, err := c.q.MarkProcessingEmbeddingsAsFailed(ctx)
	return tag.RowsAffected(), err
}

func (c *Client) UpdateArticleEmbeddingStatus(ctx context.Context, id int64, status domain.EmbeddingStatus) error {
	return c.q.UpdateArticleEmbeddingStatus(ctx, sqlc.UpdateArticleEmbeddingStatusParams{
		ID:              int32(id),
		EmbeddingStatus: sqlc.EmbeddingStatus(status),
	})
}

func (c *Client) UpdateArticleEmbeddingAndStatus(ctx context.Context, id int64, embedding []float32, status domain.EmbeddingStatus) error {
	vec := pgvector.NewVector(embedding)
	return c.q.UpdateArticleEmbeddingAndStatus(ctx, sqlc.UpdateArticleEmbeddingAndStatusParams{
		ID:              int32(id),
		Embedding:       &vec,
		EmbeddingStatus: sqlc.EmbeddingStatus(status),
	})
}
