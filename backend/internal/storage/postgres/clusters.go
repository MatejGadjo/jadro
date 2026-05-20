package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateClusterWithStats inserts a new cluster initialized with stats for incremental clustering.
func CreateClusterWithStats(
	ctx context.Context,
	pool *pgxpool.Pool,
	log *logger.Logger,
	headline, summary, displayTitle, topic string,
	lastSeenAt time.Time,
	sourceID int64,
) (int64, error) {
	query := `
		INSERT INTO clusters (
			headline, summary, display_title, topic, score,
			last_seen_at, article_count, source_count
		)
		VALUES ($1, $2, $3, $4, 0, $5, 1, 1)
		RETURNING id
	`

	var id int64
	err := pool.QueryRow(ctx, query, headline, summary, displayTitle, topic, lastSeenAt).Scan(&id)
	if err != nil {
		log.Errorw("failed to create cluster with stats", "headline", headline, "topic", topic, "error", err)
		return 0, err
	}

	if sourceID == 0 {
		return id, nil
	}

	if err := insertClusterSource(ctx, pool, log, id, sourceID); err != nil {
		return 0, err
	}

	return id, nil
}

// UpdateClusterScoreValue updates the score without touching the updated_at timestamp.
func UpdateClusterScoreValue(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, clusterID int64, newScore float64) error {
	query := `
		UPDATE clusters
		SET score = $1
		WHERE id = $2
	`

	result, err := pool.Exec(ctx, query, newScore, clusterID)
	if err != nil {
		log.Errorw("failed to update cluster score", "cluster_id", clusterID, "score", newScore, "error", err)
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		log.Warnw("no cluster found to update", "cluster_id", clusterID)
		return pgx.ErrNoRows
	}

	return nil
}

// UpdateClusterDisplayTitle updates the cluster display title for feed/card rendering.
func UpdateClusterDisplayTitle(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, clusterID int64, displayTitle string) error {
	query := `
		UPDATE clusters
		SET display_title = $1
		WHERE id = $2
	`

	result, err := pool.Exec(ctx, query, strings.TrimSpace(displayTitle), clusterID)
	if err != nil {
		log.Errorw("failed to update cluster display title", "cluster_id", clusterID, "error", err)
		return err
	}

	if result.RowsAffected() == 0 {
		log.Warnw("no cluster found to update display title", "cluster_id", clusterID)
		return pgx.ErrNoRows
	}

	return nil
}

