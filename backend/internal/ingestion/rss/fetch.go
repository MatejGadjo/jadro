package rss

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MatejGadjo/jadro/internal/safehttp"
	"github.com/go-resty/resty/v2"
	"github.com/mmcdole/gofeed"
)

// maxFeedBytes caps a feed response; real feeds are well under 1 MiB.
const maxFeedBytes = 10 << 20

// feedClient only connects to public addresses, so a compromised publisher
// can't redirect the fetcher to services on this host or its private network.
var feedClient = resty.NewWithClient(safehttp.NewClient(safehttp.Options{
	Timeout:      10 * time.Second,
	MaxRedirects: 5,
})).
	SetHeader("User-Agent", "Jadro/1.0 (RSS reader; +https://github.com/MatejGadjo/jadro)").
	SetResponseBodyLimit(maxFeedBytes)

// fetchRSSFeed fetches and parses an RSS feed from the given URL.
// Returns the parsed feed or an error.
func fetchRSSFeed(rssURL string) (*gofeed.Feed, error) {
	client := feedClient

	resp, err := client.R().Get(rssURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() < http.StatusOK || resp.StatusCode() >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode(), resp.Status())
	}

	parser := gofeed.NewParser()
	feed, err := parser.ParseString(string(resp.Body()))
	if err != nil {
		return nil, err
	}

	return feed, nil
}
