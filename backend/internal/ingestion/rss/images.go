package rss

import (
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/MatejGadjo/jadro/domain"
	"github.com/mmcdole/gofeed"
)

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

	for _, ext := range ns[name] {
		if !isImageMedia(ext.Attrs) {
			continue
		}
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
		// media:group nests media:content/media:thumbnail in children
		for childName, children := range ext.Children {
			if childName != "content" && childName != "thumbnail" {
				continue
			}
			for _, child := range children {
				if !isImageMedia(child.Attrs) {
					continue
				}
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

// isImageMedia reports whether Media RSS attrs describe an image.
// Some feeds (e.g. WordPress) emit media:content with medium="video"
// pointing at YouTube pages — storing those as image_url renders a broken img.
func isImageMedia(attrs map[string]string) bool {
	if medium := strings.ToLower(strings.TrimSpace(attrs["medium"])); medium != "" && medium != "image" {
		return false
	}
	if mimeType := strings.ToLower(strings.TrimSpace(attrs["type"])); mimeType != "" && !strings.HasPrefix(mimeType, "image/") {
		return false
	}
	return true
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
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	if parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil
	}
	normalized := parsed.String()
	return &normalized
}

func isLogoMissing(logoURL *string) bool {
	return logoURL == nil || strings.TrimSpace(*logoURL) == ""
}

func resolveSourceLogoURL(source domain.Source, feed *gofeed.Feed) *string {
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
	favicon := &url.URL{Scheme: scheme, Host: parsed.Host, Path: "/favicon.ico"}
	return normalizeImageURL(favicon.String())
}
