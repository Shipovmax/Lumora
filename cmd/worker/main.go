// Command worker обрабатывает фоновые задачи пайплайна (ingest → pipeline → ai →
// briefing → notification) из очереди asynq и планировщик утреннего/вечернего
// briefing:dispatch (cron, UTC). Обработчики задач регистрируются доменами по
// мере их реализации (Этапы 6–10).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	aiprovider "github.com/Shipovmax/Lumora/internal/ai/provider"
	airepo "github.com/Shipovmax/Lumora/internal/ai/repository"
	aiservice "github.com/Shipovmax/Lumora/internal/ai/service"
	aiworker "github.com/Shipovmax/Lumora/internal/ai/transport/worker"
	briefingdomain "github.com/Shipovmax/Lumora/internal/briefing/domain"
	briefingrepo "github.com/Shipovmax/Lumora/internal/briefing/repository"
	briefingservice "github.com/Shipovmax/Lumora/internal/briefing/service"
	briefingworker "github.com/Shipovmax/Lumora/internal/briefing/transport/worker"
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
	usercontextrepo "github.com/Shipovmax/Lumora/internal/usercontext/repository"
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

	pipelineRepo := pipelinerepo.New(pgPool)
	pipelineSvc := pipelineservice.New(pipelineRepo)
	pipelineHandler := pipelineworker.NewHandler(pipelineSvc, log)

	userContextRepo := usercontextrepo.New(pgPool)
	aiRepo := airepo.New(pgPool)
	aiSvc := aiservice.New(aiRepo, pipelineRepo, userContextRepo, aiprovider.NewClaudeProvider())
	aiHandler := aiworker.NewHandler(aiSvc, log)

	briefingRepo := briefingrepo.New(pgPool)
	briefingSvc := briefingservice.New(briefingRepo, aiRepo, aiSvc, log)
	briefingHandler := briefingworker.NewHandler(briefingSvc, log)
	briefingDispatchHandler := briefingworker.NewDispatchHandler(briefingRepo, asynqClient, log)

	mux.HandleFunc(queue.TypeIngestFetch, ingestHandler.HandleFetch)
	mux.HandleFunc(queue.TypePipelineProcess, pipelineHandler.HandleProcess)
	mux.HandleFunc(queue.TypeAIGenerate, aiHandler.HandleGenerate)
	mux.HandleFunc(queue.TypeBriefingBuild, briefingHandler.HandleBuild)
	mux.HandleFunc(queue.TypeBriefingDispatch, briefingDispatchHandler.HandleDispatch)

	// Этап 10 добавит сюда обработчик notification:push.

	scheduler := queue.NewScheduler(cfg.Redis)

	morningDispatch, err := queue.NewBriefingDispatchTask(string(briefingdomain.TypeMorning))
	if err != nil {
		return err
	}
	if _, err := scheduler.Register("0 8 * * *", morningDispatch); err != nil {
		return fmt.Errorf("register morning briefing schedule: %w", err)
	}

	eveningDispatch, err := queue.NewBriefingDispatchTask(string(briefingdomain.TypeEvening))
	if err != nil {
		return err
	}
	if _, err := scheduler.Register("0 20 * * *", eveningDispatch); err != nil {
		return fmt.Errorf("register evening briefing schedule: %w", err)
	}

	if err := scheduler.Start(); err != nil {
		return fmt.Errorf("start scheduler: %w", err)
	}
	defer scheduler.Shutdown()

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
