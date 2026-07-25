// Command worker обрабатывает фоновые задачи пайплайна (ingest → pipeline → ai →
// briefing → notification) из очереди asynq. Обработчики задач регистрируются
// доменами по мере их реализации (Этапы 6–10).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/Shipovmax/Lumora/internal/config"
	ingestrepo "github.com/Shipovmax/Lumora/internal/ingest/repository"
	ingestservice "github.com/Shipovmax/Lumora/internal/ingest/service"
	ingestworker "github.com/Shipovmax/Lumora/internal/ingest/transport/worker"
	pipelinerepo "github.com/Shipovmax/Lumora/internal/pipeline/repository"
	pipelineservice "github.com/Shipovmax/Lumora/internal/pipeline/service"
	pipelineworker "github.com/Shipovmax/Lumora/internal/pipeline/transport/worker"
	"github.com/Shipovmax/Lumora/internal/platform/logger"
	"github.com/Shipovmax/Lumora/internal/platform/postgres"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
	"github.com/Shipovmax/Lumora/internal/platform/redis"
	"github.com/Shipovmax/Lumora/internal/source/fetcher"
	sourcerepo "github.com/Shipovmax/Lumora/internal/source/repository"
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

	asynqClient := queue.NewClient(cfg.Redis)
	defer asynqClient.Close()

	srv := queue.NewServer(cfg.Redis, cfg.Worker.Concurrency)
	mux := asynq.NewServeMux()

	sourceRepo := sourcerepo.New(pgPool)
	ingestSvc := ingestservice.New(ingestrepo.New(pgPool), sourceRepo, fetcher.NewRegistry())
	ingestHandler := ingestworker.NewHandler(ingestSvc, asynqClient, log)

	pipelineSvc := pipelineservice.New(pipelinerepo.New(pgPool))
	pipelineHandler := pipelineworker.NewHandler(pipelineSvc, log)

	mux.HandleFunc(queue.TypeIngestFetch, ingestHandler.HandleFetch)
	mux.HandleFunc(queue.TypePipelineProcess, pipelineHandler.HandleProcess)

	// Этапы 8–10 добавят сюда обработчики ai:generate, briefing:build, notification:push.

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
