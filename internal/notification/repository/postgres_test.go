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

	"github.com/Shipovmax/Lumora/internal/notification/repository"
)

// Интеграционный тест поднимает настоящий Postgres в Docker, прогоняет goose-миграции
// репозитория и проверяет Repository без моков БД, включая upsert по токену.
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

	var userID, otherUserID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		"device-owner@example.com", "hashed-password",
	).Scan(&userID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		"new-owner@example.com", "hashed-password",
	).Scan(&otherUserID))

	repo := repository.New(pool)

	device, err := repo.RegisterDevice(ctx, userID, "android", "token-abc")
	require.NoError(t, err)
	require.Equal(t, "android", device.Platform)

	tokens, err := repo.ListDeviceTokens(ctx, userID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.Equal(t, "token-abc", tokens[0].Token)

	// Тот же токен переустанавливается на другого пользователя (например, смена
	// аккаунта на устройстве) — upsert, а не дубль.
	reassigned, err := repo.RegisterDevice(ctx, otherUserID, "android", "token-abc")
	require.NoError(t, err)
	require.Equal(t, device.ID, reassigned.ID)

	tokensForOriginalOwner, err := repo.ListDeviceTokens(ctx, userID)
	require.NoError(t, err)
	require.Empty(t, tokensForOriginalOwner)

	tokensForNewOwner, err := repo.ListDeviceTokens(ctx, otherUserID)
	require.NoError(t, err)
	require.Len(t, tokensForNewOwner, 1)

	require.NoError(t, repo.RemoveDeviceToken(ctx, "token-abc"))

	tokensAfterRemoval, err := repo.ListDeviceTokens(ctx, otherUserID)
	require.NoError(t, err)
	require.Empty(t, tokensAfterRemoval)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// internal/notification/repository/postgres_test.go -> repo root
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}
