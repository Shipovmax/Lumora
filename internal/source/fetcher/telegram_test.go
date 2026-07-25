package fetcher

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Фрагмент разметки t.me/s/<channel> (упрощён до полей, которые парсим).
const telegramPreviewFixture = `<!DOCTYPE html>
<html>
<body>
  <div class="tgme_widget_message" data-post="examplechannel/123">
    <div class="tgme_widget_message_text js-message_text">Hello from Telegram &amp; welcome</div>
    <div class="tgme_widget_message_date">
      <time datetime="2024-01-02T15:04:05+00:00"></time>
    </div>
  </div>
  <div class="tgme_widget_message" data-post="examplechannel/124">
    <div class="tgme_widget_message_text js-message_text">Second message</div>
    <div class="tgme_widget_message_date">
      <time datetime="2024-01-03T00:00:00+00:00"></time>
    </div>
  </div>
</body>
</html>`

func TestParseTelegramPreview(t *testing.T) {
	posts, err := parseTelegramPreview(strings.NewReader(telegramPreviewFixture))
	require.NoError(t, err)
	require.Len(t, posts, 2)

	require.Equal(t, "examplechannel/123", posts[0].ExternalID)
	require.Equal(t, "https://t.me/examplechannel/123", posts[0].URL)
	require.Equal(t, "Hello from Telegram & welcome", posts[0].Content)
	require.Equal(t, time.Date(2024, 1, 2, 15, 4, 5, 0, time.UTC), posts[0].PublishedAt.UTC())

	require.Equal(t, "examplechannel/124", posts[1].ExternalID)
}

func TestParseTelegramPreviewSkipsMessagesWithoutDataPost(t *testing.T) {
	const html = `<div class="tgme_widget_message"><div class="tgme_widget_message_text">no id</div></div>`

	posts, err := parseTelegramPreview(strings.NewReader(html))
	require.NoError(t, err)
	require.Empty(t, posts)
}

func TestTelegramPreviewURL(t *testing.T) {
	cases := map[string]string{
		"https://t.me/examplechannel":   "https://t.me/s/examplechannel",
		"https://t.me/s/examplechannel": "https://t.me/s/examplechannel",
	}
	for input, want := range cases {
		got, err := telegramPreviewURL(input)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}

	_, err := telegramPreviewURL("https://example.com/channel")
	require.Error(t, err)

	_, err = telegramPreviewURL("https://t.me/")
	require.Error(t, err)
}
