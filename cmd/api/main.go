// Command api запускает REST API Lumora (Этап 11 подключит сюда доменные роуты).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Shipovmax/Lumora/internal/apihttp"
	authrepo "github.com/Shipovmax/Lumora/internal/auth/repository"
	authservice "github.com/Shipovmax/Lumora/internal/auth/service"
	authhttp "github.com/Shipovmax/Lumora/internal/auth/transport/http"
	"github.com/Shipovmax/Lumora/internal/config"
	"github.com/Shipovmax/Lumora/internal/platform/httpserver"
	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
	"github.com/Shipovmax/Lumora/internal/platform/logger"
	"github.com/Shipovmax/Lumora/internal/platform/postgres"
	"github.com/Shipovmax/Lumora/internal/platform/redis"
	userrepo "github.com/Shipovmax/Lumora/internal/user/repository"
	userservice "github.com/Shipovmax/Lumora/internal/user/service"
	userhttp "github.com/Shipovmax/Lumora/internal/user/transport/http"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api exited with error", slog.Any("error", err))
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

	router := httpserver.NewRouter(log)
	httpserver.RegisterHealth(router, map[string]httpserver.Checker{
		"postgres": func(ctx context.Context) error { return postgres.Ping(ctx, pgPool) },
		"redis":    func(ctx context.Context) error { return redis.Ping(ctx, redisClient) },
	})

	tokenIssuer := jwtauth.NewIssuer(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL)
	authSvc := authservice.New(authrepo.New(pgPool), tokenIssuer, cfg.Auth.RefreshTokenTTL)
	userSvc := userservice.New(userrepo.New(pgPool))

	apihttp.Mount(router, apihttp.Deps{
		Auth:           authhttp.NewHandler(authSvc, log),
		User:           userhttp.NewHandler(userSvc, log),
		AuthMiddleware: tokenIssuer.Middleware,
	})

	return httpserver.Run(
		ctx,
		log,
		cfg.Server.Addr(),
		router,
		cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout,
		cfg.Server.ShutdownTimeout,
	)
}
