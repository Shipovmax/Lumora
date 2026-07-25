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

	"github.com/Shipovmax/Lumora/internal/briefing/domain"
	"github.com/Shipovmax/Lumora/internal/briefing/repository"
)

// Интеграционный тест поднимает настоящий Postgres в Docker, прогоняет goose-миграции
// репозитория и проверяет Repository без моков БД: отбор кандидатов только через
// собственные источники пользователя, исключение уже включённых в брифинг событий,
// атомарное создание брифинга. Запуск: go test -tags=integration ./...
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
		"briefing-owner@example.com", "hashed-password",
	).Scan(&userID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		"other-owner@example.com", "hashed-password",
	).Scan(&otherUserID))

	var sourceID, otherSourceID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO sources (user_id, type, name, url) VALUES ($1, 'rss', 'Feed', 'https://a.example.com/rss') RETURNING id`,
		userID,
	).Scan(&sourceID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO sources (user_id, type, name, url) VALUES ($1, 'rss', 'Other Feed', 'https://b.example.com/rss') RETURNING id`,
		otherUserID,
	).Scan(&otherSourceID))

	now := time.Now()

	var ownEventID, otherEventID string
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO events (topic, title, match_text, importance, last_seen_at) VALUES ('ai', 'Own event', 'content', 40, $1) RETURNING id`,
		now,
	).Scan(&ownEventID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO events (topic, title, match_text, importance, last_seen_at) VALUES ('world', 'Other event', 'content', 60, $1) RETURNING id`,
		now,
	).Scan(&otherEventID))

	_, err = pool.Exec(ctx, `INSERT INTO posts (source_id, external_id, title, content, event_id) VALUES ($1, 'p1', 'Post', 'content', $2)`, sourceID, ownEventID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO posts (source_id, external_id, title, content, event_id) VALUES ($1, 'p2', 'Post', 'content', $2)`, otherSourceID, otherEventID)
	require.NoError(t, err)

	repo := repository.New(pool)

	// Пользователю видно только событие через его собственный источник.
	candidates, err := repo.ListCandidateEvents(ctx, userID, now.Add(-time.Hour), 10)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, ownEventID, candidates[0].ID)

	activeUserIDs, err := repo.ListActiveUserIDs(ctx)
	require.NoError(t, err)
	require.Contains(t, activeUserIDs, userID)
	require.Contains(t, activeUserIDs, otherUserID)

	briefingID, generatedAt, err := repo.CreateBriefing(ctx, userID, domain.TypeMorning, []string{ownEventID})
	require.NoError(t, err)
	require.NotEmpty(t, briefingID)
	require.WithinDuration(t, time.Now(), generatedAt, time.Minute)

	// Уже включённое в брифинг событие больше не кандидат для этого пользователя.
	candidatesAfter, err := repo.ListCandidateEvents(ctx, userID, now.Add(-time.Hour), 10)
	require.NoError(t, err)
	require.Empty(t, candidatesAfter)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// internal/briefing/repository/postgres_test.go -> repo root
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}
