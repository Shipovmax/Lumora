package fetcher

import (
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/stretchr/testify/require"
)

func TestMapFeedItemPrefersGUIDAndContent(t *testing.T) {
	published := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	item := &gofeed.Item{
		Title:           "First post",
		Link:            "https://example.com/1",
		GUID:            "urn:uuid:1",
		Content:         "<p>Full content</p>",
		Description:     "Short description",
		PublishedParsed: &published,
	}

	post := mapFeedItem(item)
	require.Equal(t, "urn:uuid:1", post.ExternalID)
	require.Equal(t, "https://example.com/1", post.URL)
	require.Equal(t, "<p>Full content</p>", post.Content)
	require.Equal(t, published, post.PublishedAt)
}

func TestMapFeedItemFallsBackWhenGUIDAndContentMissing(t *testing.T) {
	updated := time.Date(2007, 3, 4, 0, 0, 0, 0, time.UTC)
	item := &gofeed.Item{
		Title:         "No guid",
		Link:          "https://example.com/2",
		Description:   "Only description",
		UpdatedParsed: &updated,
	}

	post := mapFeedItem(item)
	require.Equal(t, "https://example.com/2", post.ExternalID, "falls back to link when guid is empty")
	require.Equal(t, "Only description", post.Content, "falls back to description when content is empty")
	require.Equal(t, updated, post.PublishedAt, "falls back to updated date when published date is missing")
}

func TestParseStringExtractsItems(t *testing.T) {
	const rssXML = `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>First post</title>
      <link>https://example.com/1</link>
      <guid>urn:uuid:1</guid>
      <description>Hello &amp; welcome</description>
      <pubDate>Mon, 02 Jan 2006 15:04:05 +0000</pubDate>
    </item>
  </channel>
</rss>`

	parser := gofeed.NewParser()
	feed, err := parser.ParseString(rssXML)
	require.NoError(t, err)
	require.Len(t, feed.Items, 1)

	post := mapFeedItem(feed.Items[0])
	require.Equal(t, "urn:uuid:1", post.ExternalID)
	require.Equal(t, "https://example.com/1", post.URL)
	require.Equal(t, "Hello & welcome", post.Content)
	require.False(t, post.PublishedAt.IsZero())
}
