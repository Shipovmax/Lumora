// Command worker обрабатывает фоновые задачи пайплайна (ingest → pipeline → ai →
// briefing → notification) из очереди asynq. Обработчики задач регистрируются
// доменами по мере их реализации (Этапы 6–10) — сейчас mux пуст.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/config"
	"github.com/Shipovmax/Lumora/internal/platform/logger"
	"github.com/Shipovmax/Lumora/internal/platform/postgres"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
	"github.com/Shipovmax/Lumora/internal/platform/redis"
)

func main() {
	if err := run(); err != nil {
		slog.Error("worker exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.LogLevel, cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pgPool, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer pgPool.Close()

	redisClient, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		return err
	}
	defer redisClient.Close()

	srv := queue.NewServer(cfg.Redis, cfg.Worker.Concurrency)
	mux := asynq.NewServeMux()

	// Этапы 6–10 добавят сюда: mux.HandleFunc(queue.TypeIngestFetch, ingestHandler.Handle)

	if err := srv.Start(mux); err != nil {
		return err
	}

	log.Info("worker started", slog.Int("concurrency", cfg.Worker.Concurrency))

	<-ctx.Done()

	log.Info("worker shutting down")
	srv.Shutdown()
	log.Info("worker stopped")

	return nil
}
