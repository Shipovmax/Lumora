// Package aiworker — asynq-транспорт домена ai: обработчик задачи
// queue.TypeAIGenerate.
package aiworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/ai/service"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) HandleGenerate(ctx context.Context, t *asynq.Task) error {
	var payload queue.AIGeneratePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal %s payload: %w", queue.TypeAIGenerate, err)
	}

	explanation, err := h.svc.GenerateExplanation(ctx, payload.EventID, payload.UserID)
	if err != nil {
		return fmt.Errorf("generate explanation for event %s user %s: %w", payload.EventID, payload.UserID, err)
	}

	h.log.Info("ai: generated explanation",
		slog.String("event_id", payload.EventID),
		slog.String("user_id", payload.UserID),
		slog.String("model", explanation.Model),
	)
	return nil
}
