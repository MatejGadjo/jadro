package storage

import (
	"context"
	"time"

	"github.com/MatejGadjo/NewsAggregator/domain"
	"github.com/MatejGadjo/NewsAggregator/internal/storage/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	pgvector "github.com/pgvector/pgvector-go"
)

func (c *Client) AttachArticleToCluster(ctx context.Context, clusterID, articleID int64) error {
	return c.q.AttachArticleToCluster(ctx, sqlc.AttachArticleToClusterParams{
		ClusterID: int32(clusterID),
		ArticleID: int32(articleID),
	})
}

// AttachAndUpdateVector atomically attaches an article to a cluster and updates the cluster vector.
// Returns true if the article was newly attached (not a duplicate).
func (c *Client) AttachAndUpdateVector(ctx context.Context, clusterID, articleID, sourceID int64, lastSeenAt time.Time, embedding []float32) (bool, error) {
	tx, err := c.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := c.q.WithTx(tx)
	count, err := qtx.AttachArticleToClusterWithStats(ctx, sqlc.AttachArticleToClusterWithStatsParams{
		ID:         int32(clusterID),
		ArticleID:  int32(articleID),
		SourceID:   int32(sourceID),
		LastSeenAt: pgtype.Timestamptz{Time: lastSeenAt, Valid: true},
	})
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, tx.Commit(ctx)
	}

	// Blend the new embedding into a running centroid instead of overwriting
	// (last-write-wins drift). FOR UPDATE locks the cluster_vectors row so
	// concurrent attaches to the same cluster serialize their read-modify-write
	// rather than losing updates; the lock is held until Commit below.
	current, err := qtx.GetClusterVectorForUpdate(ctx, clusterID)
	if err != nil {
		return false, err
	}
	n := int(current.ArticleCount)
	blended := blendCentroid(current.Vector.Slice(), n, embedding)
	if err := qtx.UpdateClusterVector(ctx, sqlc.UpdateClusterVectorParams{
		ClusterID:    clusterID,
		Vector:       pgvector.NewVector(blended),
		ArticleCount: int32(n + 1),
	}); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (c *Client) CreateCluster(ctx context.Context, headline, summary, displayTitle, topic string, lastSeenAt time.Time) (int64, error) {
	id, err := c.q.CreateCluster(ctx, sqlc.CreateClusterParams{
		Headline:     headline,
		Summary:      pgtype.Text{String: summary, Valid: true},
		DisplayTitle: displayTitle,
		Topic:        topic,
		LastSeenAt:   pgtype.Timestamptz{Time: lastSeenAt, Valid: true},
	})
	return int64(id), err
}

