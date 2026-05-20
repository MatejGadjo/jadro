package rss

import (
	"context"
	"encoding/json"
	"html"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/ingestion/normalize"
	"github.com/MatejGadjo/NewsAggregator/internal/jobs"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/storage"
	"github.com/MatejGadjo/NewsAggregator/internal/storage/postgres"
	"github.com/MatejGadjo/NewsAggregator/internal/topics"
	"github.com/alitto/pond/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mmcdole/gofeed"
)

// StartRSSIngestion fetches active sources and processes them using a pond worker pool.
// It respects rate limits per source and logs failures without stopping the entire process.
type ingestionStats struct {
	storedCount     int
	errorCount      int
	embeddingQueued int
}

const ingestionWorkerCount = 4

func StartRSSIngestion(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, pipeline *jobs.Pipeline) (int, error) {
	sources, err := postgres.ListActiveSources(ctx, pool, log)
	if err != nil {
		log.Errorw("failed to fetch active sources", "error", err)
		return 0, err
	}

	if len(sources) == 0 {
		log.Infow("no active sources to process")
		return 0, nil
	}

	// RSS fetch/store can run in parallel; embedding/clustering stays serialized in the dispatcher.
	pondPool := pond.NewPool(ingestionWorkerCount)

	// Rate limiters per source
	rateLimiters := make(map[int64]*rateLimiter)
	var mu sync.Mutex
	var totals ingestionStats
	var totalsMu sync.Mutex

	// Process each source
	for _, source := range sources {
		source := source // capture loop variable

		// Get or create rate limiter for this source
		mu.Lock()
		limiter, exists := rateLimiters[source.ID]
		if !exists {
			limiter = newRateLimiter(source.RateLimitPerMin)
			rateLimiters[source.ID] = limiter
		}
		mu.Unlock()

		pondPool.Submit(func() {
			// Wait for rate limit
			limiter.Wait()

			// Fetch RSS feed
			feed, err := FetchRSSFeed(source.RSSURL)
			if err != nil {
				log.Errorw("failed to fetch RSS feed",
					"source_id", source.ID,
					"source_name", source.Name,
					"rss_url", source.RSSURL,
					"error", err)
				return
			}

			log.Infow("successfully fetched RSS feed",
				"source_id", source.ID,
				"source_name", source.Name,
				"items_count", len(feed.Items))

			if isLogoMissing(source.LogoURL) {
				if logoURL := resolveSourceLogoURL(source, feed); logoURL != nil {
					if err := postgres.UpdateSourceLogoURL(ctx, pool, log, source.ID, logoURL); err != nil {
						log.Warnw("failed to persist source logo_url",
							"source_id", source.ID,
							"source_name", source.Name,
							"logo_url", logoURL,
							"error", err)
					}
				}
			}

			// Store fetched articles
			stats := StoreFetchedArticles(ctx, pool, log, pipeline, source.ID, source.Name, source.BaseURL, source.RSSURL, feed.Items)
			totalsMu.Lock()
			totals.storedCount += stats.storedCount
			totals.errorCount += stats.errorCount
			totals.embeddingQueued += stats.embeddingQueued
			totalsMu.Unlock()
		})
	}

	pondPool.StopAndWait()
	log.Infow("finished RSS ingestion totals",
		"stored", totals.storedCount,
		"errors", totals.errorCount,
		"embedding_queued", totals.embeddingQueued)

	return totals.storedCount, nil
}

