-- name: CreateCluster :one
INSERT INTO clusters (headline, summary, display_title, topic, score, last_seen_at, article_count, source_count)
VALUES ($1, $2, $3, $4, 0, $5, 1, 1)
RETURNING id;

-- name: InsertClusterSource :exec
INSERT INTO cluster_sources (cluster_id, source_id)
VALUES ($1, $2)
ON CONFLICT (cluster_id, source_id) DO NOTHING;

-- name: GetClusterByID :one
SELECT id, headline, display_title, summary, topic, score, created_at, updated_at,
       last_seen_at, article_count, source_count, articles_last_6h, topic_confidence
FROM clusters
WHERE id = $1;

-- name: GetMaxClusterID :one
SELECT COALESCE(MAX(id), 0)::int AS max_id FROM clusters;

-- name: UpdateClusterScoreValue :exec
UPDATE clusters SET score = $1 WHERE id = $2;

-- name: UpdateClusterDisplayTitle :exec
UPDATE clusters SET display_title = $1 WHERE id = $2;

-- name: ListClustersForFeedAll :many
SELECT id, headline, display_title, summary, topic, score, created_at, updated_at,
       last_seen_at, article_count, source_count, articles_last_6h, topic_confidence
FROM clusters
WHERE updated_at >= NOW() - make_interval(hours => 24)
ORDER BY score DESC,
         (EXISTS (
           SELECT 1 FROM cluster_articles ca
           JOIN articles a ON a.id = ca.article_id
           WHERE ca.cluster_id = clusters.id
             AND a.image_url IS NOT NULL
         )) DESC,
         updated_at DESC
LIMIT $1 OFFSET $2;

-- name: ListClustersForFeedByTopic :many
SELECT id, headline, display_title, summary, topic, score, created_at, updated_at,
       last_seen_at, article_count, source_count, articles_last_6h, topic_confidence
FROM clusters
WHERE updated_at >= NOW() - make_interval(hours => 24)
  AND clusters.topic = $1
ORDER BY score DESC,
         (EXISTS (
           SELECT 1 FROM cluster_articles ca
           JOIN articles a ON a.id = ca.article_id
           WHERE ca.cluster_id = clusters.id
             AND a.image_url IS NOT NULL
         )) DESC,
         updated_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTopClustersByScore :many
SELECT id, headline, display_title, summary, topic, score, created_at, updated_at,
       last_seen_at, article_count, source_count, articles_last_6h, topic_confidence
FROM clusters
WHERE last_seen_at >= NOW() - INTERVAL '48 hours'
ORDER BY score DESC,
         (EXISTS (
           SELECT 1 FROM cluster_articles ca
           JOIN articles a ON a.id = ca.article_id
           WHERE ca.cluster_id = clusters.id
             AND a.image_url IS NOT NULL
         )) DESC,
         updated_at DESC, id ASC
LIMIT $1;

-- name: ListTopCategoryClustersWithArticles :many
WITH ranked AS (
    SELECT c.id, c.headline, c.display_title, c.summary, c.topic, c.score,
           c.created_at, c.updated_at, c.last_seen_at, c.article_count, c.source_count,
           c.articles_last_6h, c.topic_confidence,
           ROW_NUMBER() OVER (
               PARTITION BY c.topic ORDER BY c.score DESC, c.updated_at DESC, c.id ASC
           ) AS rn
    FROM clusters c
    WHERE c.last_seen_at >= NOW() - INTERVAL '48 hours'
      AND c.article_count > 0
      AND c.topic = ANY($1::text[])
),
top_by_topic AS (SELECT * FROM ranked WHERE rn <= $2::int4),
best_article AS (
    SELECT DISTINCT ON (ca.cluster_id)
        ca.cluster_id,
        a.id          AS article_id,
        a.url         AS article_url,
        a.image_url   AS article_image_url,
        a.title       AS article_title,
        a.snippet     AS article_snippet,
        a.published_at,
        a.fetched_at,
        s.id          AS source_id,
        s.name        AS source_name,
        s.base_url    AS source_base_url,
        s.rss_url     AS source_rss_url,
        s.logo_url    AS source_logo_url
    FROM cluster_articles ca
    JOIN articles a ON a.id = ca.article_id
    JOIN sources  s ON s.id  = a.source_id
    ORDER BY ca.cluster_id,
             (a.image_url IS NOT NULL) DESC,
             COALESCE(a.published_at, a.fetched_at) DESC
)
SELECT t.id, t.headline, t.display_title, t.summary, t.topic, t.score,
       t.created_at, t.updated_at, t.last_seen_at, t.article_count, t.source_count,
       t.articles_last_6h, t.topic_confidence,
       ba.article_id, ba.article_url, ba.article_image_url, ba.article_title, ba.article_snippet,
       ba.published_at, ba.fetched_at,
       ba.source_id, ba.source_name, ba.source_base_url, ba.source_rss_url, ba.source_logo_url