// ListClustersForFeed returns recent clusters for the main feed ordered by score DESC, updated_at DESC.
// Topic is an optional filter. Offset is used for pagination.
func ListClustersForFeed(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, limit, offset int, topic *string) ([]storage.Cluster, error) {
	const homeFeedHours = 24

	var query string
	var args []interface{}

	if topic != nil {
		query = `
			SELECT id, headline, display_title, summary, topic, score, created_at, updated_at,
			       last_seen_at, article_count, source_count, articles_last_6h, topic_confidence
			FROM clusters
			WHERE updated_at >= NOW() - make_interval(hours => $1)
			  AND topic = $2
			ORDER BY score DESC,
			         (EXISTS (
			           SELECT 1 FROM cluster_articles ca
			           JOIN articles a ON a.id = ca.article_id
			           WHERE ca.cluster_id = clusters.id
			             AND a.image_url IS NOT NULL
			         )) DESC,
			         updated_at DESC
			LIMIT $3 OFFSET $4
		`
		args = []interface{}{homeFeedHours, *topic, limit, offset}
	} else {
		query = `
			SELECT id, headline, display_title, summary, topic, score, created_at, updated_at,
			       last_seen_at, article_count, source_count, articles_last_6h, topic_confidence
			FROM clusters
			WHERE updated_at >= NOW() - make_interval(hours => $1)
			ORDER BY score DESC,
			         (EXISTS (
			           SELECT 1 FROM cluster_articles ca
			           JOIN articles a ON a.id = ca.article_id
			           WHERE ca.cluster_id = clusters.id
			             AND a.image_url IS NOT NULL
			         )) DESC,
			         updated_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{homeFeedHours, limit, offset}
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		log.Errorw("failed to query clusters for feed", "limit", limit, "error", err)
		return nil, err
	}
	defer rows.Close()

	var clusters []storage.Cluster
	for rows.Next() {
		var c storage.Cluster
		if err := rows.Scan(
			&c.ID,
			&c.Headline,
			&c.DisplayTitle,
			&c.Summary,
			&c.Topic,
			&c.Score,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.LastSeenAt,
			&c.ArticleCount,
			&c.SourceCount,
			&c.ArticlesLast6h,
			&c.TopicConfidence,
		); err != nil {
			log.Errorw("failed to scan cluster row", "error", err)
			return nil, err
		}
		clusters = append(clusters, c)
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating cluster rows", "error", err)
		return nil, err
	}

	return clusters, nil
}

// ListTopClustersByScore returns clusters updated in the last 48 hours ordered by score DESC for homepage sections.
func ListTopClustersByScore(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, limit int) ([]storage.Cluster, error) {
	query := `
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
		LIMIT $1
	`

	rows, err := pool.Query(ctx, query, limit)
	if err != nil {
		log.Errorw("failed to query top clusters by score", "limit", limit, "error", err)
		return nil, err
	}
	defer rows.Close()

	var clusters []storage.Cluster
	for rows.Next() {
		var c storage.Cluster
		if err := rows.Scan(
			&c.ID,
			&c.Headline,
			&c.DisplayTitle,
			&c.Summary,
			&c.Topic,
			&c.Score,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.LastSeenAt,
			&c.ArticleCount,
			&c.SourceCount,
			&c.ArticlesLast6h,
			&c.TopicConfidence,
		); err != nil {
			log.Errorw("failed to scan cluster row", "error", err)
			return nil, err
		}
		clusters = append(clusters, c)
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating cluster rows", "error", err)
		return nil, err
	}

	return clusters, nil
}

// ListTopCategoryClustersWithArticles returns the top perTopicLimit clusters per topic for the given topics,
// each paired with a single representative article (image preferred, then most recent).
func ListTopCategoryClustersWithArticles(
	ctx context.Context,
	pool *pgxpool.Pool,
	log *logger.Logger,
	topicsFilter []string,
	perTopicLimit int,
) ([]storage.ClusterWithRepresentativeArticle, error) {
	if len(topicsFilter) == 0 {
		return nil, nil
	}

	query := `
		WITH cluster_has_image AS (
			SELECT DISTINCT ca.cluster_id
			FROM cluster_articles ca
			JOIN articles a ON a.id = ca.article_id
			WHERE a.image_url IS NOT NULL
		),
		ranked AS (
			SELECT c.id, c.headline, c.display_title, c.summary, c.topic, c.score,
			       c.created_at, c.updated_at, c.last_seen_at, c.article_count, c.source_count,
			       c.articles_last_6h, c.topic_confidence,
			       ROW_NUMBER() OVER (
			         PARTITION BY c.topic
			         ORDER BY c.score DESC, (chi.cluster_id IS NOT NULL) DESC, c.updated_at DESC, c.id ASC
			       ) AS rn
			FROM clusters c
			LEFT JOIN cluster_has_image chi ON chi.cluster_id = c.id
			WHERE c.last_seen_at >= NOW() - INTERVAL '48 hours'
			  AND c.article_count > 0
			  AND c.topic = ANY($1)
		),
		top_by_topic AS (
			SELECT * FROM ranked WHERE rn <= $2
		),
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
				to_jsonb(s)->>'logo_url' AS source_logo_url
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
		ORDER BY t.topic, t.score DESC, t.updated_at DESC, t.id ASC
	`

	rows, err := pool.Query(ctx, query, topicsFilter, perTopicLimit)
	if err != nil {
		log.Errorw("failed to query top category clusters with articles", "error", err)
		return nil, err
	}
	defer rows.Close()

	var out []storage.ClusterWithRepresentativeArticle
	for rows.Next() {
		var r storage.ClusterWithRepresentativeArticle
		if err := rows.Scan(
			&r.ID,
			&r.Headline,
			&r.DisplayTitle,
			&r.Summary,
			&r.Topic,
			&r.Score,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.LastSeenAt,
			&r.ArticleCount,
			&r.SourceCount,
			&r.ArticlesLast6h,
			&r.TopicConfidence,
			&r.ArticleID,
			&r.ArticleURL,
			&r.ArticleImageURL,
			&r.ArticleTitle,
			&r.ArticleSnippet,
			&r.PublishedAt,
			&r.FetchedAt,
			&r.SourceID,
			&r.SourceName,
			&r.SourceBaseURL,
			&r.SourceRSSURL,
			&r.SourceLogoURL,
		); err != nil {
			log.Errorw("failed to scan cluster with representative article", "error", err)
			return nil, err
		}
		out = append(out, r)
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating cluster with representative article rows", "error", err)
		return nil, err
	}

	return out, nil
}

// ListLatestArticles returns the most recently published article per cluster,
// ordered by recency. Used by the "Свежо" sidebar feed.
func ListLatestArticles(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, limit int) ([]storage.LatestArticle, error) {
	query := `
		SELECT
			a.id,
			a.url,
			a.title,
			a.topic,
			a.published_at,
			a.fetched_at,
			s.name       AS source_name,
			s.logo_url   AS source_logo_url
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
		LIMIT $1
	`

	rows, err := pool.Query(ctx, query, limit)
	if err != nil {
		log.Errorw("failed to query latest articles", "limit", limit, "error", err)
		return nil, err
	}
	defer rows.Close()

	var articles []storage.LatestArticle
	for rows.Next() {
		var a storage.LatestArticle
		if err := rows.Scan(
			&a.ID,
			&a.URL,
			&a.Title,
			&a.Topic,
			&a.PublishedAt,
			&a.FetchedAt,
			&a.SourceName,
			&a.SourceLogoURL,
		); err != nil {
			log.Errorw("failed to scan latest article row", "error", err)
			return nil, err
		}
		articles = append(articles, a)
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating latest article rows", "error", err)
		return nil, err
	}

	return articles, nil
}

// GetClusterByID retrieves a cluster by its ID.
// Returns nil, nil if the cluster is not found.
func GetClusterByID(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, clusterID int64) (*storage.Cluster, error) {
	query := `
		SELECT id, headline, display_title, summary, topic, score, created_at, updated_at,
		       last_seen_at, article_count, source_count, articles_last_6h, topic_confidence
		FROM clusters
		WHERE id = $1
	`

	var c storage.Cluster
	err := pool.QueryRow(ctx, query, clusterID).Scan(
		&c.ID,
		&c.Headline,
		&c.DisplayTitle,
		&c.Summary,
		&c.Topic,
		&c.Score,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.LastSeenAt,
		&c.ArticleCount,
		&c.SourceCount,
		&c.ArticlesLast6h,
		&c.TopicConfidence,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			log.Warnw("cluster not found", "cluster_id", clusterID)
			return nil, nil
		}
		log.Errorw("failed to get cluster by id", "cluster_id", clusterID, "error", err)
		return nil, err
	}

	return &c, nil
}

// GetClusterIDsForArticles returns cluster IDs for the provided article IDs.
func GetClusterIDsForArticles(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, articleIDs []int64) (map[int64]int64, error) {
	if len(articleIDs) == 0 {
		return map[int64]int64{}, nil
	}

	query := `
		SELECT article_id, cluster_id
		FROM cluster_articles
		WHERE article_id = ANY($1)
	`

	rows, err := pool.Query(ctx, query, articleIDs)
	if err != nil {
		log.Errorw("failed to query cluster ids for articles", "error", err)
		return nil, err
	}
	defer rows.Close()

	ids := make(map[int64]int64, len(articleIDs))
	for rows.Next() {
		var articleID int64
		var clusterID int64
		if err := rows.Scan(&articleID, &clusterID); err != nil {
			log.Errorw("failed to scan cluster article row", "error", err)
			return nil, err
		}
		if _, exists := ids[articleID]; !exists {
			ids[articleID] = clusterID
		}
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating cluster article rows", "error", err)
		return nil, err
	}

	return ids, nil
}

// AttachArticleToCluster attaches an article to a cluster.
// It uses INSERT ... ON CONFLICT to handle duplicate attachments gracefully.
func AttachArticleToCluster(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, clusterID, articleID int64) error {
	query := `
		INSERT INTO cluster_articles (cluster_id, article_id)
		VALUES ($1, $2)
		ON CONFLICT (cluster_id, article_id) DO NOTHING
	`

	_, err := pool.Exec(ctx, query, clusterID, articleID)
	if err != nil {
		log.Errorw("failed to attach article to cluster", "cluster_id", clusterID, "article_id", articleID, "error", err)
		return err
	}

	return nil
}

// AttachArticleToClusterWithStatsTx attaches an article and updates cluster stats incrementally.
func AttachArticleToClusterWithStatsTx(
	ctx context.Context,
	tx pgx.Tx,
	log *logger.Logger,
	clusterID, articleID, sourceID int64,
	publishedAt time.Time,
) (bool, error) {
	query := `
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
		    source_count = source_count + (SELECT COUNT(*) FROM inserted_source),
		    last_seen_at = CASE
		        WHEN (SELECT COUNT(*) FROM inserted_article) = 0 THEN last_seen_at
		        ELSE GREATEST(last_seen_at, $4)
		    END,
		    updated_at = CASE
		        WHEN (SELECT COUNT(*) FROM inserted_article) = 0 THEN updated_at
		        ELSE now()
		    END
		WHERE id = $1
		RETURNING (SELECT COUNT(*) FROM inserted_article)
	`

	var insertedCount int
	if err := tx.QueryRow(ctx, query, clusterID, articleID, sourceID, publishedAt).Scan(&insertedCount); err != nil {
		log.Errorw("failed to attach article to cluster with stats", "cluster_id", clusterID, "article_id", articleID, "error", err)
		if err == pgx.ErrNoRows {
			log.Warnw("no cluster found to update stats", "cluster_id", clusterID)
		}
		return false, err
	}

	return insertedCount > 0, nil
}

func insertClusterSource(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, clusterID, sourceID int64) error {
	query := `
		INSERT INTO cluster_sources (cluster_id, source_id)
		VALUES ($1, $2)
		ON CONFLICT (cluster_id, source_id) DO NOTHING
	`

	_, err := pool.Exec(ctx, query, clusterID, sourceID)
	if err != nil {
		log.Errorw("failed to insert cluster source", "cluster_id", clusterID, "source_id", sourceID, "error", err)
		return err
	}

	return nil
}

// UpdateClusterScoresBatch updates scores for multiple clusters in a single statement.
func UpdateClusterScoresBatch(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, ids []int64, scores []float64) error {
	if len(ids) == 0 {
		return nil
	}
	query := `
		UPDATE clusters AS c
		SET score = v.score
		FROM unnest($1::bigint[], $2::float8[]) AS v(id, score)
		WHERE c.id = v.id
	`
	_, err := pool.Exec(ctx, query, ids, scores)
	if err != nil {
		log.Errorw("failed to batch update cluster scores", "count", len(ids), "error", err)
		return err
	}
	return nil
}

// UpdateClusterVelocityBatch recalculates articles_last_6h for all clusters active in the last 48 hours.
func UpdateClusterVelocityBatch(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger) error {
	query := `
		UPDATE clusters
		SET articles_last_6h = (
			SELECT COUNT(*) FROM cluster_articles ca
			JOIN articles a ON a.id = ca.article_id
			WHERE ca.cluster_id = clusters.id
			  AND COALESCE(a.published_at, a.fetched_at) >= NOW() - INTERVAL '6 hours'
		)
		WHERE last_seen_at >= NOW() - INTERVAL '48 hours'
	`

	_, err := pool.Exec(ctx, query)
	if err != nil {
		log.Errorw("failed to batch update cluster velocity", "error", err)
		return err
	}

	return nil
}

// GetRecentArticleCount returns the number of articles added to a cluster in the last 6 hours.
func GetRecentArticleCount(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, clusterID int64) (int, error) {
	query := `
		SELECT COUNT(*) FROM cluster_articles ca
		JOIN articles a ON a.id = ca.article_id
		WHERE ca.cluster_id = $1
		  AND COALESCE(a.published_at, a.fetched_at) >= NOW() - INTERVAL '6 hours'
	`

	var count int
	err := pool.QueryRow(ctx, query, clusterID).Scan(&count)
	if err != nil {
		log.Errorw("failed to get recent article count for cluster", "cluster_id", clusterID, "error", err)
		return 0, err
	}

	return count, nil
}

// ListActiveClusters returns clusters with last_seen_at within the last 48 hours.
func ListActiveClusters(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger) ([]storage.Cluster, error) {
	query := `
		SELECT id, headline, display_title, summary, topic, score, created_at, updated_at,
		       last_seen_at, article_count, source_count, articles_last_6h, topic_confidence
		FROM clusters
		WHERE last_seen_at >= NOW() - INTERVAL '48 hours'
		ORDER BY id ASC
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Errorw("failed to query active clusters", "error", err)
		return nil, err
	}
	defer rows.Close()

	var clusters []storage.Cluster
	for rows.Next() {
		var c storage.Cluster
		if err := rows.Scan(
			&c.ID,
			&c.Headline,
			&c.DisplayTitle,
			&c.Summary,
			&c.Topic,
			&c.Score,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.LastSeenAt,
			&c.ArticleCount,
			&c.SourceCount,
			&c.ArticlesLast6h,
			&c.TopicConfidence,
		); err != nil {
			log.Errorw("failed to scan active cluster row", "error", err)
			return nil, err
		}
		clusters = append(clusters, c)
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating active cluster rows", "error", err)
		return nil, err
	}

	return clusters, nil
}

