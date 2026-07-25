// Package domain содержит сущности и порты (интерфейсы) домена pipeline.
// Пакет не зависит от service/repository/transport.
package domain

import "time"

// Topic — фиксированная MVP-таксономия тем событий (см. README, пример брифинга).
// Расширение/пересмотр списка — по мере необходимости, не обязательно AI-driven.
type Topic string

const (
	TopicAI      Topic = "ai"
	TopicEconomy Topic = "economy"
	TopicCrypto  Topic = "crypto"
	TopicWorld   Topic = "world"
	TopicOther   Topic = "other"
)

// Event — кластер связанных публикаций (одно реальное событие, освещённое,
// возможно, несколькими источниками). Формируется пайплайном (Этап 7) из
// импортированных публикаций (Этап 6); AI-объяснение (Этап 8) и брифинг
// (Этап 9) строятся поверх Event, а не поверх сырых Post.
type Event struct {
	ID          string
	Topic       Topic
	Title       string
	MatchText   string // нормализованный текст для сравнения при кластеризации; не для показа пользователю.
	Importance  int    // 0..100, эвристика: чем больше независимых источников подтвердили событие, тем выше.
	SourceCount int
	FirstSeenAt time.Time
	LastSeenAt  time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PostRef — минимальные данные о публикации, нужные пайплайну для кластеризации.
// Не сама sqlc-модель и не ingest-domain.Post: pipeline не импортирует internal/ingest
// (никаких прямых междоменных импортов бизнес-логики), а читает posts через
// собственный узкий репозиторий-порт.
type PostRef struct {
	ID          string
	SourceID    string
	Title       string
	Content     string
	PublishedAt time.Time
}
