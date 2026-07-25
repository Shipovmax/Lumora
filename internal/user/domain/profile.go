// Package domain содержит сущности, доменные ошибки и порты (интерфейсы)
// домена user. Пакет не зависит от service/repository/transport.
package domain

import "time"

type Profile struct {
	UserID     string
	Name       string
	Country    string
	Language   string
	Profession string
	Interests  []string
	Topics     []string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
