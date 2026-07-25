package domain

import "context"

// Repository — порт доступа к данным домена usercontext. Реализуется в
// internal/usercontext/repository поверх Postgres/sqlc.
type Repository interface {
	// GetContext возвращает контекст пользователя, создавая пустой (default) при первом обращении.
	GetContext(ctx context.Context, userID string) (Context, error)
	UpdateContext(ctx context.Context, userID, content string) (Context, error)
}
