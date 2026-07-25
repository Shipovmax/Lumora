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

	"github.com/Shipovmax/Lumora/internal/pipeline/domain"
	"github.com/Shipovmax/Lumora/internal/pipeline/repository"
)

// Интеграционный тест поднимает настоящий Postgres в Docker, прогоняет goose-миграции
// репозитория и проверяет Repository без моков БД, включая пересчёт source_count/importance
// при присоединении публикаций из разных источников к одному событию.
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
		"pipeline-owner@example.com", "hashed-password",
	).Scan(&userID))

	var sourceAID, sourceBID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO sources (user_id, type, name, url) VALUES ($1, 'rss', 'Feed A', 'https://a.example.com/rss') RETURNING id`,
		userID,
	).Scan(&sourceAID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO sources (user_id, type, name, url) VALUES ($1, 'rss', 'Feed B', 'https://b.example.com/rss') RETURNING id`,
		userID,
	).Scan(&sourceBID))

	var postAID, postBID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO posts (source_id, external_id, title, content) VALUES ($1, 'a1', 'Post A', 'content a') RETURNING id`,
		sourceAID,
	).Scan(&postAID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO posts (source_id, external_id, title, content) VALUES ($1, 'b1', 'Post B', 'content b') RETURNING id`,
		sourceBID,
	).Scan(&postBID))

	repo := repository.New(pool)

	posts, err := repo.GetPosts(ctx, []string{postAID, postBID})
	require.NoError(t, err)
	require.Len(t, posts, 2)

	now := time.Now()

	event, err := repo.CreateEventWithPost(ctx, domain.TopicWorld, "Post A", "post a content", postAID, now)
	require.NoError(t, err)
	require.Equal(t, 1, event.SourceCount)
	require.Equal(t, 20, event.Importance)

	recent, err := repo.ListRecentEvents(ctx, now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, recent, 1)
	require.Equal(t, event.ID, recent[0].ID)

	updated, err := repo.AttachPost(ctx, event.ID, postBID, "post a content post b content", now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, 2, updated.SourceCount, "two distinct sources now contribute to the event")
	require.Equal(t, 40, updated.Importance)
	require.Equal(t, "post a content post b content", updated.MatchText)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// internal/pipeline/repository/postgres_test.go -> repo root
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}