FROM top_by_topic t
INNER JOIN best_article ba ON ba.cluster_id = t.id
ORDER BY t.topic, t.score DESC, t.updated_at DESC, t.id ASC;

-- name: ListLatestArticles :many
SELECT a.id, a.url, a.title, a.topic, a.published_at, a.fetched_at,
       s.name AS source_name, s.logo_url AS source_logo_url
FROM (
    SELECT DISTINCT ON (ca.cluster_id)
        a.id,
        COALESCE(a.published_at, a.fetched_at) AS article_time
    FROM articles a
    JOIN cluster_articles ca ON ca.article_id = a.id
    ORDER BY ca.cluster_id, COALESCE(a.published_at, a.fetched_at) DESC
) deduped
JOIN articles a ON a.id = deduped.id
JOIN sources  s ON s.id  = a.source_id
ORDER BY deduped.article_time DESC
LIMIT $1;

-- name: ListActiveClusters :many
SELECT id, headline, display_title, summary, topic, score, created_at, updated_at,
       last_seen_at, article_count, source_count, articles_last_6h, topic_confidence
FROM clusters
WHERE last_seen_at >= NOW() - INTERVAL '48 hours'
ORDER BY id ASC;

-- name: GetClusterIDForArticle :one
SELECT cluster_id
FROM cluster_articles
WHERE article_id = $1
LIMIT 1;

-- name: GetClusterIDsForArticles :many
SELECT article_id, cluster_id
FROM cluster_articles
WHERE article_id = ANY($1::bigint[]);

-- name: AttachArticleToCluster :exec
INSERT INTO cluster_articles (cluster_id, article_id)
VALUES ($1, $2)
ON CONFLICT (cluster_id, article_id) DO NOTHING;

-- name: AttachArticleToClusterWithStats :one
WITH inserted_article AS (
    INSERT INTO cluster_articles (cluster_id, article_id)
    VALUES ($1, $2)
    ON CONFLICT (cluster_id, article_id) DO NOTHING
    RETURNING 1
),
inserted_source AS (
    INSERT INTO cluster_sources (cluster_id, source_id)
    SELECT $1, $3
    WHERE $3 <> 0
      AND EXISTS (SELECT 1 FROM inserted_article)
    ON CONFLICT (cluster_id, source_id) DO NOTHING
    RETURNING 1
)
UPDATE clusters
SET article_count = article_count + (SELECT COUNT(*) FROM inserted_article),
    source_count  = source_count  + (SELECT COUNT(*) FROM inserted_source),
    last_seen_at  = CASE
        WHEN (SELECT COUNT(*) FROM inserted_article) = 0 THEN last_seen_at
        ELSE GREATEST(last_seen_at, $4)
    END,
    updated_at    = CASE
        WHEN (SELECT COUNT(*) FROM inserted_article) = 0 THEN updated_at
        ELSE now()
    END
WHERE id = $1
RETURNING (SELECT COUNT(*) FROM inserted_article);

-- name: GetArticlesInCluster :many
SELECT a.id, a.source_id, a.url, a.image_url, a.title, a.snippet,
       a.normalized_title, a.normalized_snippet, a.rss_categories,
       a.topic, a.published_at, a.fetched_at
