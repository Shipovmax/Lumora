package domain

import "errors"

var (
	ErrTokenRequired   = errors.New("device token is required")
	ErrInvalidPlatform = errors.New("platform must be one of: ios, android, web")
	ErrInvalidToken    = errors.New("device token is invalid or unregistered")
	// ErrNoDeviceTokens — у пользователя нет ни одного зарегистрированного
	// устройства. Не ошибка обработки, а нормальное состояние (слать некуда) —
	// вызывающий код (asynq-обработчик) не должен считать это сбоем, тот же
	// паттерн, что и briefing.ErrNoRelevantEvents.
	ErrNoDeviceTokens = errors.New("user has no registered device tokens")
)
