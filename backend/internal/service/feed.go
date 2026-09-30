package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MatejGadjo/jadro/domain"
	"github.com/MatejGadjo/jadro/internal/logger"
	"github.com/MatejGadjo/jadro/internal/topics"
)

const feedVersionKey = "meta:feed_version"

const (
	FeedLimit = 10
	// MaxFeedPage bounds ?page= so deep pages can't force expensive OFFSET
	// scans or flood the cache with one key per page number. 100 pages of 10
	// is far beyond what the retention window ever holds.
	MaxFeedPage          = 100
	featuredClusterCount = 2
	featuredMaxSources   = 4
	categoryClusterCount = 3
	homepageCandidateN   = 25
	// feedDetailArticleCap limits how many articles are embedded per cluster in
	// the feed response. Cards render the lead article plus three secondaries,
	// so five gives a small margin without bloating the payload.
	feedDetailArticleCap = 5
)

var homepageTopics = []string{
	"Македонија", "Хроника", "Свет", "Економија", "Спорт", "Регион",
}

type sourceDTO struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	BaseURL string  `json:"base_url"`
	RSSURL  string  `json:"rss_url"`
	LogoURL *string `json:"logo_url,omitempty"`
	Domain  string  `json:"domain"`
}

type articleDTO struct {
	ID          int64      `json:"id"`
	URL         string     `json:"url"`
	ImageURL    *string    `json:"image_url,omitempty"`
	Title       string     `json:"title"`
	Snippet     string     `json:"snippet"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	FetchedAt   time.Time  `json:"fetched_at"`
}

type sourceGroupDTO struct {
	Source   sourceDTO    `json:"source"`
	Articles []articleDTO `json:"articles"`
}

// ClusterSectionDTO is a cluster with its article sources, used in feed responses.
type ClusterSectionDTO struct {
	Category string           `json:"category,omitempty"`
	Cluster  domain.Cluster   `json:"cluster"`
	Sources  []sourceGroupDTO `json:"sources"`
}

// clusterDetailDTO is a cluster with its articles embedded directly, matching the
// shape of the /api/clusters/:id endpoint. Embedding these in the feed response
// lets the frontend render cluster cards without a second round-trip per cluster.
type clusterDetailDTO struct {
	Cluster  domain.Cluster      `json:"cluster"`
	Articles []clusterArticleDTO `json:"articles"`
}

type sourceBucket struct {
	source          sourceDTO
	articles        []articleDTO
	latestArticleAt time.Time
}

// FeedService handles feed assembly and cache warming.
type FeedService struct {
	store        FeedStore
	clusterStore ClusterStore
	cache        Cache
	log          *logger.Logger
}

func NewFeedService(store FeedStore, clusterStore ClusterStore, cache Cache, log *logger.Logger) *FeedService {
	return &FeedService{store: store, clusterStore: clusterStore, cache: cache, log: log}
}

// GetFeed returns a marshaled JSON feed response, reading from cache when available.
func (s *FeedService) GetFeed(ctx context.Context, topic *string, page int) ([]byte, error) {
	cacheKey := s.feedCacheKey(topic, page)
	if s.cache != nil {
		if raw, err := s.cache.Get(ctx, cacheKey); err == nil {
			return raw, nil
		}
	}

	offset := int32(page * FeedLimit)

	var clusters []domain.Cluster
	if topic != nil {
		rows, err := s.store.ListClustersForFeedByTopic(ctx, *topic, FeedLimit, offset)
		if err != nil {
			return nil, fmt.Errorf("list clusters by topic: %w", err)
		}
		clusters = rows
	} else {
		rows, err := s.store.ListClustersForFeedAll(ctx, FeedLimit, offset)
		if err != nil {
			return nil, fmt.Errorf("list clusters: %w", err)
		}
		clusters = rows
	}

	feedVersion := s.readFeedVersion(ctx)

	details, err := s.buildClusterDetails(ctx, clusters)
	if err != nil {
		return nil, fmt.Errorf("build cluster details: %w", err)
	}

	var response map[string]any

	if topic != nil {
		response = map[string]any{
			"feed_version":             feedVersion,
			"clusters":                 clusters,
			"cluster_details":          details,
			"featured_clusters":        []ClusterSectionDTO{},
			"top_clusters_by_category": []ClusterSectionDTO{},
		}
	} else {
		topCandidates, err := s.store.ListTopClustersByScore(ctx, homepageCandidateN)
		if err != nil {
			return nil, fmt.Errorf("list top clusters: %w", err)
		}

		featured, err := s.buildSections(ctx, topCandidates, featuredClusterCount, featuredMaxSources, false)
		if err != nil {
			return nil, fmt.Errorf("build featured sections: %w", err)
		}

		byCategory, err := s.buildCategorySections(ctx)
		if err != nil {
			return nil, fmt.Errorf("build category sections: %w", err)
		}

		response = map[string]any{
			"feed_version":             feedVersion,
			"clusters":                 clusters,
			"cluster_details":          details,
			"featured_clusters":        featured,
			"top_clusters_by_category": byCategory,
		}
	}

	raw, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("marshal feed: %w", err)
	}
	if s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, raw, 5*time.Minute); err != nil {
			s.log.Warnw("feed cache set failed", "key", cacheKey, "error", err)
		}
	}
	return raw, nil
}

// WarmCache invalidates stale cache entries then pre-populates Redis with fresh feed and cluster responses.
func (s *FeedService) WarmCache(ctx context.Context) {
	if s.cache == nil {
		return
	}

	// bump feed version so clients detect new content
	version := strconv.FormatInt(time.Now().Unix(), 10)
	if err := s.cache.Set(ctx, feedVersionKey, []byte(version), 0); err != nil {
		s.log.Warnw("cache invalidate: failed to write feed version", "error", err)
	}
	if err := s.cache.DeleteByPattern(ctx, "feed:*"); err != nil {
		s.log.Warnw("cache invalidate: feed pattern delete failed", "error", err)
	}
	if err := s.cache.DeleteByPattern(ctx, "cluster:*"); err != nil {
		s.log.Warnw("cache invalidate: cluster pattern delete failed", "error", err)
	}

	feedVersion := s.readFeedVersion(ctx)
	clusterIDs := make(map[int64]struct{})

	for _, topicStr := range topics.AllTopics() {
		t := topicStr
		rows, err := s.store.ListClustersForFeedByTopic(ctx, t, FeedLimit, 0)
		if err != nil {
			s.log.Warnw("cache warm: topic feed query failed", "topic", t, "error", err)
			continue
		}
		details, err := s.buildClusterDetails(ctx, rows)
		if err != nil {
			s.log.Warnw("cache warm: topic cluster details failed", "topic", t, "error", err)
			continue
		}
		resp := map[string]any{
			"feed_version":             feedVersion,
			"clusters":                 rows,
			"cluster_details":          details,
			"featured_clusters":        []ClusterSectionDTO{},
			"top_clusters_by_category": []ClusterSectionDTO{},
		}
		if raw, err := json.Marshal(resp); err == nil {
			key := s.feedCacheKey(&t, 0)
			if err := s.cache.Set(ctx, key, raw, 20*time.Minute); err != nil {
				s.log.Warnw("cache warm: topic feed write failed", "topic", t, "error", err)
			}
		}
		for _, c := range rows {
			clusterIDs[c.ID] = struct{}{}
		}
	}

	homeRows, err := s.store.ListClustersForFeedAll(ctx, FeedLimit, 0)
	if err != nil {
		s.log.Warnw("cache warm: homepage feed query failed", "error", err)
	} else {
		topCandidates, err := s.store.ListTopClustersByScore(ctx, homepageCandidateN)
		if err != nil {
			s.log.Warnw("cache warm: homepage top clusters query failed", "error", err)
		} else {
			featured, _ := s.buildSections(ctx, topCandidates, featuredClusterCount, featuredMaxSources, false)
			byCategory, _ := s.buildCategorySections(ctx)
			details, _ := s.buildClusterDetails(ctx, homeRows)
			if featured == nil {
				featured = []ClusterSectionDTO{}
			}
			if byCategory == nil {
				byCategory = []ClusterSectionDTO{}
			}
			if details == nil {
				details = []clusterDetailDTO{}
			}
			resp := map[string]any{
				"feed_version":             feedVersion,
				"clusters":                 homeRows,
				"cluster_details":          details,
				"featured_clusters":        featured,
				"top_clusters_by_category": byCategory,
			}
			if raw, err := json.Marshal(resp); err == nil {
				key := s.feedCacheKey(nil, 0)
				if err := s.cache.Set(ctx, key, raw, 20*time.Minute); err != nil {
					s.log.Warnw("cache warm: homepage feed write failed", "error", err)
				}
			}
		}
		for _, c := range homeRows {
			clusterIDs[c.ID] = struct{}{}
		}
	}

	warmed := 0
	for id := range clusterIDs {
		if _, _, err := buildClusterJSON(ctx, s.clusterStore, s.cache, s.log, id); err == nil {
			warmed++
		} else {
			s.log.Warnw("cache warm: cluster build failed", "cluster_id", id, "error", err)
		}
	}
	s.log.Infow("cache warm completed", "clusters_warmed", warmed)
}

func (s *FeedService) readFeedVersion(ctx context.Context) int64 {
	if s.cache == nil {
		return 0
	}
	raw, err := s.cache.Get(ctx, feedVersionKey)
	if err != nil {
		return 0
	}
	v, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func (s *FeedService) feedCacheKey(topic *string, page int) string {
	if topic != nil {
		return fmt.Sprintf("feed:topic:%s:p%d", *topic, page)
	}
	return fmt.Sprintf("feed:home:p%d", page)
}

func (s *FeedService) buildSections(
	ctx context.Context,
	candidates []domain.Cluster,
	maxClusters int,
	maxSources int,
	includeCategory bool,
) ([]ClusterSectionDTO, error) {
	out := make([]ClusterSectionDTO, 0, maxClusters)
	for _, cluster := range candidates {
		if len(out) >= maxClusters {
			break
		}
		if cluster.ArticleCount <= 0 {
			continue
		}
		rows, err := s.store.ListClusterArticlesWithSources(ctx, cluster.ID)
		if err != nil {
			return nil, err
		}
		groups := buildSourceGroups(rows, maxSources)
		if len(groups) == 0 {
			continue
		}
		item := ClusterSectionDTO{Cluster: cluster, Sources: groups}
		if includeCategory {
			item.Category = cluster.Topic
		}
		out = append(out, item)
	}
	return out, nil
}

// buildClusterDetails embeds each cluster's articles (capped) directly into the
// feed response, eliminating the per-cluster round-trip the frontend would
// otherwise make. Articles come pre-ordered image-first by the query, matching
// the frontend's own image-first reordering.
func (s *FeedService) buildClusterDetails(ctx context.Context, clusters []domain.Cluster) ([]clusterDetailDTO, error) {
	out := make([]clusterDetailDTO, 0, len(clusters))
	for _, cluster := range clusters {
		rows, err := s.store.ListClusterArticlesWithSources(ctx, cluster.ID)
		if err != nil {
			return nil, err
		}
		articles := make([]clusterArticleDTO, 0, feedDetailArticleCap)
		for _, r := range rows {
			if len(articles) >= feedDetailArticleCap {
				break
			}
			articles = append(articles, clusterArticleDTO{
				ID: r.ArticleID, URL: r.URL, ImageURL: r.ImageURL,
				Title: r.Title, Snippet: r.Snippet,
				PublishedAt: r.PublishedAt, FetchedAt: r.FetchedAt,
				Source: clusterSourceDTO{
					ID: r.SourceID, Name: r.SourceName,
					BaseURL: r.SourceBaseURL, RSSURL: r.SourceRSSURL,
					LogoURL: r.SourceLogoURL,
				},
			})
		}
		out = append(out, clusterDetailDTO{Cluster: cluster, Articles: articles})
	}
	return out, nil
}

func (s *FeedService) buildCategorySections(ctx context.Context) ([]ClusterSectionDTO, error) {
	rows, err := s.store.ListTopCategoryClustersWithArticles(ctx, homepageTopics, categoryClusterCount)
	if err != nil {
		return nil, err
	}

	out := make([]ClusterSectionDTO, 0, len(rows))
	for _, r := range rows {
		art := articleDTO{
			ID: r.FeaturedArticle.ArticleID, URL: r.FeaturedArticle.URL,
			ImageURL:    r.FeaturedArticle.ImageURL,
			Title:       r.FeaturedArticle.Title,
			Snippet:     r.FeaturedArticle.Snippet,
			PublishedAt: r.FeaturedArticle.PublishedAt,
			FetchedAt:   r.FeaturedArticle.FetchedAt,
		}
		src := sourceDTO{
			ID: r.FeaturedArticle.SourceID, Name: r.FeaturedArticle.SourceName,
			BaseURL: r.FeaturedArticle.SourceBaseURL,
			RSSURL:  r.FeaturedArticle.SourceRSSURL,
			LogoURL: r.FeaturedArticle.SourceLogoURL,
			Domain:  sourceDomain(r.FeaturedArticle.SourceBaseURL, r.FeaturedArticle.SourceRSSURL, r.FeaturedArticle.SourceName),
		}
		out = append(out, ClusterSectionDTO{
			Category: r.Cluster.Topic,
			Cluster:  r.Cluster,
			Sources:  []sourceGroupDTO{{Source: src, Articles: []articleDTO{art}}},
		})
	}
	return out, nil
}

func buildSourceGroups(rows []domain.ClusterArticleSource, maxSources int) []sourceGroupDTO {
	if maxSources <= 0 {
		return nil
	}

	articleSeen := make(map[int64]struct{}, len(rows))
	bucketsBySourceID := make(map[int64]*sourceBucket, len(rows))

	for _, row := range rows {
		if _, exists := articleSeen[row.ArticleID]; exists {
			continue
		}
		articleSeen[row.ArticleID] = struct{}{}

		bucket, exists := bucketsBySourceID[row.SourceID]
		if !exists {
			bucket = &sourceBucket{
				source: sourceDTO{
					ID:      row.SourceID,
					Name:    row.SourceName,
					BaseURL: row.SourceBaseURL,
					RSSURL:  row.SourceRSSURL,
					LogoURL: row.SourceLogoURL,
					Domain:  sourceDomain(row.SourceBaseURL, row.SourceRSSURL, row.SourceName),
				},
			}
			bucketsBySourceID[row.SourceID] = bucket
		}

		art := articleDTO{
			ID: row.ArticleID, URL: row.URL, ImageURL: row.ImageURL,
			Title: row.Title, Snippet: row.Snippet,
			PublishedAt: row.PublishedAt, FetchedAt: row.FetchedAt,
		}
		bucket.articles = append(bucket.articles, art)

		rankTime := articleOrderTime(row.PublishedAt, row.FetchedAt)
		if bucket.latestArticleAt.IsZero() || rankTime.After(bucket.latestArticleAt) {
			bucket.latestArticleAt = rankTime
		}
	}

	buckets := make([]*sourceBucket, 0, len(bucketsBySourceID))
	for _, b := range bucketsBySourceID {
		buckets = append(buckets, b)
	}

	sort.Slice(buckets, func(i, j int) bool {
		l, r := buckets[i], buckets[j]
		if !l.latestArticleAt.Equal(r.latestArticleAt) {
			return l.latestArticleAt.After(r.latestArticleAt)
		}
		if len(l.articles) != len(r.articles) {
			return len(l.articles) > len(r.articles)
		}
		ln := strings.ToLower(strings.TrimSpace(l.source.Name))
		rn := strings.ToLower(strings.TrimSpace(r.source.Name))
		if ln != rn {
			return ln < rn
		}
		if l.source.ID != r.source.ID {
			return l.source.ID < r.source.ID
		}
		return l.source.Domain < r.source.Domain
	})

	domainSeen := make(map[string]struct{}, len(buckets))
	out := make([]sourceGroupDTO, 0, maxSources)
	for _, b := range buckets {
		if _, exists := domainSeen[b.source.Domain]; exists {
			continue
		}
		domainSeen[b.source.Domain] = struct{}{}
		out = append(out, sourceGroupDTO{Source: b.source, Articles: b.articles})
		if len(out) >= maxSources {
			break
		}
	}
	return out
}

func articleOrderTime(publishedAt *time.Time, fetchedAt time.Time) time.Time {
	if publishedAt != nil {
		return *publishedAt
	}
	return fetchedAt
}

func sourceDomain(baseURL, rssURL, sourceName string) string {
	for _, raw := range []string{baseURL, rssURL} {
		if domain := extractDomain(raw); domain != "" {
			return domain
		}
	}
	fallback := strings.ToLower(strings.TrimSpace(sourceName))
	if fallback == "" {
		return "unknown-source"
	}
	return fallback
}

func extractDomain(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		if !strings.Contains(raw, "://") {
			u, err = url.Parse("https://" + raw)
		}
	}
	host := ""
	if err == nil && u != nil {
		host = u.Hostname()
	}
	if host == "" {
		host = raw
		if idx := strings.Index(host, "/"); idx >= 0 {
			host = host[:idx]
		}
		if idx := strings.Index(host, ":"); idx >= 0 {
			host = host[:idx]
		}
	}
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "www.")
	return host
}
