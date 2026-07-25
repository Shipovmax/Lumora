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

	"github.com/Shipovmax/Lumora/internal/source/domain"
	"github.com/Shipovmax/Lumora/internal/source/repository"
)

// Интеграционный тест поднимает настоящий Postgres в Docker, прогоняет goose-миграции
// репозитория и проверяет Repository без моков БД. Запуск: go test -tags=integration ./...
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

	// sources.user_id references users(id), поэтому сначала создаём пользователя.
	var userID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		"source-owner@example.com", "hashed-password",
	).Scan(&userID))

	repo := repository.New(pool)

	created, err := repo.CreateSource(ctx, userID, domain.TypeRSS, "Hacker News", "https://news.ycombinator.com/rss")
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.True(t, created.Enabled)

	list, err := repo.ListSources(ctx, userID)
	require.NoError(t, err)
	require.Len(t, list, 1)

	fetched, err := repo.GetSource(ctx, userID, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)

	disabled, err := repo.SetEnabled(ctx, userID, created.ID, false)
	require.NoError(t, err)
	require.False(t, disabled.Enabled)

	require.NoError(t, repo.DeleteSource(ctx, userID, created.ID))

	_, err = repo.GetSource(ctx, userID, created.ID)
	require.ErrorIs(t, err, domain.ErrSourceNotFound)

	err = repo.DeleteSource(ctx, userID, created.ID)
	require.ErrorIs(t, err, domain.ErrSourceNotFound)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// internal/source/repository/postgres_test.go -> repo root
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}
