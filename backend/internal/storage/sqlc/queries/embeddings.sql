-- name: UpdateArticleEmbeddingAndStatus :exec
UPDATE articles
SET embedding = $1, embedding_status = $2
WHERE id = $3;

-- name: UpdateArticleEmbeddingStatus :exec
UPDATE articles
SET embedding_status = $1
WHERE id = $2;

-- name: GetArticleEmbedding :one
SELECT embedding
FROM articles
WHERE id = $1
  AND embedding IS NOT NULL;

-- name: MarkProcessingEmbeddingsAsFailed :execresult
UPDATE articles SET embedding_status = 'failed' WHERE embedding_status = 'processing';

-- name: MarkPendingEmbeddingsAsFailed :execresult
UPDATE articles SET embedding_status = 'failed' WHERE embedding_status = 'pending';

-- name: GetArticleEmbeddingsByIDs :many
SELECT id, embedding
FROM articles
WHERE id = ANY($1::bigint[])
  AND embedding IS NOT NULL;
