package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/cache"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/storage"
	"github.com/MatejGadjo/NewsAggregator/internal/storage/postgres"
	"github.com/MatejGadjo/NewsAggregator/internal/topics"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const feedVersionKey = "meta:feed_version"

const (
	feedLimit            = 10
	featuredClusterCount = 2
	featuredMaxSources   = 4
	categoryClusterCount = 3
	homepageCandidateN   = 25
)

var homepageTopics = []string{
	"Македонија", "Хроника", "Свет", "Економија", "Спорт", "Регион",
}

type FeedHandler struct {
	log   *logger.Logger
	pool  *pgxpool.Pool
	cache *cache.Client
}

func NewFeedHandler(log *logger.Logger, pool *pgxpool.Pool, cache *cache.Client) *FeedHandler {
	return &FeedHandler{log: log, pool: pool, cache: cache}
}

type homepageSourceDTO struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	BaseURL string  `json:"base_url"`
	RSSURL  string  `json:"rss_url"`
	LogoURL *string `json:"logo_url,omitempty"`
	Domain  string  `json:"domain"`
}

type homepageArticleDTO struct {
	ID          int64      `json:"id"`
	URL         string     `json:"url"`
	ImageURL    *string    `json:"image_url,omitempty"`
	Title       string     `json:"title"`
	Snippet     string     `json:"snippet"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	FetchedAt   time.Time  `json:"fetched_at"`
}

type homepageSourceGroupDTO struct {
	Source   homepageSourceDTO    `json:"source"`
	Articles []homepageArticleDTO `json:"articles"`
}

type homepageClusterSectionDTO struct {
	Category string                   `json:"category,omitempty"`
	Cluster  storage.Cluster          `json:"cluster"`
	Sources  []homepageSourceGroupDTO `json:"sources"`
}

type homepageSourceBucket struct {
	source          homepageSourceDTO
	articles        []homepageArticleDTO
	latestArticleAt time.Time
}

func (h *FeedHandler) GetFeed(c *gin.Context) {
	ctx := c.Request.Context()
	requestedTopic := strings.TrimSpace(c.Query("topic"))
	var topicFilter *string
	if requestedTopic != "" {
		if !topics.IsAllowedTopicName(requestedTopic) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid topic"})
			return
		}
		topicFilter = &requestedTopic
	}

	page := 0
	if p := strings.TrimSpace(c.Query("page")); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	cacheKey := h.feedCacheKey(topicFilter, page)
	if h.cache != nil {
		if raw, err := h.cache.Get(ctx, cacheKey); err == nil {
			c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
			return
		}
	}

	offset := page * feedLimit

	clusters, err := postgres.ListClustersForFeed(ctx, h.pool, h.log, feedLimit, offset, topicFilter)
	if err != nil {
		h.log.Errorw("failed to fetch clusters for feed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch clusters"})
		return
	}

	feedVersion := h.readFeedVersion(ctx)

	var response gin.H

	if topicFilter != nil {
		response = gin.H{
			"feed_version":             feedVersion,
			"clusters":                 clusters,
			"featured_clusters":        []homepageClusterSectionDTO{},
			"top_clusters_by_category": []homepageClusterSectionDTO{},
		}
	} else {
		topCandidates, err := postgres.ListTopClustersByScore(ctx, h.pool, h.log, homepageCandidateN)
		if err != nil {
			h.log.Errorw("failed to fetch top clusters for homepage", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch homepage clusters"})
			return
		}

		featuredClusters, err := h.buildHomepageSections(ctx, topCandidates, featuredClusterCount, featuredMaxSources, false)
		if err != nil {
			h.log.Errorw("failed to build featured homepage clusters", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build featured clusters"})
			return
		}

		topClustersByCategory, err := h.buildCategoryHomepageSections(ctx)
		if err != nil {
			h.log.Errorw("failed to build category homepage clusters", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build category clusters"})
			return
		}

		response = gin.H{
			"feed_version":             feedVersion,
			"clusters":                 clusters,
			"featured_clusters":        featuredClusters,
			"top_clusters_by_category": topClustersByCategory,
		}
	}

	h.writeCached(c, ctx, cacheKey, response, 5*time.Minute)
}

func (h *FeedHandler) readFeedVersion(ctx context.Context) int64 {
	if h.cache == nil {
		return 0
	}
	raw, err := h.cache.Get(ctx, feedVersionKey)
	if err != nil {
		return 0
	}
	v, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func (h *FeedHandler) feedCacheKey(topic *string, page int) string {
	if topic != nil {
		return fmt.Sprintf("feed:topic:%s:p%d", *topic, page)
	}
	return fmt.Sprintf("feed:home:p%d", page)
}

// writeCached marshals v to JSON, stores it in cache under key, then writes it to the response.
// Falls back to c.JSON if marshaling fails.
func (h *FeedHandler) writeCached(c *gin.Context, ctx context.Context, key string, v any, ttl time.Duration) {
	raw, err := json.Marshal(v)
	if err != nil {
		c.JSON(http.StatusOK, v)
		return
	}
	if h.cache != nil {
		if err := h.cache.Set(ctx, key, raw, ttl); err != nil {
			h.log.Warnw("feed cache set failed", "key", key, "error", err)
		}
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
}

func (h *FeedHandler) buildHomepageSections(
	ctx context.Context,
	candidates []storage.Cluster,
	maxClusters int,
	maxSources int,
	includeCategory bool,
) ([]homepageClusterSectionDTO, error) {
	out := make([]homepageClusterSectionDTO, 0, maxClusters)
	for _, cluster := range candidates {
		if len(out) >= maxClusters {
			break
		}
		if cluster.ArticleCount <= 0 {
			continue
		}

		rows, err := postgres.ListClusterArticlesWithSources(ctx, h.pool, h.log, cluster.ID)
		if err != nil {
			return nil, err
		}

		sourceGroups := buildHomepageSourceGroups(rows, maxSources)
		if len(sourceGroups) == 0 {
			continue
		}

		item := homepageClusterSectionDTO{
			Cluster: cluster,
			Sources: sourceGroups,
		}
		if includeCategory {
			item.Category = cluster.Topic
		}
		out = append(out, item)
	}
	return out, nil
}

func buildHomepageSourceGroups(rows []storage.ClusterArticleWithSource, maxSources int) []homepageSourceGroupDTO {
	if maxSources <= 0 {
		return nil
	}

	articleSeen := make(map[int64]struct{}, len(rows))
	bucketsBySourceID := make(map[int64]*homepageSourceBucket, len(rows))

	for _, row := range rows {
		if _, exists := articleSeen[row.ArticleID]; exists {
			continue
		}
		articleSeen[row.ArticleID] = struct{}{}

		bucket, exists := bucketsBySourceID[row.SourceID]
		if !exists {
			bucket = &homepageSourceBucket{
				source: homepageSourceDTO{
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

		article := homepageArticleDTO{
			ID:          row.ArticleID,
			URL:         row.ArticleURL,
			ImageURL:    row.ArticleImageURL,
			Title:       row.ArticleTitle,
			Snippet:     row.Snippet,
			PublishedAt: row.PublishedAt,
			FetchedAt:   row.FetchedAt,
		}
		bucket.articles = append(bucket.articles, article)

		articleRankTime := articleOrderTime(row.PublishedAt, row.FetchedAt)
		if bucket.latestArticleAt.IsZero() || articleRankTime.After(bucket.latestArticleAt) {
			bucket.latestArticleAt = articleRankTime
		}
	}

	buckets := make([]*homepageSourceBucket, 0, len(bucketsBySourceID))
	for _, b := range bucketsBySourceID {
		buckets = append(buckets, b)
	}

	// Deterministic source ranking:
	// 1) newest article timestamp (published_at, fallback fetched_at),
	// 2) source article count in cluster,
	// 3) source name asc, then source id asc.
	sort.Slice(buckets, func(i, j int) bool {
		left := buckets[i]
		right := buckets[j]

		if !left.latestArticleAt.Equal(right.latestArticleAt) {
			return left.latestArticleAt.After(right.latestArticleAt)
		}

		if len(left.articles) != len(right.articles) {
			return len(left.articles) > len(right.articles)
		}

		leftName := strings.ToLower(strings.TrimSpace(left.source.Name))
		rightName := strings.ToLower(strings.TrimSpace(right.source.Name))
		if leftName != rightName {
			return leftName < rightName
		}

		if left.source.ID != right.source.ID {
			return left.source.ID < right.source.ID
		}

		return left.source.Domain < right.source.Domain
	})

	domainSeen := make(map[string]struct{}, len(buckets))
	out := make([]homepageSourceGroupDTO, 0, maxSources)
	for _, b := range buckets {
		domainKey := b.source.Domain
		if _, exists := domainSeen[domainKey]; exists {
			continue
		}
		domainSeen[domainKey] = struct{}{}

		out = append(out, homepageSourceGroupDTO{
			Source:   b.source,
			Articles: b.articles,
		})
		if len(out) >= maxSources {
			break
		}
	}

	return out
}

func (h *FeedHandler) buildCategoryHomepageSections(ctx context.Context) ([]homepageClusterSectionDTO, error) {
	rows, err := postgres.ListTopCategoryClustersWithArticles(ctx, h.pool, h.log, homepageTopics, categoryClusterCount)
	if err != nil {
		return nil, err
	}

	out := make([]homepageClusterSectionDTO, 0, len(rows))
	for _, r := range rows {
		article := homepageArticleDTO{
			ID:          r.ArticleID,
			URL:         r.ArticleURL,
			ImageURL:    r.ArticleImageURL,
			Title:       r.ArticleTitle,
			Snippet:     r.ArticleSnippet,
			PublishedAt: r.PublishedAt,
			FetchedAt:   r.FetchedAt,
		}
		source := homepageSourceDTO{
			ID:      r.SourceID,
			Name:    r.SourceName,
			BaseURL: r.SourceBaseURL,
			RSSURL:  r.SourceRSSURL,
			LogoURL: r.SourceLogoURL,
			Domain:  sourceDomain(r.SourceBaseURL, r.SourceRSSURL, r.SourceName),
		}
		out = append(out, homepageClusterSectionDTO{
			Category: r.Topic,
			Cluster:  r.Cluster,
			Sources: []homepageSourceGroupDTO{{
				Source:   source,
				Articles: []homepageArticleDTO{article},
			}},
		})
	}
	return out, nil
}

// WarmCache pre-populates Redis with feed and cluster responses for all topics + homepage.
// Called in a goroutine after each successful ingest+rescore cycle.
func (h *FeedHandler) WarmCache(ctx context.Context, ch *ClusterHandler) {
	if h.cache == nil {
		return
	}

	feedVersion := h.readFeedVersion(ctx)
	clusterIDs := make(map[int64]struct{})

	for _, topicStr := range topics.AllTopics() {
		t := topicStr
		clusters, err := postgres.ListClustersForFeed(ctx, h.pool, h.log, feedLimit, 0, &t)
		if err != nil {
			h.log.Warnw("cache warm: topic feed query failed", "topic", t, "error", err)
			continue
		}
		resp := gin.H{
			"feed_version":             feedVersion,
			"clusters":                 clusters,
			"featured_clusters":        []homepageClusterSectionDTO{},
			"top_clusters_by_category": []homepageClusterSectionDTO{},
		}
		if raw, err := json.Marshal(resp); err == nil {
			key := h.feedCacheKey(&t, 0)
			if err := h.cache.Set(ctx, key, raw, 20*time.Minute); err != nil {
				h.log.Warnw("cache warm: topic feed write failed", "topic", t, "error", err)
			}
		}
		for _, c := range clusters {
			clusterIDs[c.ID] = struct{}{}
		}
	}

	homeClusters, err := postgres.ListClustersForFeed(ctx, h.pool, h.log, feedLimit, 0, nil)
	if err != nil {
		h.log.Warnw("cache warm: homepage feed query failed", "error", err)
	} else {
		topCandidates, err := postgres.ListTopClustersByScore(ctx, h.pool, h.log, homepageCandidateN)
		if err != nil {
			h.log.Warnw("cache warm: homepage top clusters query failed", "error", err)
		} else {
			featured, _ := h.buildHomepageSections(ctx, topCandidates, featuredClusterCount, featuredMaxSources, false)
			byCategory, _ := h.buildCategoryHomepageSections(ctx)
			if featured == nil {
				featured = []homepageClusterSectionDTO{}
			}
			if byCategory == nil {
				byCategory = []homepageClusterSectionDTO{}
			}
			resp := gin.H{
				"feed_version":             feedVersion,
				"clusters":                 homeClusters,
				"featured_clusters":        featured,
				"top_clusters_by_category": byCategory,
			}
			if raw, err := json.Marshal(resp); err == nil {
				key := h.feedCacheKey(nil, 0)
				if err := h.cache.Set(ctx, key, raw, 20*time.Minute); err != nil {
					h.log.Warnw("cache warm: homepage feed write failed", "error", err)
				}
			}
		}
		for _, c := range homeClusters {
			clusterIDs[c.ID] = struct{}{}
		}
	}

	warmed := 0
	for id := range clusterIDs {
		if _, _, errMsg := ch.buildClusterJSON(ctx, id); errMsg == "" {
			warmed++
		} else {
			h.log.Warnw("cache warm: cluster build failed", "cluster_id", id, "error", errMsg)
		}
	}
	h.log.Infow("cache warm completed", "clusters_warmed", warmed)
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
