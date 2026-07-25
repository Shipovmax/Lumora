// Package queue настраивает asynq поверх Redis — шину фоновых задач пайплайна
// (ingest → pipeline → ai → briefing → notification). Конкретные типы задач и
// их payload объявляются здесь как общий контракт между доменом-производителем
// и доменом-потребителем задачи; сама бизнес-логика обработки — в доменах.
package queue

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/config"
)

// Типы задач пайплайна. Каждый асинхронный переход между стадиями (Этапы 6–10)
// — отдельный тип, обработчик регистрируется доменом, которому он принадлежит.
const (
	TypeIngestFetch     = "ingest:fetch"
	TypePipelineProcess = "pipeline:process"
)

// IngestFetchPayload — payload задачи TypeIngestFetch.
type IngestFetchPayload struct {
	SourceID string `json:"source_id"`
}

// PipelineProcessPayload — payload задачи TypePipelineProcess.
type PipelineProcessPayload struct {
	PostIDs []string `json:"post_ids"`
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