// GetArticlesInCluster retrieves all articles in a cluster.
func GetArticlesInCluster(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, clusterID int64) ([]storage.Article, error) {
	query := `
		SELECT a.id, a.source_id, a.url, a.image_url, a.title, a.snippet, a.normalized_title, a.normalized_snippet, a.rss_categories, a.topic, a.published_at, a.fetched_at
		FROM articles a
		INNER JOIN cluster_articles ca ON a.id = ca.article_id
		WHERE ca.cluster_id = $1
		ORDER BY a.published_at DESC
	`

	rows, err := pool.Query(ctx, query, clusterID)
	if err != nil {
		log.Errorw("failed to query articles in cluster", "cluster_id", clusterID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var articles []storage.Article
	for rows.Next() {
		var a storage.Article
		if err := rows.Scan(
			&a.ID,
			&a.SourceID,
			&a.URL,
			&a.ImageURL,
			&a.Title,
			&a.Snippet,
			&a.NormalizedTitle,
			&a.NormalizedSnippet,
			&a.RSSCategories,
			&a.Topic,
			&a.PublishedAt,
			&a.FetchedAt,
		); err != nil {
			log.Errorw("failed to scan article row", "error", err)
			return nil, err
		}
		articles = append(articles, a)
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating article rows", "error", err)
		return nil, err
	}

	return articles, nil
}

// ListClusterArticlesWithSources returns cluster articles joined with source metadata.
// Ordering is deterministic: newest published article first, then fetched_at, then article id.
func ListClusterArticlesWithSources(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, clusterID int64) ([]storage.ClusterArticleWithSource, error) {
	query := `
		SELECT a.id, a.url, a.image_url, a.title, a.snippet, a.published_at, a.fetched_at,
		       s.id, s.name, s.base_url, s.rss_url, to_jsonb(s)->>'logo_url' AS logo_url
		FROM cluster_articles ca
		INNER JOIN articles a ON a.id = ca.article_id
		INNER JOIN sources s ON s.id = a.source_id
		WHERE ca.cluster_id = $1
		ORDER BY (a.image_url IS NOT NULL) DESC, a.published_at DESC NULLS LAST, a.fetched_at DESC, a.id DESC
	`

	rows, err := pool.Query(ctx, query, clusterID)
	if err != nil {
		log.Errorw("failed to query cluster articles with sources", "cluster_id", clusterID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var out []storage.ClusterArticleWithSource
	for rows.Next() {
		var row storage.ClusterArticleWithSource
		if err := rows.Scan(
			&row.ArticleID,
			&row.ArticleURL,
			&row.ArticleImageURL,
			&row.ArticleTitle,
			&row.Snippet,
			&row.PublishedAt,
			&row.FetchedAt,
			&row.SourceID,
			&row.SourceName,
			&row.SourceBaseURL,
			&row.SourceRSSURL,
			&row.SourceLogoURL,
		); err != nil {
			log.Errorw("failed to scan cluster article/source row", "cluster_id", clusterID, "error", err)
			return nil, err
		}
		out = append(out, row)
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating cluster article/source rows", "cluster_id", clusterID, "error", err)
		return nil, err
	}

	return out, nil
}
