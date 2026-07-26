// Package notificationworker — asynq-транспорт домена notification:
// обработчик задачи queue.TypeNotificationPush.
package notificationworker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/notification/domain"
	"github.com/Shipovmax/Lumora/internal/notification/service"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) HandlePush(ctx context.Context, t *asynq.Task) error {
	var payload queue.NotificationPushPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal %s payload: %w", queue.TypeNotificationPush, err)
	}

	err := h.svc.NotifyEvent(ctx, payload.UserID, domain.PushMessage{
		Title: payload.Title,
		Body:  payload.Body,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNoDeviceTokens) {
			h.log.Info("notification: no devices to push to", slog.String("user_id", payload.UserID))
			return nil
		}
		return fmt.Errorf("notify user %s: %w", payload.UserID, err)
	}

	h.log.Info("notification: pushed", slog.String("user_id", payload.UserID))
	return nil
}
