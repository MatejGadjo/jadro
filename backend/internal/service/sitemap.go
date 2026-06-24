package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"
)

const sitemapBaseURL = "https://jadro.mk"

// sitemapClusterLimit caps how many cluster URLs go in the sitemap. With the
// 3-day retention the live set stays well under this; the cap is a safety bound.
const sitemapClusterLimit = 2000

// sitemapCacheTTL keeps repeated crawler hits from re-querying on every request.
const sitemapCacheTTL = 15 * time.Minute

// sitemapTopicSlugs are the latin URL slugs for topic pages, mirroring frontend
// src/lib/topics.ts. The Cyrillic topic values are used only for feed queries.
var sitemapTopicSlugs = []string{
	"makedonija", "hronika", "svet", "ekonomija", "sport",
	"skopje", "region", "tehnologija", "kultura", "scena",
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// Sitemap builds an XML sitemap: homepage + topic pages + recent cluster pages
// (the URLs the prerender now makes crawlable). Result is cached briefly.
func (s *FeedService) Sitemap(ctx context.Context) ([]byte, error) {
	const cacheKey = "sitemap:main"
	if s.cache != nil {
		if raw, err := s.cache.Get(ctx, cacheKey); err == nil {
			return raw, nil
		}
	}

	set := sitemapURLSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}

	set.URLs = append(set.URLs, sitemapURL{
		Loc: sitemapBaseURL + "/", ChangeFreq: "hourly", Priority: "1.0",
	})
	for _, slug := range sitemapTopicSlugs {
		set.URLs = append(set.URLs, sitemapURL{
			Loc: sitemapBaseURL + "/" + slug, ChangeFreq: "hourly", Priority: "0.8",
		})
	}

	clusters, err := s.store.ListClustersForFeedAll(ctx, sitemapClusterLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("sitemap: list clusters: %w", err)
	}
	for _, cl := range clusters {
		lastmod := cl.UpdatedAt
		if lastmod.IsZero() {
			lastmod = cl.CreatedAt
		}
		set.URLs = append(set.URLs, sitemapURL{
			Loc:        fmt.Sprintf("%s/cluster/%d", sitemapBaseURL, cl.ID),
			LastMod:    lastmod.UTC().Format(time.RFC3339),
			ChangeFreq: "daily",
			Priority:   "0.6",
		})
	}

	body, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("sitemap: marshal: %w", err)
	}
	out := append([]byte(xml.Header), body...)

	if s.cache != nil {
		if err := s.cache.Set(ctx, cacheKey, out, sitemapCacheTTL); err != nil {
			s.log.Warnw("sitemap cache set failed", "error", err)
		}
	}
	return out, nil
}
