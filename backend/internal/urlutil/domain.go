// Package urlutil holds small URL helpers shared by ingestion, topic
// classification and the feed service.
package urlutil

import (
	"net"
	"net/url"
	"strings"
)

// Domain reduces a URL, host or bare "host/path" to a lowercase hostname
// without port or leading "www.". Returns "" for empty input.
func Domain(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}

	if strings.Contains(trimmed, "://") {
		if parsed, err := url.Parse(trimmed); err == nil {
			trimmed = parsed.Host
		}
	}

	if strings.Contains(trimmed, "/") {
		if parsed, err := url.Parse("https://" + trimmed); err == nil && parsed.Host != "" {
			trimmed = parsed.Host
		}
	}

	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		trimmed = host
	}

	return strings.TrimPrefix(trimmed, "www.")
}
