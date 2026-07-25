package domain

import "context"

// Repository — порт сохранения импортированных публикаций. Реализуется в
// internal/ingest/repository поверх Postgres/sqlc.
type Repository interface {
	// SaveNewPosts сохраняет публикации, пропуская уже импортированные
	// (дедупликация по паре source_id+external_id на уровне БД), и возвращает
	// только реально сохранённые (новые) записи.
	SaveNewPosts(ctx context.Context, posts []Post) ([]Post, error)
}
