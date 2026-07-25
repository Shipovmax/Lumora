package domain

import (
	"context"
	"time"
)

// Repository — порт доступа к данным домена briefing. Реализуется в
// internal/briefing/repository поверх Postgres/sqlc.
type Repository interface {
	// ListCandidateEvents возвращает события, релевантные пользователю через его
	// источники (sources.user_id), обновлявшиеся не раньше since и ещё не
	// включённые ни в один предыдущий брифинг этого пользователя — отсортированные
	// по важности, не более limit штук.
	ListCandidateEvents(ctx context.Context, userID string, since time.Time, limit int) ([]CandidateEvent, error)
	// CreateBriefing создаёт брифинг и привязывает к нему события (в переданном
	// порядке = rank) одной транзакцией.
	CreateBriefing(ctx context.Context, userID string, typ Type, eventIDs []string) (id string, generatedAt time.Time, err error)
	// ListActiveUserIDs возвращает ID пользователей, у которых есть хотя бы один
	// источник — кандидаты на диспетчеризацию briefing:build планировщиком
	// (без источников брифинг не соберёт ни одного события).
	ListActiveUserIDs(ctx context.Context) ([]string, error)
}