FROM articles a
INNER JOIN cluster_articles ca ON a.id = ca.article_id
WHERE ca.cluster_id = $1
ORDER BY a.published_at DESC;

-- name: ListClusterArticlesWithSources :many
SELECT a.id AS article_id, a.url, a.image_url, a.title, a.snippet,
       a.published_at, a.fetched_at,
       s.id AS source_id, s.name, s.base_url, s.rss_url, s.logo_url
FROM cluster_articles ca
INNER JOIN articles a ON a.id = ca.article_id
INNER JOIN sources  s ON s.id  = a.source_id
WHERE ca.cluster_id = $1
ORDER BY (a.image_url IS NOT NULL) DESC, a.published_at DESC NULLS LAST, a.fetched_at DESC, a.id DESC;

-- name: UpdateClusterScoresBatch :exec
UPDATE clusters AS c
SET score = v.score
FROM (SELECT unnest($1::bigint[]) AS id, unnest($2::float8[]) AS score) AS v
WHERE c.id = v.id;

-- name: UpdateClusterVelocityBatch :exec
UPDATE clusters
SET articles_last_6h = (
    SELECT COUNT(*) FROM cluster_articles ca
    JOIN articles a ON a.id = ca.article_id
    WHERE ca.cluster_id = clusters.id
      AND COALESCE(a.published_at, a.fetched_at) >= NOW() - INTERVAL '6 hours'
)
WHERE last_seen_at >= NOW() - INTERVAL '48 hours';

-- name: GetRecentArticleCount :one
SELECT COUNT(*) FROM cluster_articles ca
JOIN articles a ON a.id = ca.article_id
WHERE ca.cluster_id = $1
  AND COALESCE(a.published_at, a.fetched_at) >= NOW() - INTERVAL '6 hours';

-- name: InsertClusterVector :exec
INSERT INTO cluster_vectors (cluster_id, vector, article_count, updated_at)
VALUES ($1, $2, $3, now());

-- name: GetClusterVector :one
SELECT vector, article_count
FROM cluster_vectors
WHERE cluster_id = $1;

-- name: GetClusterVectorForUpdate :one
SELECT vector, article_count
FROM cluster_vectors
WHERE cluster_id = $1
FOR UPDATE;

-- name: UpdateClusterVector :exec
UPDATE cluster_vectors
SET vector        = $2,
    article_count = $3,
    updated_at    = now()
WHERE cluster_id = $1;

-- name: FindSimilarClusterNeighbors :many
SELECT cv.cluster_id, (1 - (cv.vector <=> $1::vector))::float8 AS similarity
FROM cluster_vectors cv
INNER JOIN clusters c ON c.id = cv.cluster_id
WHERE c.last_seen_at >= NOW() - make_interval(hours => $2::int4)
  AND c.created_at >= NOW() - make_interval(hours => $4::int4)
ORDER BY cv.vector <=> $1::vector
LIMIT $3;

-- name: FindNearestArticleSimilarityInClusters :many
-- For each candidate cluster, the similarity of its single closest member
-- article to the query embedding. Used as the actual merge decision instead
-- of the cluster's blended centroid, which drifts as a cluster grows and can
-- end up merging in a story that no individual member closely resembles.
SELECT DISTINCT ON (ca.cluster_id)
    ca.cluster_id,
    (1 - (a.embedding <=> $1::vector))::float8 AS similarity
FROM cluster_articles ca
INNER JOIN articles a ON a.id = ca.article_id
WHERE ca.cluster_id = ANY($2::bigint[])
  AND a.embedding IS NOT NULL
ORDER BY ca.cluster_id, a.embedding <=> $1::vector;

-- name: DeleteOrphanedClusters :execresult
DELETE FROM clusters WHERE id NOT IN (SELECT DISTINCT cluster_id FROM cluster_articles);

-- name: CountClusters :one
SELECT COUNT(*) FROM clusters;
