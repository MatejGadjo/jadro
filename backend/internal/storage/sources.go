package storage

import (
	"context"

	"github.com/MatejGadjo/NewsAggregator/domain"
	"github.com/MatejGadjo/NewsAggregator/internal/storage/sqlc"
)

func (c *Client) DisableSource(ctx context.Context, id int64) error {
	return c.q.DisableSource(ctx, int32(id))
}

func (c *Client) GetSourceByID(ctx context.Context, id int64) (domain.Source, error) {
	r, err := c.q.GetSourceByID(ctx, int32(id))
	if err != nil {
		return domain.Source{}, err
	}
	return domain.Source{
		ID:              int64(r.ID),
		Name:            r.Name,
		BaseURL:         r.BaseUrl,
		RSSURL:          r.RssUrl,
		LogoURL:         pgtextPtr(r.LogoUrl),
		Active:          r.Active,
		RateLimitPerMin: int(r.RateLimitPerMin),
		CreatedAt:       pgtstzTime(r.CreatedAt),
	}, nil
}

func (c *Client) ListActiveSources(ctx context.Context) ([]domain.Source, error) {
	rows, err := c.q.ListActiveSources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Source, len(rows))
	for i, r := range rows {
		out[i] = domain.Source{
			ID:              int64(r.ID),
			Name:            r.Name,
			BaseURL:         r.BaseUrl,
			RSSURL:          r.RssUrl,
			LogoURL:         pgtextPtr(r.LogoUrl),
			Active:          r.Active,
			RateLimitPerMin: int(r.RateLimitPerMin),
			CreatedAt:       pgtstzTime(r.CreatedAt),
		}
	}
	return out, nil
}

func (c *Client) UpdateSourceLogoURL(ctx context.Context, id int64, logoURL *string) error {
	return c.q.UpdateSourceLogoURL(ctx, sqlc.UpdateSourceLogoURLParams{
		ID:      int32(id),
		LogoUrl: strToText(logoURL),
	})
}
