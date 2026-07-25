// Package fetcher содержит реализации domain.Fetcher по типу источника
// (см. internal/source/domain.Fetcher — порт объявлен в Этапе 5, реализации
// добавлены в Этапе 6, когда появилась причина их писать: internal/ingest).
package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/Shipovmax/Lumora/internal/source/domain"
)

// RSSFetcher читает RSS/Atom-фид по прямой ссылке. Используется и для типа
// "youtube" — публичный Atom-фид канала (youtube.com/feeds/videos.xml?channel_id=...)
// имеет тот же формат, отдельная реализация не нужна.
type RSSFetcher struct {
	parser *gofeed.Parser
}

func NewRSSFetcher() *RSSFetcher {
	return &RSSFetcher{parser: gofeed.NewParser()}
}

var _ domain.Fetcher = (*RSSFetcher)(nil)

func (f *RSSFetcher) Fetch(ctx context.Context, source domain.Source) ([]domain.RawPost, error) {
	feed, err := f.parser.ParseURLWithContext(source.URL, ctx)
	if err != nil {
		return nil, fmt.Errorf("parse feed %q: %w", source.URL, err)
	}

	posts := make([]domain.RawPost, 0, len(feed.Items))
	for _, item := range feed.Items {
		posts = append(posts, mapFeedItem(item))
	}
	return posts, nil
}

func mapFeedItem(item *gofeed.Item) domain.RawPost {
	return domain.RawPost{
		ExternalID:  feedItemExternalID(item),
		Title:       item.Title,
		URL:         item.Link,
		Content:     feedItemContent(item),
		PublishedAt: feedItemPublishedAt(item),
	}
}

func feedItemExternalID(item *gofeed.Item) string {
	if item.GUID != "" {
		return item.GUID
	}
	return item.Link
}

func feedItemContent(item *gofeed.Item) string {
	if item.Content != "" {
		return item.Content
	}
	return item.Description
}

func feedItemPublishedAt(item *gofeed.Item) time.Time {
	if item.PublishedParsed != nil {
		return *item.PublishedParsed
	}
	if item.UpdatedParsed != nil {
		return *item.UpdatedParsed
	}
	return time.Time{}
}
