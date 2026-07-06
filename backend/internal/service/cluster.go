package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/jackc/pgx/v5"
)

const clusterCacheTTL = 20 * time.Minute

type clusterSourceDTO struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	BaseURL string  `json:"base_url"`
	RSSURL  string  `json:"rss_url"`
	LogoURL *string `json:"logo_url,omitempty"`
}

type clusterArticleDTO struct {
	ID          int64            `json:"id"`
	URL         string           `json:"url"`
	ImageURL    *string          `json:"image_url,omitempty"`
	Title       string           `json:"title"`
	Snippet     string           `json:"snippet"`
	PublishedAt *time.Time       `json:"published_at,omitempty"`
	FetchedAt   time.Time        `json:"fetched_at"`
	Source      clusterSourceDTO `json:"source"`
}

// LatestArticleItem is a lightweight article summary for the "Свежо" sidebar.
type LatestArticleItem struct {
	ID            int64      `json:"id"`
	URL           string     `json:"url"`
	Title         string     `json:"title"`
	Topic         string     `json:"topic"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	FetchedAt     time.Time  `json:"fetched_at"`
	SourceName    string     `json:"source_name"`
	SourceLogoURL *string    `json:"source_logo_url,omitempty"`
}

const (
	LatestArticlesDefault = 8
	LatestArticlesMax     = 20
)

// ClusterService handles cluster detail and latest article responses.
type ClusterService struct {
	store ClusterStore
	cache Cache
	log   *logger.Logger
}

func NewClusterService(store ClusterStore, cache Cache, log *logger.Logger) *ClusterService {
	return &ClusterService{store: store, cache: cache, log: log}
}

func clusterCacheKey(id int64) string {
	return fmt.Sprintf("cluster:%d", id)
}

// GetCluster returns marshaled JSON for a cluster detail page, using cache when available.
func (s *ClusterService) GetCluster(ctx context.Context, id int64) ([]byte, int, error) {
	if s.cache != nil {
		if raw, err := s.cache.Get(ctx, clusterCacheKey(id)); err == nil {
			return raw, http.StatusOK, nil
		}
	}
	return buildClusterJSON(ctx, s.store, s.cache, s.log, id)
}

// WasEverAssigned reports whether id falls within the range of cluster ids
// ever issued (id <= current max), meaning a missing cluster with this id was
// deleted rather than never having existed.
func (s *ClusterService) WasEverAssigned(ctx context.Context, id int64) (bool, error) {
	maxID, err := s.store.MaxClusterID(ctx)
	if err != nil {
		return false, err
	}
	return id <= maxID, nil
}

// GetLatestArticles returns the most recent article per cluster for the sidebar feed.
func (s *ClusterService) GetLatestArticles(ctx context.Context, limit int) ([]LatestArticleItem, error) {
	rows, err := s.store.ListLatestArticles(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]LatestArticleItem, len(rows))
	for i, r := range rows {
		out[i] = LatestArticleItem{
			ID:            r.ID,
			URL:           r.URL,
			Title:         r.Title,
			Topic:         r.Topic,
			PublishedAt:   r.PublishedAt,
			FetchedAt:     r.FetchedAt,
			SourceName:    r.SourceName,
			SourceLogoURL: r.SourceLogoURL,
		}
	}
	return out, nil
}

// ClearArticleImage sets image_url to NULL for an article.
// Returns pgx.ErrNoRows if the article does not exist.
func (s *ClusterService) ClearArticleImage(ctx context.Context, id int64) error {
	return s.store.ClearArticleImageURL(ctx, id)
}

// buildClusterJSON fetches cluster detail, marshals to JSON, writes to cache, and returns bytes.
// Shared by ClusterService.GetCluster and FeedService.WarmCache.
func buildClusterJSON(ctx context.Context, store ClusterStore, c Cache, log *logger.Logger, id int64) ([]byte, int, error) {
	cluster, err := store.GetClusterByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, http.StatusNotFound, fmt.Errorf("cluster not found")
		}
		log.Errorw("failed to fetch cluster", "cluster_id", id, "error", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch cluster")
	}

	articles, err := store.GetArticlesInCluster(ctx, id)
	if err != nil {
		log.Errorw("failed to fetch articles in cluster", "cluster_id", id, "error", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch cluster articles")
	}

	sourceCache := make(map[int64]clusterSourceDTO, 16)
	outArticles := make([]clusterArticleDTO, 0, len(articles))
	for _, a := range articles {
		src, ok := sourceCache[a.SourceID]
		if !ok {
			s, err := store.GetSourceByID(ctx, a.SourceID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					log.Warnw("source missing for article", "source_id", a.SourceID, "article_id", a.ID)
					return nil, http.StatusInternalServerError, fmt.Errorf("source missing for article")
				}
				log.Errorw("failed to fetch source for article", "source_id", a.SourceID, "article_id", a.ID, "error", err)
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch article source")
			}
			src = clusterSourceDTO{
				ID: s.ID, Name: s.Name, BaseURL: s.BaseURL,
				RSSURL: s.RSSURL, LogoURL: s.LogoURL,
			}
			sourceCache[a.SourceID] = src
		}

		outArticles = append(outArticles, clusterArticleDTO{
			ID: a.ID, URL: a.URL, ImageURL: a.ImageURL,
			Title: a.Title, Snippet: a.Snippet,
			PublishedAt: a.PublishedAt, FetchedAt: a.FetchedAt,
			Source: src,
		})
	}

	raw, err := json.Marshal(map[string]any{"cluster": cluster, "articles": outArticles})
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal cluster response")
	}

	if c != nil {
		if err := c.Set(ctx, clusterCacheKey(id), raw, clusterCacheTTL); err != nil {
			log.Warnw("cluster cache set failed", "cluster_id", id, "error", err)
		}
	}

	return raw, http.StatusOK, nil
}
