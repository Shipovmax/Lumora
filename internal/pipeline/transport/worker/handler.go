// Package pipelineworker — asynq-транспорт домена pipeline: обработчик задачи
// queue.TypePipelineProcess.
package pipelineworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/pipeline/service"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) HandleProcess(ctx context.Context, t *asynq.Task) error {
	var payload queue.PipelineProcessPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal %s payload: %w", queue.TypePipelineProcess, err)
	}

	events, err := h.svc.Process(ctx, payload.PostIDs)
	if err != nil {
		return fmt.Errorf("process posts: %w", err)
	}

	h.log.Info("pipeline: processed posts", slog.Int("post_count", len(payload.PostIDs)), slog.Int("event_count", len(events)))
	return nil
}
