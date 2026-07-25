// Package ingestworker — asynq-транспорт домена ingest: обработчик задачи
// queue.TypeIngestFetch.
package ingestworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/ingest/domain"
	"github.com/Shipovmax/Lumora/internal/ingest/service"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
)

// TaskEnqueuer — узкий порт постановки задач в очередь (реализуется *asynq.Client).
// Объявлен здесь, а не в domain/service, так как постановка следующей задачи
// пайплайна — забота транспортного/инфраструктурного слоя, а не бизнес-логики
// импорта: service.ImportSource ничего не знает про asynq.
type TaskEnqueuer interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type Handler struct {
	svc      *service.Service
	enqueuer TaskEnqueuer
	log      *slog.Logger
}

func NewHandler(svc *service.Service, enqueuer TaskEnqueuer, log *slog.Logger) *Handler {
	return &Handler{svc: svc, enqueuer: enqueuer, log: log}
}

func (h *Handler) HandleFetch(ctx context.Context, t *asynq.Task) error {
	var payload queue.IngestFetchPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal %s payload: %w", queue.TypeIngestFetch, err)
	}

	posts, err := h.svc.ImportSource(ctx, payload.SourceID)
	if err != nil {
		return fmt.Errorf("import source %s: %w", payload.SourceID, err)
	}

	h.log.Info("ingest: imported posts", slog.String("source_id", payload.SourceID), slog.Int("count", len(posts)))

	if len(posts) == 0 {
		return nil
	}

	h.enqueuePipelineProcess(posts)
	return nil
}

// enqueuePipelineProcess передаёт новые публикации на обработку (Этап 7). Ошибка
// постановки задачи не проваливает ingest:fetch — импорт уже сохранён и durable
// в БД, обработку можно поставить в очередь позже вручную.
func (h *Handler) enqueuePipelineProcess(posts []domain.Post) {
	postIDs := make([]string, 0, len(posts))
	for _, p := range posts {
		postIDs = append(postIDs, p.ID)
	}

	task, err := queue.NewPipelineProcessTask(postIDs)
	if err != nil {
		h.log.Error("ingest: build pipeline:process task", slog.Any("error", err))
		return
	}

	if _, err := h.enqueuer.Enqueue(task); err != nil {
		h.log.Error("ingest: enqueue pipeline:process", slog.Any("error", err))
	}
}
