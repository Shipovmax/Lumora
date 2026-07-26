// Package domain содержит сущности и порты (интерфейсы) домена notification.
// Пакет не зависит от service/repository/transport.
package domain

import "time"

// Платформы устройств, поддерживаемые регистрацией токена.
const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
	PlatformWeb     = "web"
)

func ValidPlatform(platform string) bool {
	switch platform {
	case PlatformIOS, PlatformAndroid, PlatformWeb:
		return true
	default:
		return false
	}
}

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
