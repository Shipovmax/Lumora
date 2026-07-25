// Package service реализует бизнес-логику домена ingest: получение новых
// публикаций источника через Fetcher, подготовку текста и сохранение с
// дедупликацией. Зависит только от портов, объявленных в domain, и от узких
// интерфейсов (SourceRepository, FetcherRegistry), которые сам объявляет для
// нужных ему операций домена source — без импорта его service/repository.
package service

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"

	ingestdomain "github.com/Shipovmax/Lumora/internal/ingest/domain"
	sourcedomain "github.com/Shipovmax/Lumora/internal/source/domain"
)

// SourceRepository — часть domain.Repository источника, нужная ingest'у:
// получение источника по ID без привязки к пользователю (вызов из воркера,
// не из авторизованного HTTP-запроса).
type SourceRepository interface {
	GetSourceByID(ctx context.Context, id string) (sourcedomain.Source, error)
}

// FetcherRegistry — выбор domain.Fetcher по типу источника.
type FetcherRegistry interface {
	For(t sourcedomain.Type) (sourcedomain.Fetcher, error)
}

type Service struct {
	posts    ingestdomain.Repository
	sources  SourceRepository
	fetchers FetcherRegistry
}

func New(posts ingestdomain.Repository, sources SourceRepository, fetchers FetcherRegistry) *Service {
	return &Service{posts: posts, sources: sources, fetchers: fetchers}
}

// ImportSource получает новые публикации источника, готовит текст и сохраняет
// только те, которых ещё нет (дедупликация по source_id+external_id в репозитории).
// Отключённый источник пропускается без ошибки.
func (s *Service) ImportSource(ctx context.Context, sourceID string) ([]ingestdomain.Post, error) {
	source, err := s.sources.GetSourceByID(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	if !source.Enabled {
		return nil, nil
	}

	f, err := s.fetchers.For(source.Type)
	if err != nil {
		return nil, err
	}

	raw, err := f.Fetch(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("fetch source %s: %w", source.ID, err)
	}

	posts := make([]ingestdomain.Post, 0, len(raw))
	for _, rp := range raw {
		if rp.ExternalID == "" {
			continue
		}
		posts = append(posts, ingestdomain.Post{
			SourceID:    source.ID,
			ExternalID:  rp.ExternalID,
			Title:       prepareText(rp.Title),
			URL:         rp.URL,
			Content:     prepareText(rp.Content),
			PublishedAt: rp.PublishedAt,
		})
	}

	return s.posts.SaveNewPosts(ctx, posts)
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// prepareText — минимальная подготовка текста публикации к дальнейшей обработке
// (Этап 7: очистка, кластеризация): снимает HTML-теги, декодирует HTML-сущности,
// схлопывает повторяющиеся пробелы/переносы строк.
func prepareText(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}
