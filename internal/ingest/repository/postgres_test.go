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

	"github.com/Shipovmax/Lumora/internal/ingest/domain"
	"github.com/Shipovmax/Lumora/internal/ingest/repository"
)

// Интеграционный тест поднимает настоящий Postgres в Docker, прогоняет goose-миграции
// репозитория и проверяет Repository без моков БД, включая дедупликацию по
// (source_id, external_id) на уровне БД. Запуск: go test -tags=integration ./...
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

	// posts.source_id references sources(id), которая references users(id).
	var userID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		"ingest-owner@example.com", "hashed-password",
	).Scan(&userID))

	var sourceID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO sources (user_id, type, name, url) VALUES ($1, 'rss', 'Feed', 'https://example.com/rss') RETURNING id`,
		userID,
	).Scan(&sourceID))

	repo := repository.New(pool)

	posts := []domain.Post{
		{SourceID: sourceID, ExternalID: "1", Title: "First", Content: "First content", PublishedAt: time.Now()},
		{SourceID: sourceID, ExternalID: "2", Title: "Second", Content: "Second content"},
	}

	saved, err := repo.SaveNewPosts(ctx, posts)
	require.NoError(t, err)
	require.Len(t, saved, 2)

	// Повторный импорт тех же публикаций (тот же source_id+external_id) не создаёт дублей.
	savedAgain, err := repo.SaveNewPosts(ctx, posts)
	require.NoError(t, err)
	require.Empty(t, savedAgain)

	// Смешанный батч: один дубль, одна новая публикация — сохраняется только новая.
	mixed, err := repo.SaveNewPosts(ctx, []domain.Post{
		posts[0],
		{SourceID: sourceID, ExternalID: "3", Title: "Third", Content: "Third content"},
	})
	require.NoError(t, err)
	require.Len(t, mixed, 1)
	require.Equal(t, "3", mixed[0].ExternalID)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// internal/ingest/repository/postgres_test.go -> repo root
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}
