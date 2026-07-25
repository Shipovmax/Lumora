package domain

import "context"

// Sender — порт отправки push-уведомления на одно устройство. Реализация —
// internal/notification/provider (Firebase Cloud Messaging, согласовано с
// пользователем 2026-07-25, см. ARCHITECTURE.md, Этап 10); интерфейс не
// завязан на конкретного вендора.
type Sender interface {
	Send(ctx context.Context, token string, msg PushMessage) error
}
