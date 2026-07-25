package fetcher

import (
	"fmt"

	"github.com/Shipovmax/Lumora/internal/source/domain"
)

// Registry сопоставляет тип источника конкретной реализации domain.Fetcher.
type Registry struct {
	fetchers map[domain.Type]domain.Fetcher
}

func NewRegistry() *Registry {
	rss := NewRSSFetcher()
	return &Registry{
		fetchers: map[domain.Type]domain.Fetcher{
			domain.TypeRSS:      rss,
			domain.TypeYouTube:  rss, // публичный Atom-фид канала — тот же формат, что и RSS.
			domain.TypeTelegram: NewTelegramFetcher(),
		},
	}
}

func (r *Registry) For(t domain.Type) (domain.Fetcher, error) {
	f, ok := r.fetchers[t]
	if !ok {
		return nil, fmt.Errorf("no fetcher registered for source type %q", t)
	}
	return f, nil
}
