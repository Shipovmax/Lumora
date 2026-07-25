package domain

import "context"

// Repository — порт хранения токенов устройств. Реализуется в
// internal/notification/repository поверх Postgres/sqlc.
type Repository interface {
	// RegisterDevice регистрирует токен устройства (upsert по токену —
	// переустановка приложения или обновление FCM-токена не создаёт дублей,
	// а переносит токен на нового владельца при смене аккаунта на устройстве).
	RegisterDevice(ctx context.Context, userID, platform, token string) (DeviceToken, error)
	ListDeviceTokens(ctx context.Context, userID string) ([]DeviceToken, error)
	// RemoveDeviceToken удаляет токен — вызывается при обнаружении невалидного/
	// отозванного токена после неудачной отправки (Sender вернул ErrInvalidToken).
	RemoveDeviceToken(ctx context.Context, token string) error
}
