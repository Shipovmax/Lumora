// Package domain содержит сущности и порты (интерфейсы) домена notification.
// Пакет не зависит от service/repository/transport.
package domain

import "time"

type DeviceToken struct {
	ID        string
	UserID    string
	Platform  string
	Token     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PushMessage — уведомление для отправки на одно устройство.
type PushMessage struct {
	Title string
	Body  string
	Data  map[string]string
}