// StoreFetchedArticles processes RSS items, normalizes them, classifies topics, and stores them.
// It handles deduplication by URL (via database constraint) and logs failures without stopping.
// It enqueues embedding jobs for new articles.
func StoreFetchedArticles(
	ctx context.Context,
	pool *pgxpool.Pool,
	log *logger.Logger,
	pipeline *jobs.Pipeline,
	sourceID int64,
	sourceName string,
	sourceBaseURL string,
	sourceRSSURL string,
	items []*gofeed.Item,
) ingestionStats {
	if len(items) == 0 {
		return ingestionStats{}
	}

	now := time.Now()
	sourceDomain := normalizeDomain(sourceBaseURL)
	storedCount := 0
	errorCount := 0
	embeddingQueued := 0

	for _, item := range items {
		// Extract data from RSS item
		title := item.Title
		if title == "" {
			log.Warnw("skipping article with empty title", "source_id", sourceID, "url", item.Link)
			errorCount++
			continue
		}

		url := item.Link
		if url == "" {
			log.Warnw("skipping article with empty URL", "source_id", sourceID, "title", title)
			errorCount++
			continue
		}
		imageURL := extractItemImageURL(item)

		// Get snippet from Description or Content
		snippet := item.Description
		if snippet == "" && item.Content != "" {
			snippet = item.Content
		}

		// Normalize title and snippet
		normalizedTitle, cleanedSnippet := normalize.NormalizeArticle(title, snippet)

		// Extract RSS categories for topic priors (cluster-first assignment)
		var rssCategories []string
		if len(item.Categories) > 0 {
			rssCategories = make([]string, 0, len(item.Categories))
			for _, cat := range item.Categories {
				cat = strings.TrimSpace(cat)
				if cat == "" {
					continue
				}
				rssCategories = append(rssCategories, cat)
			}
		}

		feedID := sourceID
		classification := topics.ClassifyArticle(topics.ClassificationInput{
			FeedID:        &feedID,
			FeedURL:       sourceRSSURL,
			Domain:        sourceDomain,
			URL:           url,
			Title:         title,
			Description:   cleanedSnippet,
			RSSCategories: rssCategories,
		})

		categorySignals, err := json.Marshal(classification.Signals)
		if err != nil {
			log.Warnw("failed to serialize category signals, storing empty object",
				"url", url,
				"source_id", sourceID,
				"error", err)
			categorySignals = []byte("{}")
		}

		if classification.Confidence < 0.70 {
			log.Infow("low-confidence category classification",
				"source_id", sourceID,
				"url", url,
				"category", classification.Category,
				"confidence", classification.Confidence,
				"method", classification.Method,
				"signals", classification.Signals)
		}

		// Create article struct
		article := storage.Article{
			SourceID:           sourceID,
			FeedID:             &feedID,
			FeedURL:            sourceRSSURL,
			SourceDomain:       sourceDomain,
			URL:                url,
			ImageURL:           imageURL,
			Title:              title,
			Snippet:            cleanedSnippet,
			NormalizedTitle:    normalizedTitle,
			NormalizedSnippet:  cleanedSnippet, // Store normalized snippet for clustering
			RSSCategories:      rssCategories,
			Category:           string(classification.Category),
			CategoryConfidence: classification.Confidence,
			CategoryMethod:     string(classification.Method),
			CategorySignals:    categorySignals,
			Topic:              string(classification.Category),
			PublishedAt:        item.PublishedParsed,
			FetchedAt:          now,
		}

		// Insert article (deduplication handled by database constraint on URL)
		articleID, isNew, err := postgres.InsertArticle(ctx, pool, log, article)
		if err != nil {
			log.Errorw("failed to insert article",
				"source_id", sourceID,
				"url", url,
				"title", title,
				"error", err)
			errorCount++
			continue
		}

		if !isNew {
			continue
		}

		if pipeline != nil {
			pipeline.EnqueueEmbed(articleID)
			embeddingQueued++
		} else {
			log.Warnw("embedding pipeline unavailable, skipping enqueue",
				"article_id", articleID,
				"url", url)
		}

		storedCount++
	}

	log.Infow("finished storing articles",
		"source_id", sourceID,
		"source_name", sourceName,
		"total_items", len(items),
		"stored", storedCount,
		"errors", errorCount,
		"embedding_queued", embeddingQueued)

	return ingestionStats{
		storedCount:     storedCount,
		errorCount:      errorCount,
		embeddingQueued: embeddingQueued,
	}
}

func normalizeDomain(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err == nil && parsed.Host != "" {
		raw = parsed.Host
	}
	raw = strings.TrimPrefix(raw, "www.")
	return raw
}

// rateLimiter implements a simple token bucket rate limiter per source.
type rateLimiter struct {
	interval time.Duration
	lastCall time.Time
	mu       sync.Mutex
}

func newRateLimiter(requestsPerMin int) *rateLimiter {
	interval := time.Minute / time.Duration(requestsPerMin)
	if interval < time.Second {
		interval = time.Second // minimum 1 second between requests
	}
	return &rateLimiter{
		interval: interval,
		lastCall: time.Time{},
	}
}

func (rl *rateLimiter) Wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if !rl.lastCall.IsZero() {
		elapsed := now.Sub(rl.lastCall)
		if elapsed < rl.interval {
			time.Sleep(rl.interval - elapsed)
		}
	}
	rl.lastCall = time.Now()
}

var imageSrcRegex = regexp.MustCompile(`(?is)<img[^>]+src=["']([^"']+)["']`)

