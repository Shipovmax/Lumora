// Package domain содержит сущности, доменные ошибки и порты (интерфейсы)
// домена usercontext. Пакет не зависит от service/repository/transport.
package domain

import "time"

// Context — пользовательский AI-контекст: свободный текст, который используется
// при генерации брифинга для персонализации объяснений событий.
type Context struct {
	UserID    string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
