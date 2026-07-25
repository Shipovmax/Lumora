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

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
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
	return nil
}
