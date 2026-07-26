// Package service реализует бизнес-логику домена notification: регистрацию
// токенов устройств и отправку push-уведомлений о важных событиях. Зависит
// только от портов, объявленных в domain (Repository, Sender).
package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Shipovmax/Lumora/internal/notification/domain"
)

type Service struct {
	repo   domain.Repository
	sender domain.Sender
	log    *slog.Logger
}

func New(repo domain.Repository, sender domain.Sender, log *slog.Logger) *Service {
	return &Service{repo: repo, sender: sender, log: log}
}

// RegisterDevice регистрирует токен устройства пользователя (upsert по токену
// — см. domain.Repository.RegisterDevice).
func (s *Service) RegisterDevice(ctx context.Context, userID, platform, token string) (domain.DeviceToken, error) {
	if token == "" {
		return domain.DeviceToken{}, domain.ErrTokenRequired
	}
	if !domain.ValidPlatform(platform) {
		return domain.DeviceToken{}, domain.ErrInvalidPlatform
	}

	return s.repo.RegisterDevice(ctx, userID, platform, token)
}

// RemoveDeviceToken удаляет токен по запросу самого пользователя (logout
// устройства, отключение уведомлений). В отличие от внутреннего вызова из
// NotifyEvent (который удаляет токен, признанный Sender-ом невалидным, без
// привязки к запросу пользователя), здесь проверяется владение токеном —
// иначе один пользователь мог бы удалить чужой токен по угаданному значению.
func (s *Service) RemoveDeviceToken(ctx context.Context, userID, token string) error {
	if token == "" {
		return domain.ErrTokenRequired
	}

	tokens, err := s.repo.ListDeviceTokens(ctx, userID)
	if err != nil {
		return err
	}
	for _, dt := range tokens {
		if dt.Token == token {
			return s.repo.RemoveDeviceToken(ctx, token)
		}
	}
	return domain.ErrInvalidToken
}

// NotifyEvent отправляет push-уведомление на все зарегистрированные устройства
// пользователя. ErrNoDeviceTokens — не ошибка обработки, а нормальное
// состояние (слать некуда), тот же паттерн, что и briefing.ErrNoRelevantEvents.
// Отправка на одно устройство не должна проваливать остальные: ошибки каждого
// Send логируются, а токен, признанный Sender-ом невалидным
// (domain.ErrInvalidToken), удаляется из хранилища — устройство отписалось
// или переустановило приложение без ротации токена.
func (s *Service) NotifyEvent(ctx context.Context, userID string, msg domain.PushMessage) error {
	tokens, err := s.repo.ListDeviceTokens(ctx, userID)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return domain.ErrNoDeviceTokens
	}

	for _, dt := range tokens {
		if err := s.sender.Send(ctx, dt.Token, msg); err != nil {
			if errors.Is(err, domain.ErrInvalidToken) {
				if rmErr := s.repo.RemoveDeviceToken(ctx, dt.Token); rmErr != nil {
					s.log.Error("notification: remove invalid token", slog.String("user_id", userID), slog.Any("error", rmErr))
				}
				continue
			}
			s.log.Error("notification: send push failed",
				slog.String("user_id", userID), slog.String("platform", dt.Platform), slog.Any("error", err))
			continue
		}
	}

	return nil
}
