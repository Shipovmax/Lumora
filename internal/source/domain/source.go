// Package domain содержит сущности, доменные ошибки и порты (интерфейсы)
// домена source. Пакет не зависит от service/repository/transport.
package domain

import "time"

type Type string

const (
	TypeRSS      Type = "rss"
	TypeYouTube  Type = "youtube"
	TypeTelegram Type = "telegram"
)

func (t Type) Valid() bool {
	switch t {
	case TypeRSS, TypeYouTube, TypeTelegram:
		return true
	default:
		return false
	}
}

type Source struct {
	ID        string
	UserID    string
	Type      Type
	Name      string
	URL       string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