func (c *Client) FindSimilarClusterNeighbors(ctx context.Context, embedding []float32, hours, maxNeighbors int) ([]domain.ClusterNeighbor, error) {
	rows, err := c.q.FindSimilarClusterNeighbors(ctx, sqlc.FindSimilarClusterNeighborsParams{
		Column1: pgvector.NewVector(embedding),
		Column2: int32(hours),
		Limit:   int32(maxNeighbors),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.ClusterNeighbor, len(rows))
	for i, r := range rows {
		out[i] = domain.ClusterNeighbor{ClusterID: r.ClusterID, Similarity: r.Similarity}
	}
	return out, nil
}

func (c *Client) GetArticlesInCluster(ctx context.Context, clusterID int64) ([]domain.Article, error) {
	rows, err := c.q.GetArticlesInCluster(ctx, int32(clusterID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.Article, len(rows))
	for i, r := range rows {
		out[i] = domain.Article{
			ID:                int64(r.ID),
			SourceID:          int64(r.SourceID),
			URL:               r.Url,
			ImageURL:          pgtextPtr(r.ImageUrl),
			Title:             r.Title,
			Snippet:           pgtextStr(r.Snippet),
			NormalizedTitle:   r.NormalizedTitle,
			NormalizedSnippet: pgtextStr(r.NormalizedSnippet),
			RSSCategories:     r.RssCategories,
			Topic:             r.Topic,
			PublishedAt:       pgtstzPtr(r.PublishedAt),
			FetchedAt:         pgtstzTime(r.FetchedAt),
		}
	}
	return out, nil
}

func (c *Client) GetClusterByID(ctx context.Context, id int64) (domain.Cluster, error) {
	r, err := c.q.GetClusterByID(ctx, int32(id))
	if err != nil {
		return domain.Cluster{}, err
	}
	return domain.Cluster{
		ID:              int64(r.ID),
		Headline:        r.Headline,
		DisplayTitle:    r.DisplayTitle,
		Summary:         pgtextStr(r.Summary),
		Topic:           r.Topic,
		Score:           float64(r.Score),
		CreatedAt:       pgtstzTime(r.CreatedAt),
		UpdatedAt:       pgtstzTime(r.UpdatedAt),
		LastSeenAt:      pgtstzTime(r.LastSeenAt),
		ArticleCount:    int(r.ArticleCount),
		SourceCount:     int(r.SourceCount),
		ArticlesLast6h:  int(r.ArticlesLast6h),
		TopicConfidence: pgfloat4Ptr(r.TopicConfidence),
	}, nil
}

func (c *Client) GetClusterIDForArticle(ctx context.Context, articleID int64) (int64, error) {
	id, err := c.q.GetClusterIDForArticle(ctx, int32(articleID))
	return int64(id), err
}

func (c *Client) GetClusterIDsForArticles(ctx context.Context, ids []int64) ([]domain.ArticleClusterMapping, error) {
	rows, err := c.q.GetClusterIDsForArticles(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ArticleClusterMapping, len(rows))
	for i, r := range rows {
		out[i] = domain.ArticleClusterMapping{
			ArticleID: int64(r.ArticleID),
			ClusterID: int64(r.ClusterID),
		}
	}
	return out, nil
}

func (c *Client) GetClusterVector(ctx context.Context, clusterID int64) (domain.ClusterVector, error) {
	r, err := c.q.GetClusterVector(ctx, clusterID)
	if err != nil {
		return domain.ClusterVector{}, err
	}
	return domain.ClusterVector{
		Vector:       r.Vector.Slice(),
		ArticleCount: int(r.ArticleCount),
	}, nil
}

func (c *Client) GetRecentArticleCount(ctx context.Context, clusterID int64) (int64, error) {
	return c.q.GetRecentArticleCount(ctx, int32(clusterID))
}

func (c *Client) InsertClusterSource(ctx context.Context, clusterID, sourceID int64) error {
	return c.q.InsertClusterSource(ctx, sqlc.InsertClusterSourceParams{
		ClusterID: int32(clusterID),
		SourceID:  int32(sourceID),
	})
}

func (c *Client) InsertClusterVector(ctx context.Context, clusterID int64, vector []float32, articleCount int) error {
	return c.q.InsertClusterVector(ctx, sqlc.InsertClusterVectorParams{
		ClusterID:    clusterID,
		Vector:       pgvector.NewVector(vector),
		ArticleCount: int32(articleCount),
	})
}

func (c *Client) ListActiveClusters(ctx context.Context) ([]domain.Cluster, error) {
	rows, err := c.q.ListActiveClusters(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Cluster, len(rows))
	for i, r := range rows {
		out[i] = domain.Cluster{
			ID:              int64(r.ID),
			Headline:        r.Headline,
			DisplayTitle:    r.DisplayTitle,
			Summary:         pgtextStr(r.Summary),
			Topic:           r.Topic,
			Score:           float64(r.Score),
			CreatedAt:       pgtstzTime(r.CreatedAt),
			UpdatedAt:       pgtstzTime(r.UpdatedAt),
			LastSeenAt:      pgtstzTime(r.LastSeenAt),
			ArticleCount:    int(r.ArticleCount),
			SourceCount:     int(r.SourceCount),
			ArticlesLast6h:  int(r.ArticlesLast6h),
			TopicConfidence: pgfloat4Ptr(r.TopicConfidence),
		}
	}
	return out, nil
}

func (c *Client) ListClusterArticlesWithSources(ctx context.Context, clusterID int64) ([]domain.ClusterArticleSource, error) {
	rows, err := c.q.ListClusterArticlesWithSources(ctx, int32(clusterID))
	if err != nil {
		return nil, err
	}
	out := make([]domain.ClusterArticleSource, len(rows))
	for i, r := range rows {
		out[i] = domain.ClusterArticleSource{
			ArticleID:     int64(r.ArticleID),
			URL:           r.Url,
			ImageURL:      pgtextPtr(r.ImageUrl),
			Title:         r.Title,
			Snippet:       pgtextStr(r.Snippet),
			PublishedAt:   pgtstzPtr(r.PublishedAt),
			FetchedAt:     pgtstzTime(r.FetchedAt),
			SourceID:      int64(r.SourceID),
			SourceName:    r.Name,
			SourceBaseURL: r.BaseUrl,
			SourceRSSURL:  r.RssUrl,
			SourceLogoURL: pgtextPtr(r.LogoUrl),
		}
	}
	return out, nil
}

func (c *Client) ListClustersForFeedAll(ctx context.Context, limit, offset int32) ([]domain.Cluster, error) {
	rows, err := c.q.ListClustersForFeedAll(ctx, sqlc.ListClustersForFeedAllParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Cluster, len(rows))
	for i, r := range rows {
		out[i] = domain.Cluster{
			ID:              int64(r.ID),
			Headline:        r.Headline,
			DisplayTitle:    r.DisplayTitle,
			Summary:         pgtextStr(r.Summary),
			Topic:           r.Topic,
			Score:           float64(r.Score),
			CreatedAt:       pgtstzTime(r.CreatedAt),
			UpdatedAt:       pgtstzTime(r.UpdatedAt),
			LastSeenAt:      pgtstzTime(r.LastSeenAt),
			ArticleCount:    int(r.ArticleCount),
			SourceCount:     int(r.SourceCount),
			ArticlesLast6h:  int(r.ArticlesLast6h),
			TopicConfidence: pgfloat4Ptr(r.TopicConfidence),
		}
	}
	return out, nil
}

func (c *Client) ListClustersForFeedByTopic(ctx context.Context, topic string, limit, offset int32) ([]domain.Cluster, error) {
	rows, err := c.q.ListClustersForFeedByTopic(ctx, sqlc.ListClustersForFeedByTopicParams{Topic: topic, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Cluster, len(rows))
	for i, r := range rows {
		out[i] = domain.Cluster{
			ID:              int64(r.ID),
			Headline:        r.Headline,
			DisplayTitle:    r.DisplayTitle,
			Summary:         pgtextStr(r.Summary),
			Topic:           r.Topic,
			Score:           float64(r.Score),
			CreatedAt:       pgtstzTime(r.CreatedAt),
			UpdatedAt:       pgtstzTime(r.UpdatedAt),
			LastSeenAt:      pgtstzTime(r.LastSeenAt),
			ArticleCount:    int(r.ArticleCount),
			SourceCount:     int(r.SourceCount),
			ArticlesLast6h:  int(r.ArticlesLast6h),
			TopicConfidence: pgfloat4Ptr(r.TopicConfidence),
		}
	}
	return out, nil
}

func (c *Client) ListTopCategoryClustersWithArticles(ctx context.Context, topics []string, limit int32) ([]domain.ClusterWithFeaturedArticle, error) {
	rows, err := c.q.ListTopCategoryClustersWithArticles(ctx, sqlc.ListTopCategoryClustersWithArticlesParams{Column1: topics, Column2: limit})
	if err != nil {
		return nil, err
	}
	out := make([]domain.ClusterWithFeaturedArticle, len(rows))
	for i, r := range rows {
		out[i] = domain.ClusterWithFeaturedArticle{
			Cluster: domain.Cluster{
				ID:              int64(r.ID),
				Headline:        r.Headline,
				DisplayTitle:    r.DisplayTitle,
				Summary:         pgtextStr(r.Summary),
				Topic:           r.Topic,
				Score:           float64(r.Score),
				CreatedAt:       pgtstzTime(r.CreatedAt),
				UpdatedAt:       pgtstzTime(r.UpdatedAt),
				LastSeenAt:      pgtstzTime(r.LastSeenAt),
				ArticleCount:    int(r.ArticleCount),
				SourceCount:     int(r.SourceCount),
				ArticlesLast6h:  int(r.ArticlesLast6h),
				TopicConfidence: pgfloat4Ptr(r.TopicConfidence),
			},
			FeaturedArticle: domain.ClusterArticleSource{
				ArticleID:     int64(r.ArticleID),
				URL:           r.ArticleUrl,
				ImageURL:      pgtextPtr(r.ArticleImageUrl),
				Title:         r.ArticleTitle,
				Snippet:       pgtextStr(r.ArticleSnippet),
				PublishedAt:   pgtstzPtr(r.PublishedAt),
				FetchedAt:     pgtstzTime(r.FetchedAt),
				SourceID:      int64(r.SourceID),
				SourceName:    r.SourceName,
				SourceBaseURL: r.SourceBaseUrl,
				SourceRSSURL:  r.SourceRssUrl,
				SourceLogoURL: pgtextPtr(r.SourceLogoUrl),
			},
		}
	}
	return out, nil
}

func (c *Client) ListTopClustersByScore(ctx context.Context, limit int32) ([]domain.Cluster, error) {
	rows, err := c.q.ListTopClustersByScore(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Cluster, len(rows))
	for i, r := range rows {
		out[i] = domain.Cluster{
			ID:              int64(r.ID),
			Headline:        r.Headline,
			DisplayTitle:    r.DisplayTitle,
			Summary:         pgtextStr(r.Summary),
			Topic:           r.Topic,
			Score:           float64(r.Score),
			CreatedAt:       pgtstzTime(r.CreatedAt),
			UpdatedAt:       pgtstzTime(r.UpdatedAt),
			LastSeenAt:      pgtstzTime(r.LastSeenAt),
			ArticleCount:    int(r.ArticleCount),
			SourceCount:     int(r.SourceCount),
			ArticlesLast6h:  int(r.ArticlesLast6h),
			TopicConfidence: pgfloat4Ptr(r.TopicConfidence),
		}
	}
	return out, nil
}

func (c *Client) UpdateClusterDisplayTitle(ctx context.Context, id int64, displayTitle string) error {
	return c.q.UpdateClusterDisplayTitle(ctx, sqlc.UpdateClusterDisplayTitleParams{
		ID:           int32(id),
		DisplayTitle: displayTitle,
	})
}

func (c *Client) UpdateClusterScoreValue(ctx context.Context, id int64, score float64) error {
	return c.q.UpdateClusterScoreValue(ctx, sqlc.UpdateClusterScoreValueParams{
		ID:    int32(id),
		Score: float32(score),
	})
}

func (c *Client) UpdateClusterScoresBatch(ctx context.Context, ids []int64, scores []float64) error {
	return c.q.UpdateClusterScoresBatch(ctx, sqlc.UpdateClusterScoresBatchParams{
		Column1: ids,
		Column2: scores,
	})
}

func (c *Client) DeleteOrphanedClusters(ctx context.Context) (int64, error) {
	tag, err := c.q.DeleteOrphanedClusters(ctx)
	return tag.RowsAffected(), err
}

func (c *Client) UpdateClusterVelocityBatch(ctx context.Context) error {
	return c.q.UpdateClusterVelocityBatch(ctx)
}
