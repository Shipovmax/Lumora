// Package domain содержит сущности и порты (интерфейсы) домена ingest.
// Пакет не зависит от service/repository/transport.
package domain

import "time"

// Post — импортированная публикация источника, после дедупликации и подготовки
// текста, но до обработки пайплайном (Этап 7: очистка, кластеризация, важность).
type Post struct {
	ID          string
	SourceID    string
	ExternalID  string
	Title       string
	URL         string
	Content     string
	PublishedAt time.Time
	CreatedAt   time.Time
}
