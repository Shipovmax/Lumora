// Package domain содержит сущности, доменные ошибки и порты (интерфейсы)
// домена auth. Пакет не зависит от service/repository/transport.
package domain

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
