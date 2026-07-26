package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/Shipovmax/Lumora/internal/source/domain"
)

// TelegramFetcher читает последние публикации с публичной HTML-версии канала
// (t.me/s/<channel>) — без токена бота и без доступа к Bot API, работает только
// для публичных каналов. Зависит от вёрстки Telegram и может сломаться при её
// изменении; официальный Bot API не рассматривался, так как требует, чтобы бот
// был добавлен администратором в каждый канал, что не подходит для произвольных
// публичных каналов, выбираемых пользователем (см. ARCHITECTURE.md, Этап 6).
type TelegramFetcher struct {
	client *http.Client
}

func NewTelegramFetcher() *TelegramFetcher {
	return &TelegramFetcher{client: &http.Client{Timeout: fetchTimeout}}
}

var _ domain.Fetcher = (*TelegramFetcher)(nil)

func (f *TelegramFetcher) Fetch(ctx context.Context, source domain.Source) ([]domain.RawPost, error) {
	previewURL, err := telegramPreviewURL(source.URL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, previewURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch telegram preview %q: %w", previewURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch telegram preview %q: unexpected status %d", previewURL, resp.StatusCode)
	}

	return parseTelegramPreview(resp.Body)
}

func parseTelegramPreview(r io.Reader) ([]domain.RawPost, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse telegram preview: %w", err)
	}

	var posts []domain.RawPost
	doc.Find(".tgme_widget_message").Each(func(_ int, s *goquery.Selection) {
		postID, ok := s.Attr("data-post")
		if !ok || postID == "" {
			return
		}

		content := strings.TrimSpace(s.Find(".tgme_widget_message_text").First().Text())
		datetime, _ := s.Find(".tgme_widget_message_date time").First().Attr("datetime")

		posts = append(posts, domain.RawPost{
			ExternalID:  postID,
			URL:         "https://t.me/" + postID,
			Content:     content,
			PublishedAt: parseTelegramTime(datetime),
		})
	})
	return posts, nil
}

func parseTelegramTime(v string) time.Time {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}
	}
	return t
}

// telegramPreviewURL превращает ссылку на канал (t.me/<channel> или уже
// t.me/s/<channel>) в ссылку на публичную HTML-версию t.me/s/<channel>.
func telegramPreviewURL(sourceURL string) (string, error) {
	u, err := url.Parse(sourceURL)
	if err != nil {
		return "", fmt.Errorf("invalid telegram source url %q: %w", sourceURL, err)
	}
	if u.Host != "t.me" && u.Host != "telegram.me" {
		return "", fmt.Errorf("telegram source url must be a t.me link, got %q", sourceURL)
	}

	channel := strings.TrimPrefix(strings.Trim(u.Path, "/"), "s/")
	if channel == "" {
		return "", fmt.Errorf("telegram source url must include a channel name, got %q", sourceURL)
	}

	return "https://t.me/s/" + channel, nil
}