func extractItemImageURL(item *gofeed.Item) *string {
	if item == nil {
		return nil
	}

	if item.Image != nil {
		if imageURL := normalizeImageURL(item.Image.URL); imageURL != nil {
			return imageURL
		}
	}

	for _, enclosure := range item.Enclosures {
		if enclosure == nil {
			continue
		}
		enclosureType := strings.ToLower(strings.TrimSpace(enclosure.Type))
		if enclosureType != "" && !strings.HasPrefix(enclosureType, "image/") {
			continue
		}
		if imageURL := normalizeImageURL(enclosure.URL); imageURL != nil {
			return imageURL
		}
	}

	if imageURL := extensionImageURL(item, "media", "content"); imageURL != nil {
		return imageURL
	}
	if imageURL := extensionImageURL(item, "media", "thumbnail"); imageURL != nil {
		return imageURL
	}
	if imageURL := extensionImageURL(item, "media", "group"); imageURL != nil {
		return imageURL
	}
	if imageURL := extensionImageURL(item, "enc", "enclosure"); imageURL != nil {
		return imageURL
	}
	if imageURL := extensionImageURL(item, "content", "encoded"); imageURL != nil {
		return imageURL
	}
	if imageURL := firstImageFromHTML(item.Content); imageURL != nil {
		return imageURL
	}
	if imageURL := firstImageFromHTML(item.Description); imageURL != nil {
		return imageURL
	}

	return nil
}

func extensionImageURL(item *gofeed.Item, namespace, name string) *string {
	if item == nil || item.Extensions == nil {
		return nil
	}

	ns := item.Extensions[namespace]
	if ns == nil {
		return nil
	}

	entries := ns[name]
	for _, ext := range entries {
		if imageURL := normalizeImageURL(ext.Attrs["url"]); imageURL != nil {
			return imageURL
		}
		if imageURL := normalizeImageURL(ext.Attrs["href"]); imageURL != nil {
			return imageURL
		}
		if imageURL := normalizeImageURL(ext.Attrs["src"]); imageURL != nil {
			return imageURL
		}
		if imageURL := normalizeImageURL(ext.Value); imageURL != nil {
			return imageURL
		}
		// media:group often nests media:content/media:thumbnail in children.
		for childName, children := range ext.Children {
			if childName != "content" && childName != "thumbnail" {
				continue
			}
			for _, child := range children {
				if imageURL := normalizeImageURL(child.Attrs["url"]); imageURL != nil {
					return imageURL
				}
				if imageURL := normalizeImageURL(child.Attrs["href"]); imageURL != nil {
					return imageURL
				}
				if imageURL := normalizeImageURL(child.Attrs["src"]); imageURL != nil {
					return imageURL
				}
				if imageURL := normalizeImageURL(child.Value); imageURL != nil {
					return imageURL
				}
			}
		}
	}

	return nil
}

func firstImageFromHTML(raw string) *string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	match := imageSrcRegex.FindStringSubmatch(raw)
	if len(match) < 2 {
		return nil
	}
	return normalizeImageURL(html.UnescapeString(match[1]))
}

func normalizeImageURL(raw string) *string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	// Support protocol-relative image URLs.
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	// Ignore non-web schemes and relative paths.
	if parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil
	}

	normalized := parsed.String()
	return &normalized
}

func isLogoMissing(logoURL *string) bool {
	return logoURL == nil || strings.TrimSpace(*logoURL) == ""
}

func resolveSourceLogoURL(source storage.Source, feed *gofeed.Feed) *string {
	if feed != nil && feed.Image != nil {
		if logoURL := normalizeImageURL(feed.Image.URL); logoURL != nil {
			return logoURL
		}
	}

	if logoURL := logoFromBaseURL(source.BaseURL); logoURL != nil {
		return logoURL
	}

	if logoURL := logoFromBaseURL(source.RSSURL); logoURL != nil {
		return logoURL
	}

	return nil
}

func logoFromBaseURL(rawURL string) *string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		parsed, err = url.Parse("https://" + strings.TrimPrefix(rawURL, "//"))
		if err != nil || parsed.Host == "" {
			return nil
		}
	}

	scheme := parsed.Scheme
	if scheme == "" {
		scheme = "https"
	}

	favicon := &url.URL{
		Scheme: scheme,
		Host:   parsed.Host,
		Path:   "/favicon.ico",
	}
	return normalizeImageURL(favicon.String())
}
