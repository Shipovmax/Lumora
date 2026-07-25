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

	"github.com/Shipovmax/Lumora/internal/auth/domain"
	"github.com/Shipovmax/Lumora/internal/auth/repository"
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

	repo := repository.New(pool)

	user1, err := repo.CreateUser(ctx, "user@example.com", "hashed-password")
	require.NoError(t, err)
	require.NotEmpty(t, user1.ID)
	require.Equal(t, "user@example.com", user1.Email)

	_, err = repo.CreateUser(ctx, "user@example.com", "hashed-password")
	require.ErrorIs(t, err, domain.ErrEmailTaken)

	fetched, err := repo.GetUserByEmail(ctx, "user@example.com")
	require.NoError(t, err)
	require.Equal(t, user1.ID, fetched.ID)

	_, err = repo.GetUserByEmail(ctx, "unknown@example.com")
	require.ErrorIs(t, err, domain.ErrUserNotFound)

	rt, err := repo.CreateRefreshToken(ctx, user1.ID, "token-hash", time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Nil(t, rt.RevokedAt)

	require.NoError(t, repo.RevokeRefreshToken(ctx, rt.ID))

	revoked, err := repo.GetRefreshTokenByHash(ctx, "token-hash")
	require.NoError(t, err)
	require.NotNil(t, revoked.RevokedAt)

	_, err = repo.GetRefreshTokenByHash(ctx, "unknown-hash")
	require.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// internal/auth/repository/postgres_test.go -> repo root
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}
