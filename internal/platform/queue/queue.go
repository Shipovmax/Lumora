// Package queue настраивает asynq поверх Redis — шину фоновых задач пайплайна
// (ingest → pipeline → ai → briefing → notification). Конкретные типы задач и
// их payload объявляются здесь как общий контракт между доменом-производителем
// и доменом-потребителем задачи; сама бизнес-логика обработки — в доменах.
package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/config"
)

// Типы задач пайплайна. Каждый асинхронный переход между стадиями (Этапы 6–10)
// — отдельный тип, обработчик регистрируется доменом, которому он принадлежит.
const (
	TypeIngestFetch      = "ingest:fetch"
	TypePipelineProcess  = "pipeline:process"
	TypeAIGenerate       = "ai:generate"
	TypeBriefingBuild    = "briefing:build"
	TypeBriefingDispatch = "briefing:dispatch"
)

// IngestFetchPayload — payload задачи TypeIngestFetch.
type IngestFetchPayload struct {
	SourceID string `json:"source_id"`
}

// PipelineProcessPayload — payload задачи TypePipelineProcess.
type PipelineProcessPayload struct {
	PostIDs []string `json:"post_ids"`
}

// AIGeneratePayload — payload задачи TypeAIGenerate. Ставится
// briefingworker.Handler (Этап 9) для событий, отобранных в брифинг пользователя.
type AIGeneratePayload struct {
	EventID string `json:"event_id"`
	UserID  string `json:"user_id"`
}

// BriefingBuildPayload — payload задачи TypeBriefingBuild.
type BriefingBuildPayload struct {
	UserID string `json:"user_id"`
	Type   string `json:"type"` // "morning" | "evening"
}

// BriefingDispatchPayload — payload задачи TypeBriefingDispatch: периодический
// (cron) триггер, который ставит TypeBriefingBuild для всех активных пользователей.
type BriefingDispatchPayload struct {
	Type string `json:"type"` // "morning" | "evening"
}

// NewPipelineProcessTask строит задачу TypePipelineProcess для переданных ID
// публикаций (обычно — только что сохранённых ingest.Service.ImportSource).
func NewPipelineProcessTask(postIDs []string) (*asynq.Task, error) {
	payload, err := json.Marshal(PipelineProcessPayload{PostIDs: postIDs})
	if err != nil {
		return nil, fmt.Errorf("marshal %s payload: %w", TypePipelineProcess, err)
	}
	return asynq.NewTask(TypePipelineProcess, payload), nil
}

// NewBriefingBuildTask строит задачу TypeBriefingBuild для конкретного
// пользователя (используется briefingworker.DispatchHandler при фан-ауте).
func NewBriefingBuildTask(userID, typ string) (*asynq.Task, error) {
	payload, err := json.Marshal(BriefingBuildPayload{UserID: userID, Type: typ})
	if err != nil {
		return nil, fmt.Errorf("marshal %s payload: %w", TypeBriefingBuild, err)
	}
	return asynq.NewTask(TypeBriefingBuild, payload), nil
}

// NewBriefingDispatchTask строит задачу TypeBriefingDispatch (используется при
// регистрации cron-расписания в cmd/worker).
func NewBriefingDispatchTask(typ string) (*asynq.Task, error) {
	payload, err := json.Marshal(BriefingDispatchPayload{Type: typ})
	if err != nil {
		return nil, fmt.Errorf("marshal %s payload: %w", TypeBriefingDispatch, err)
	}
	return asynq.NewTask(TypeBriefingDispatch, payload), nil
}

func redisConnOpt(cfg config.RedisConfig) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
}

// NewClient создаёт asynq.Client для постановки задач в очередь (используется из cmd/api и доменных сервисов).
func NewClient(cfg config.RedisConfig) *asynq.Client {
	return asynq.NewClient(redisConnOpt(cfg))
}

// NewServer создаёт asynq.Server для обработки задач (используется из cmd/worker).
func NewServer(cfg config.RedisConfig, concurrency int) *asynq.Server {
	return asynq.NewServer(redisConnOpt(cfg), asynq.Config{
		Concurrency: concurrency,
	})
}

// NewScheduler создаёт asynq.Scheduler для периодической (cron) постановки
// задач (используется из cmd/worker — утренний/вечерний briefing:dispatch).
// Часовой пояс — UTC: планировщик не учитывает часовой пояс пользователя
// (MVP-упрощение, см. ARCHITECTURE.md, Этап 9).
func NewScheduler(cfg config.RedisConfig) *asynq.Scheduler {
	return asynq.NewScheduler(redisConnOpt(cfg), &asynq.SchedulerOpts{Location: time.UTC})
}
