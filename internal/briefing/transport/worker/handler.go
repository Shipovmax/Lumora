// Package briefingworker — asynq-транспорт домена briefing: обработчики
// задач queue.TypeBriefingBuild и queue.TypeBriefingDispatch.
package briefingworker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/briefing/domain"
	"github.com/Shipovmax/Lumora/internal/briefing/service"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
)

// pushImportanceThreshold — минимальная важность события (см.
// pipeline.Repository: importance = source_count*20, до 100), начиная с
// которой брифинг триггерит push (Этап 10). 60 = событие подтверждено
// минимум 3 независимыми источниками — согласовано с принципом task.md
// «избегать лишних уведомлений»: пороговое значение, кандидат на пересмотр
// по метрикам, а не проектный инвариант (тот же статус, что и
// similarityThreshold в internal/pipeline/service).
const pushImportanceThreshold = 60

type Handler struct {
	svc      *service.Service
	enqueuer TaskEnqueuer
	log      *slog.Logger
}

func NewHandler(svc *service.Service, enqueuer TaskEnqueuer, log *slog.Logger) *Handler {
	return &Handler{svc: svc, enqueuer: enqueuer, log: log}
}

func (h *Handler) HandleBuild(ctx context.Context, t *asynq.Task) error {
	var payload queue.BriefingBuildPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal %s payload: %w", queue.TypeBriefingBuild, err)
	}

	briefing, err := h.svc.Build(ctx, payload.UserID, domain.Type(payload.Type))
	if err != nil {
		if errors.Is(err, domain.ErrNoRelevantEvents) {
			h.log.Info("briefing: nothing to send", slog.String("user_id", payload.UserID), slog.String("type", payload.Type))
			return nil
		}
		return fmt.Errorf("build briefing for user %s: %w", payload.UserID, err)
	}

	h.log.Info("briefing: built",
		slog.String("user_id", payload.UserID),
		slog.String("type", payload.Type),
		slog.Int("event_count", len(briefing.Events)),
	)

	h.pushImportantEvents(payload.UserID, briefing.Events)
	return nil
}

// pushImportantEvents ставит queue.TypeNotificationPush для событий брифинга,
// важность которых не ниже pushImportanceThreshold. Ошибка постановки задачи
// логируется и не проваливает briefing:build — брифинг уже сохранён, push
// можно поставить позже вручную (тот же принцип, что и у ingestworker.Handler
// для pipeline:process).
func (h *Handler) pushImportantEvents(userID string, events []domain.BriefingEvent) {
	for _, e := range events {
		if e.Importance < pushImportanceThreshold {
			continue
		}

		task, err := queue.NewNotificationPushTask(userID, e.Title, e.WhatHappened)
		if err != nil {
			h.log.Error("briefing: build notification:push task", slog.Any("error", err))
			continue
		}
		if _, err := h.enqueuer.Enqueue(task); err != nil {
			h.log.Error("briefing: enqueue notification:push",
				slog.String("user_id", userID), slog.String("event_id", e.EventID), slog.Any("error", err))
		}
	}
}
