// Package repository — адаптер домена briefing к PostgreSQL поверх кода,
// сгенерированного sqlc (см. sqlc/gen). Реализует domain.Repository.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shipovmax/Lumora/internal/briefing/domain"
	sqlcgen "github.com/Shipovmax/Lumora/internal/briefing/repository/sqlc/gen"
)

type Repository struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: sqlcgen.New(pool)}
}

var _ domain.Repository = (*Repository)(nil)

func (r *Repository) ListCandidateEvents(ctx context.Context, userID string, since time.Time, limit int) ([]domain.CandidateEvent, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id %q: %w", userID, err)
	}

	rows, err := r.q.ListCandidateEvents(ctx, sqlcgen.ListCandidateEventsParams{
		UserID:     uid,
		LastSeenAt: toTimestamptz(since),
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, err
	}

	events := make([]domain.CandidateEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, domain.CandidateEvent{
			ID:         uuidString(row.ID),
			Topic:      row.Topic,
			Title:      row.Title,
			Importance: int(row.Importance),
		})
	}
	return events, nil
}

// CreateBriefing создаёт брифинг и привязывает к нему события одной
// транзакцией — без неё возможен брифинг без событий при сбое между запросами.
func (r *Repository) CreateBriefing(ctx context.Context, userID string, typ domain.Type, eventIDs []string) (string, time.Time, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid user id %q: %w", userID, err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	defer tx.Rollback(ctx)

	q := r.q.WithTx(tx)

	briefing, err := q.CreateBriefing(ctx, sqlcgen.CreateBriefingParams{UserID: uid, Type: string(typ)})
	if err != nil {
		return "", time.Time{}, err
	}

	for i, eventID := range eventIDs {
		eid, err := parseUUID(eventID)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("invalid event id %q: %w", eventID, err)
		}

		if err := q.AddBriefingEvent(ctx, sqlcgen.AddBriefingEventParams{
			BriefingID: briefing.ID,
			EventID:    eid,
			Rank:       int32(i + 1),
		}); err != nil {
			return "", time.Time{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", time.Time{}, err
	}

	return uuidString(briefing.ID), briefing.GeneratedAt.Time, nil
}

func (r *Repository) ListActiveUserIDs(ctx context.Context) ([]string, error) {
	rows, err := r.q.ListActiveUserIDs(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, id := range rows {
		ids = append(ids, uuidString(id))
	}
	return ids, nil
}

func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

func uuidString(id pgtype.UUID) string {
	v, err := id.Value()
	if err != nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}
