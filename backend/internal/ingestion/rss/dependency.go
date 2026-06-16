package rss

import (
	"context"

	"github.com/MatejGadjo/NewsAggregator/domain"
)

type IngestionStore interface {
	ListActiveSources(ctx context.Context) ([]domain.Source, error)
	UpdateSourceLogoURL(ctx context.Context, id int64, logoURL *string) error
	InsertArticle(ctx context.Context, arg domain.InsertArticleParams) (domain.InsertArticleResult, error)
}
