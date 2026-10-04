package rss

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/MatejGadjo/jadro/domain"
	"github.com/MatejGadjo/jadro/internal/ingestion/normalize"
	"github.com/MatejGadjo/jadro/internal/jobs"
	"github.com/MatejGadjo/jadro/internal/logger"
	"github.com/MatejGadjo/jadro/internal/topics"
	"github.com/MatejGadjo/jadro/internal/urlutil"
	"github.com/alitto/pond/v2"
	"github.com/mmcdole/gofeed"
)

type ingestionStats struct {
	storedCount     int
	errorCount      int
	embeddingQueued int
}

const ingestionWorkerCount = 4

func StartRSSIngestion(ctx context.Context, store IngestionStore, log *logger.Logger, pipeline *jobs.Pipeline) (int, error) {
	sources, err := store.ListActiveSources(ctx)
	if err != nil {
		log.Errorw("failed to fetch active sources", "error", err)
		return 0, err
	}

	if len(sources) == 0 {
		log.Infow("no active sources to process")
		return 0, nil
	}

	pondPool := pond.NewPool(ingestionWorkerCount)

	// Only this loop touches the map, so it needs no lock.
	rateLimiters := make(map[int64]*rateLimiter)
	var totals ingestionStats
	var totalsMu sync.Mutex

	for _, source := range sources {
		limiter, exists := rateLimiters[source.ID]
		if !exists {
			limiter = newRateLimiter(source.RateLimitPerMin)
			rateLimiters[source.ID] = limiter
		}

		pondPool.Submit(func() {
			limiter.wait()

			feed, err := fetchRSSFeed(source.RSSURL)
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
					if err := store.UpdateSourceLogoURL(ctx, source.ID, logoURL); err != nil {
						log.Warnw("failed to persist source logo_url",
							"source_id", source.ID,
							"source_name", source.Name,
							"logo_url", logoURL,
							"error", err)
					}
				}
			}

			stats := storeFetchedArticles(ctx, store, log, pipeline, source.ID, source.Name, source.BaseURL, source.RSSURL, feed.Items)
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

func storeFetchedArticles(
	ctx context.Context,
	store IngestionStore,
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
	sourceDomain := urlutil.Domain(sourceBaseURL)
	storedCount := 0
	errorCount := 0
	embeddingQueued := 0

	for _, item := range items {
		title := item.Title
		if title == "" {
			log.Warnw("skipping article with empty title", "source_id", sourceID, "url", item.Link)
			errorCount++
			continue
		}

		articleURL := item.Link
		if articleURL == "" {
			log.Warnw("skipping article with empty URL", "source_id", sourceID, "title", title)
			errorCount++
			continue
		}

		snippet := item.Description
		if snippet == "" && item.Content != "" {
			snippet = item.Content
		}

		normalizedTitle, cleanedSnippet := normalize.NormalizeArticle(title, snippet)

		var rssCategories []string
		for _, cat := range item.Categories {
			if cat = strings.TrimSpace(cat); cat != "" {
				rssCategories = append(rssCategories, cat)
			}
		}

		classification := topics.ClassifyArticle(topics.ClassificationInput{
			FeedURL:       sourceRSSURL,
			Domain:        sourceDomain,
			URL:           articleURL,
			Title:         title,
			Description:   cleanedSnippet,
			RSSCategories: rssCategories,
		})

		categorySignals, err := json.Marshal(classification.Signals)
		if err != nil {
			log.Warnw("failed to serialize category signals, storing empty object",
				"url", articleURL, "source_id", sourceID, "error", err)
			categorySignals = []byte("{}")
		}

		if classification.Confidence < 0.70 {
			log.Infow("low-confidence category classification",
				"source_id", sourceID,
				"url", articleURL,
				"category", classification.Category,
				"confidence", classification.Confidence,
				"method", classification.Method,
				"signals", classification.Signals)
		}

		feedURL := sourceRSSURL
		sourceDomainVal := sourceDomain
		insertRow, err := store.InsertArticle(ctx, domain.InsertArticleParams{
			SourceID:           sourceID,
			FeedID:             sourceID,
			FeedURL:            &feedURL,
			SourceDomain:       &sourceDomainVal,
			URL:                articleURL,
			ImageURL:           extractItemImageURL(item),
			Title:              title,
			Snippet:            &cleanedSnippet,
			NormalizedTitle:    normalizedTitle,
			NormalizedSnippet:  &cleanedSnippet,
			RSSCategories:      rssCategories,
			Category:           string(classification.Category),
			CategoryConfidence: classification.Confidence,
			CategoryMethod:     string(classification.Method),
			CategorySignals:    categorySignals,
			Topic:              string(classification.Category),
			PublishedAt:        item.PublishedParsed,
			FetchedAt:          now,
		})
		if err != nil {
			log.Errorw("failed to insert article",
				"source_id", sourceID, "url", articleURL, "title", title, "error", err)
			errorCount++
			continue
		}

		articleID, isNew := insertRow.ID, insertRow.IsNew
		if !isNew {
			continue
		}

		if pipeline != nil {
			pipeline.EnqueueEmbed(articleID)
			embeddingQueued++
		} else {
			log.Warnw("embedding pipeline unavailable, skipping enqueue",
				"article_id", articleID, "url", articleURL)
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
