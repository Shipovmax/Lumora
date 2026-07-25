package domain

import (
	"context"
	"time"
)

// Repository — порт доступа к данным домена source. Реализуется в
// internal/source/repository поверх Postgres/sqlc.
type Repository interface {
	CreateSource(ctx context.Context, userID string, typ Type, name, url string) (Source, error)
	ListSources(ctx context.Context, userID string) ([]Source, error)
	GetSource(ctx context.Context, userID, id string) (Source, error)
	SetEnabled(ctx context.Context, userID, id string, enabled bool) (Source, error)
	DeleteSource(ctx context.Context, userID, id string) error
}

// RawPost — сырая публикация, полученная от источника, до дедупликации и
// обработки (Этап 6/7).
type RawPost struct {
	ExternalID  string
	Title       string
	URL         string
	Content     string
	PublishedAt time.Time
}

// Fetcher — порт получения новых публикаций конкретного типа источника.
// Реализации (RSS/YouTube/Telegram) подключаются в Этапе 6 (internal/ingest),
// когда появляется логика получения, дедупликации и сохранения публикаций.
type Fetcher interface {
	Fetch(ctx context.Context, source Source) ([]RawPost, error)
}
