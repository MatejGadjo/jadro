package normalize

import (
	"html"
	"regexp"
	"strings"
)

const maxSnippetLength = 300

var (
	htmlTagRegex    = regexp.MustCompile(`<[^>]*>`)
	whitespaceRegex = regexp.MustCompile(`\s+`)
)

// NormalizeArticle normalizes an article's title and snippet according to MVP rules.
// It strips HTML from the snippet, truncates it to max 300 chars, and normalizes
// the title (lowercase + trim).
// Returns the normalized title and cleaned snippet.
func NormalizeArticle(title, snippet string) (normalizedTitle, cleanedSnippet string) {
	// Normalize title: lowercase, trim, remove punctuation
	normalizedTitle = normalizeTitle(title)

	// Clean snippet: strip HTML, decode entities, trim, truncate
	cleanedSnippet = cleanSnippet(snippet)

	return normalizedTitle, cleanedSnippet
}

// normalizeTitle converts title to lowercase and trims whitespace.
func normalizeTitle(title string) string {
	// Trim whitespace
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}

	// Decode HTML entities in title (common in RSS feeds)
	title = html.UnescapeString(title)

	// Convert to lowercase
	title = strings.ToLower(strings.TrimSpace(title))
	if title == "" {
		return ""
	}
	return title
}

// cleanSnippet strips HTML tags, decodes HTML entities, trims whitespace, and truncates to max 300 chars.
func cleanSnippet(snippet string) string {
	// Strip HTML tags
	cleaned := htmlTagRegex.ReplaceAllString(snippet, "")

	// Decode HTML entities (e.g., &amp; -> &, &lt; -> <)
	cleaned = html.UnescapeString(cleaned)

	// Trim whitespace
	cleaned = strings.TrimSpace(cleaned)

	// Normalize whitespace (multiple spaces/newlines to single space)
	cleaned = whitespaceRegex.ReplaceAllString(cleaned, " ")

	// Truncate to max 300 chars
	if len(cleaned) > maxSnippetLength {
		cleaned = cleaned[:maxSnippetLength]
		// Try to truncate at word boundary if possible
		if lastSpace := strings.LastIndex(cleaned, " "); lastSpace > maxSnippetLength/2 {
			cleaned = cleaned[:lastSpace]
		}
	}

	return strings.TrimSpace(cleaned)
}
