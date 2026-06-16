-- name: InsertArticle :one
WITH inserted AS (
    INSERT INTO articles (
        source_id, feed_id, feed_url, source_domain, url, image_url,
        title, snippet, normalized_title, normalized_snippet,
        rss_categories, category, category_confidence, category_method,
        category_signals, topic, published_at, fetched_at
    )
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
    ON CONFLICT (url) DO NOTHING
    RETURNING id
)
SELECT id, true AS is_new FROM inserted
UNION ALL
SELECT id, false AS is_new FROM articles WHERE url = $5
LIMIT 1;

-- name: GetArticleByID :one
SELECT id, source_id, feed_id, feed_url, source_domain, url, image_url,
       title, snippet, normalized_title, normalized_snippet, rss_categories,
       category, category_confidence, category_method, category_signals,
       topic, published_at, fetched_at, embedding_status
FROM articles
WHERE id = $1;

-- name: ClearArticleImageURL :execrows
UPDATE articles SET image_url = NULL WHERE id = $1 AND image_url IS NOT NULL;

-- name: DeleteOldArticles :execresult
DELETE FROM articles WHERE COALESCE(published_at, fetched_at) < $1;

-- name: CountArticles :one
SELECT COUNT(*) FROM articles;
