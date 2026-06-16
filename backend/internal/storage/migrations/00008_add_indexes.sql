-- +goose NO TRANSACTION

-- +goose Up

-- cluster_articles: article_id lookup (GetClusterIDsForArticles uses WHERE article_id = ANY(...))
-- PK is (cluster_id, article_id) so article_id alone has no efficient path without this.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cluster_articles_article
    ON cluster_articles(article_id);

-- clusters: composite for topic feed queries (ListClustersForFeedByTopic)
-- Covers: WHERE topic = $1 AND updated_at >= ... ORDER BY score DESC
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_clusters_topic_updated_score
    ON clusters(topic, updated_at DESC, score DESC);

-- clusters: composite for top-N and active queries (ListTopClustersByScore, ListActiveClusters, ListTopCategoryClustersWithArticles)
-- Covers: WHERE last_seen_at >= ... ORDER BY score DESC
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_clusters_last_seen_score
    ON clusters(last_seen_at DESC, score DESC);

-- articles: source_id FK join (used in ListClusterArticlesWithSources, ListLatestArticles, etc.)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_articles_source_id
    ON articles(source_id);

-- articles: partial index for image tiebreaker subqueries (image_url IS NOT NULL sort criterion)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_articles_has_image
    ON articles(id)
    WHERE image_url IS NOT NULL;

-- +goose Down

DROP INDEX CONCURRENTLY IF EXISTS idx_cluster_articles_article;
DROP INDEX CONCURRENTLY IF EXISTS idx_clusters_topic_updated_score;
DROP INDEX CONCURRENTLY IF EXISTS idx_clusters_last_seen_score;
DROP INDEX CONCURRENTLY IF EXISTS idx_articles_source_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_articles_has_image;
