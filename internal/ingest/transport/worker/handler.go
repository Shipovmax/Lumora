// Package ingestworker — asynq-транспорт домена ingest: обработчик задачи
// queue.TypeIngestFetch.
package ingestworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/ingest/service"
)

// TaskPayload — payload задачи queue.TypeIngestFetch.
type TaskPayload struct {
	SourceID string `json:"source_id"`
}

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) HandleFetch(ctx context.Context, t *asynq.Task) error {
	var payload TaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal ingest:fetch payload: %w", err)
	}

	posts, err := h.svc.ImportSource(ctx, payload.SourceID)
	if err != nil {
		return fmt.Errorf("import source %s: %w", payload.SourceID, err)
	}

	h.log.Info("ingest: imported posts", slog.String("source_id", payload.SourceID), slog.Int("count", len(posts)))
	return nil
}
