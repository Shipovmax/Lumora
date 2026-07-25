// Package queue настраивает asynq поверх Redis — шину фоновых задач пайплайна
// (ingest → pipeline → ai → briefing → notification). Конкретные типы задач
// объявляются доменами, которым они принадлежат; здесь — только общая инфраструктура.
package queue

import (
	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/config"
)

// Типы задач пайплайна. Каждый асинхронный переход между стадиями (Этапы 6–10)
// — отдельный тип, обработчик регистрируется доменом, которому он принадлежит.
const (
	TypeIngestFetch = "ingest:fetch"
)

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
