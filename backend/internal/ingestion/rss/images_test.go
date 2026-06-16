package rss

import (
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseSingleItem(t *testing.T, itemXML string) *gofeed.Item {
	t.Helper()
	feedXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:media="http://search.yahoo.com/mrss/" xmlns:content="http://purl.org/rss/1.0/modules/content/">
<channel><title>t</title>` + itemXML + `</channel></rss>`
	feed, err := gofeed.NewParser().Parse(strings.NewReader(feedXML))
	require.NoError(t, err)
	require.Len(t, feed.Items, 1)
	return feed.Items[0]
}

func TestExtractItemImageURL_SkipsVideoMediaContent(t *testing.T) {
	// Skopsko Eho shape: media:content is a YouTube video, not an image.
	item := parseSingleItem(t, `<item>
		<title>x</title>
		<media:content url="https://www.youtube.com/watch?v=kC69hSVUHsw" medium="video" />
	</item>`)

	assert.Nil(t, extractItemImageURL(item))
}

func TestExtractItemImageURL_SkipsNonImageType(t *testing.T) {
	item := parseSingleItem(t, `<item>
		<title>x</title>
		<media:content url="https://example.com/clip.mp4" type="video/mp4" />
	</item>`)

	assert.Nil(t, extractItemImageURL(item))
}

func TestExtractItemImageURL_AcceptsImageMediaContent(t *testing.T) {
	item := parseSingleItem(t, `<item>
		<title>x</title>
		<media:content url="https://example.com/photo.jpg" medium="image" />
	</item>`)

	got := extractItemImageURL(item)
	require.NotNil(t, got)
	assert.Equal(t, "https://example.com/photo.jpg", *got)
}

func TestExtractItemImageURL_AcceptsMediaContentWithoutMedium(t *testing.T) {
	item := parseSingleItem(t, `<item>
		<title>x</title>
		<media:content url="https://example.com/photo.jpg" />
	</item>`)

	got := extractItemImageURL(item)
	require.NotNil(t, got)
	assert.Equal(t, "https://example.com/photo.jpg", *got)
}

func TestExtractItemImageURL_VideoThenImageFallsThrough(t *testing.T) {
	// Video media:content must not block a later image source.
	item := parseSingleItem(t, `<item>
		<title>x</title>
		<media:content url="https://www.youtube.com/watch?v=abc" medium="video" />
		<media:thumbnail url="https://example.com/thumb.jpg" />
	</item>`)

	got := extractItemImageURL(item)
	require.NotNil(t, got)
	assert.Equal(t, "https://example.com/thumb.jpg", *got)
}

func TestExtractItemImageURL_MediaGroupSkipsVideoChild(t *testing.T) {
	item := parseSingleItem(t, `<item>
		<title>x</title>
		<media:group>
			<media:content url="https://www.youtube.com/watch?v=abc" medium="video" />
			<media:content url="https://example.com/frame.jpg" medium="image" />
		</media:group>
	</item>`)

	got := extractItemImageURL(item)
	require.NotNil(t, got)
	assert.Equal(t, "https://example.com/frame.jpg", *got)
}
