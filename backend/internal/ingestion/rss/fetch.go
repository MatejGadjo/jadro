package rss

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mmcdole/gofeed"
)

// fetchRSSFeed fetches and parses an RSS feed from the given URL.
// It uses resty with a 5-second timeout and sets a clear User-Agent.
// Returns the parsed feed or an error.
func fetchRSSFeed(rssURL string) (*gofeed.Feed, error) {
	client := resty.New()
	client.SetTimeout(10 * time.Second)
	client.SetHeader("User-Agent", "NewsAggregator/1.0 (RSS Feed Reader)")

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
