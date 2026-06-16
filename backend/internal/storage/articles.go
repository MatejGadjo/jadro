package storage

import (
	"context"
	"time"

	"github.com/MatejGadjo/NewsAggregator/domain"
	"github.com/MatejGadjo/NewsAggregator/internal/storage/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (c *Client) InsertArticle(ctx context.Context, arg domain.InsertArticleParams) (domain.InsertArticleResult, error) {
	row, err := c.q.InsertArticle(ctx, sqlc.InsertArticleParams{
		SourceID:           int32(arg.SourceID),
		FeedID:             int64ToInt4(arg.FeedID),
		FeedUrl:            strToText(arg.FeedURL),
		SourceDomain:       strToText(arg.SourceDomain),
		Url:                arg.URL,
		ImageUrl:           strToText(arg.ImageURL),
		Title:              arg.Title,
		Snippet:            strToText(arg.Snippet),
		NormalizedTitle:    arg.NormalizedTitle,
		NormalizedSnippet:  strToText(arg.NormalizedSnippet),
		RssCategories:      arg.RSSCategories,
		Category:           arg.Category,
		CategoryConfidence: float32(arg.CategoryConfidence),
		CategoryMethod:     arg.CategoryMethod,
		CategorySignals:    arg.CategorySignals,
		Topic:              arg.Topic,
		PublishedAt:        timeToTimestamptz(arg.PublishedAt),
		FetchedAt:          pgtype.Timestamptz{Time: arg.FetchedAt, Valid: true},
	})
	if err != nil {
		return domain.InsertArticleResult{}, err
	}
	return domain.InsertArticleResult{ID: int64(row.ID), IsNew: row.IsNew}, nil
}

func (c *Client) GetArticleByID(ctx context.Context, id int64) (domain.Article, error) {
	row, err := c.q.GetArticleByID(ctx, int32(id))
	if err != nil {
		return domain.Article{}, err
	}
	return domain.Article{
		ID:                 int64(row.ID),
		SourceID:           int64(row.SourceID),
		FeedID:             pgint4Ptr(row.FeedID),
		FeedURL:            pgtextStr(row.FeedUrl),
		SourceDomain:       pgtextStr(row.SourceDomain),
		URL:                row.Url,
		ImageURL:           pgtextPtr(row.ImageUrl),
		Title:              row.Title,
		Snippet:            pgtextStr(row.Snippet),
		NormalizedTitle:    row.NormalizedTitle,
		NormalizedSnippet:  pgtextStr(row.NormalizedSnippet),
		RSSCategories:      row.RssCategories,
		Category:           row.Category,
		CategoryConfidence: float64(row.CategoryConfidence),
		CategoryMethod:     row.CategoryMethod,
		CategorySignals:    row.CategorySignals,
		Topic:              row.Topic,
		PublishedAt:        pgtstzPtr(row.PublishedAt),
		FetchedAt:          pgtstzTime(row.FetchedAt),
		EmbeddingStatus:    domain.EmbeddingStatus(row.EmbeddingStatus),
	}, nil
}

// ClearArticleImageURL nulls an article's image_url. Returns pgx.ErrNoRows when
// no row was updated — i.e. the article doesn't exist or its image_url is
// already NULL — so the caller can return 404 without churning the row.
func (c *Client) ClearArticleImageURL(ctx context.Context, id int64) error {
	rows, err := c.q.ClearArticleImageURL(ctx, int32(id))
	if err != nil {
		return err
	}
	if rows == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (c *Client) DeleteOldArticles(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := c.q.DeleteOldArticles(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true})
	return tag.RowsAffected(), err
}

func (c *Client) ListLatestArticles(ctx context.Context, limit int32) ([]domain.LatestArticle, error) {
	rows, err := c.q.ListLatestArticles(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]domain.LatestArticle, len(rows))
	for i, r := range rows {
		out[i] = domain.LatestArticle{
			ID:            int64(r.ID),
			URL:           r.Url,
			Title:         r.Title,
			Topic:         r.Topic,
			PublishedAt:   pgtstzPtr(r.PublishedAt),
			FetchedAt:     pgtstzTime(r.FetchedAt),
			SourceName:    r.SourceName,
			SourceLogoURL: pgtextPtr(r.SourceLogoUrl),
		}
	}
	return out, nil
}
