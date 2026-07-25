//go:build integration

package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Shipovmax/Lumora/internal/ai/domain"
	"github.com/Shipovmax/Lumora/internal/ai/repository"
)

// Интеграционный тест поднимает настоящий Postgres в Docker, прогоняет goose-миграции
// репозитория и проверяет Repository без моков БД, включая upsert по (event_id, user_id).
// Запуск: go test -tags=integration ./...
func TestRepository(t *testing.T) {
	ctx := context.Background()

	const (
		user     = "lumora_test"
		password = "lumora_test"
		dbName   = "lumora_test"
	)

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:17-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     user,
				"POSTGRES_PASSWORD": password,
				"POSTGRES_DB":       dbName,
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port.Port(), dbName)

	migrationsDir := filepath.Join(repoRoot(t), "migrations")

	sqlDB, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer sqlDB.Close()

	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.Up(sqlDB, migrationsDir))

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	defer pool.Close()

	var userID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		"ai-owner@example.com", "hashed-password",
	).Scan(&userID))

	var sourceID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO sources (user_id, type, name, url) VALUES ($1, 'rss', 'Feed', 'https://example.com/rss') RETURNING id`,
		userID,
	).Scan(&sourceID))

	var eventID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO events (topic, title, match_text) VALUES ('ai', 'Some event', 'some event content') RETURNING id`,
	).Scan(&eventID))

	repo := repository.New(pool)

	_, err = repo.GetExplanation(ctx, eventID, userID)
	require.ErrorIs(t, err, domain.ErrExplanationNotFound)

	saved, err := repo.SaveExplanation(ctx, domain.Explanation{
		EventID:            eventID,
		UserID:             userID,
		WhatHappened:       "Something happened",
		WhyItHappened:      "Because of reasons",
		WhatChanged:        "Things changed",
		WhatItMeansForUser: "It matters to you",
		Model:              "claude-opus-5",
	})
	require.NoError(t, err)
	require.NotEmpty(t, saved.ID)

	fetched, err := repo.GetExplanation(ctx, eventID, userID)
	require.NoError(t, err)
	require.Equal(t, "Something happened", fetched.WhatHappened)

	// Повторная генерация для той же пары (event, user) — upsert, не дубль.
	updated, err := repo.SaveExplanation(ctx, domain.Explanation{
		EventID:            eventID,
		UserID:             userID,
		WhatHappened:       "Updated explanation",
		WhyItHappened:      "Because of reasons",
		WhatChanged:        "Things changed",
		WhatItMeansForUser: "It matters to you",
		Model:              "claude-opus-5",
	})
	require.NoError(t, err)
	require.Equal(t, saved.ID, updated.ID)
	require.Equal(t, "Updated explanation", updated.WhatHappened)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// internal/ai/repository/postgres_test.go -> repo root
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}
