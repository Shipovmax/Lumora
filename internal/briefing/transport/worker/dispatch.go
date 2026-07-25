package briefingworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/briefing/domain"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
)

// TaskEnqueuer — узкий порт постановки задач в очередь (реализуется *asynq.Client).
type TaskEnqueuer interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// DispatchHandler обрабатывает queue.TypeBriefingDispatch — периодический
// (cron) триггер: находит пользователей с хотя бы одним источником и ставит
// queue.TypeBriefingBuild для каждого.
type DispatchHandler struct {
	repo     domain.Repository
	enqueuer TaskEnqueuer
	log      *slog.Logger
}

func NewDispatchHandler(repo domain.Repository, enqueuer TaskEnqueuer, log *slog.Logger) *DispatchHandler {
	return &DispatchHandler{repo: repo, enqueuer: enqueuer, log: log}
}

func (h *DispatchHandler) HandleDispatch(ctx context.Context, t *asynq.Task) error {
	var payload queue.BriefingDispatchPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal %s payload: %w", queue.TypeBriefingDispatch, err)
	}

	userIDs, err := h.repo.ListActiveUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("list active users: %w", err)
	}

	for _, userID := range userIDs {
		task, err := queue.NewBriefingBuildTask(userID, payload.Type)
		if err != nil {
			h.log.Error("briefing: build briefing:build task", slog.Any("error", err))
			continue
		}
		if _, err := h.enqueuer.Enqueue(task); err != nil {
			h.log.Error("briefing: enqueue briefing:build", slog.String("user_id", userID), slog.Any("error", err))
		}
	}

	h.log.Info("briefing: dispatched", slog.String("type", payload.Type), slog.Int("user_count", len(userIDs)))
	return nil
}
