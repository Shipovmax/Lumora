package domain

import (
	"context"
	"time"
)

// Repository — порт доступа к данным домена pipeline. Реализуется в
// internal/pipeline/repository поверх Postgres/sqlc. Вся кластеризационная
// логика (сходство, выбор темы, порог важности) — в service; репозиторий
// только читает/пишет.
type Repository interface {
	// GetPosts возвращает публикации по ID (батч, полученный из ingest).
	GetPosts(ctx context.Context, postIDs []string) ([]PostRef, error)
	// GetEventByID возвращает событие по ID — используется доменом ai (Этап 8)
	// для чтения заголовка/match_text события перед генерацией объяснения.
	GetEventByID(ctx context.Context, id string) (Event, error)
	// ListRecentEvents возвращает события, обновлявшиеся не раньше since —
	// кандидаты для присоединения новых публикаций.
	ListRecentEvents(ctx context.Context, since time.Time) ([]Event, error)
	// CreateEventWithPost создаёт новое событие и сразу привязывает к нему post
	// (атомарно, в одной транзакции).
	CreateEventWithPost(ctx context.Context, topic Topic, title, matchText, postID string, publishedAt time.Time) (Event, error)
	// AttachPost присоединяет публикацию к существующему событию и пересчитывает
	// его source_count/importance/last_seen_at/match_text (атомарно).
	AttachPost(ctx context.Context, eventID, postID, matchText string, publishedAt time.Time) (Event, error)
}
