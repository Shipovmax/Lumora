// Package repository — адаптер домена pipeline к PostgreSQL поверх кода,
// сгенерированного sqlc (см. sqlc/gen). Реализует domain.Repository.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shipovmax/Lumora/internal/pipeline/domain"
	sqlcgen "github.com/Shipovmax/Lumora/internal/pipeline/repository/sqlc/gen"
)

type Repository struct {
	pool *pgxpool.Pool
	q    *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: sqlcgen.New(pool)}
}

var _ domain.Repository = (*Repository)(nil)

func (r *Repository) GetPosts(ctx context.Context, postIDs []string) ([]domain.PostRef, error) {
	ids := make([]pgtype.UUID, 0, len(postIDs))
	for _, id := range postIDs {
		uid, err := parseUUID(id)
		if err != nil {
			continue
		}
		ids = append(ids, uid)
	}

	rows, err := r.q.GetPostsByID(ctx, ids)
	if err != nil {
		return nil, err
	}

	posts := make([]domain.PostRef, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, domain.PostRef{
			ID:          uuidString(row.ID),
			SourceID:    uuidString(row.SourceID),
			Title:       row.Title,
			Content:     row.Content,
			PublishedAt: row.PublishedAt.Time,
		})
	}
	return posts, nil
}

func (r *Repository) ListRecentEvents(ctx context.Context, since time.Time) ([]domain.Event, error) {
	rows, err := r.q.ListRecentEvents(ctx, toTimestamptz(since))
	if err != nil {
		return nil, err
	}

	events := make([]domain.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, toDomainEvent(row))
	}
	return events, nil
}

// CreateEventWithPost создаёт событие и привязывает к нему post одной транзакцией:
// без неё возможно рассинхронизированное состояние (событие есть, публикация ещё
// не привязана), если процесс упадёт между двумя запросами.
func (r *Repository) CreateEventWithPost(ctx context.Context, topic domain.Topic, title, matchText, postID string, publishedAt time.Time) (domain.Event, error) {
	postUID, err := parseUUID(postID)
	if err != nil {
		return domain.Event{}, fmt.Errorf("invalid post id %q: %w", postID, err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Event{}, err
	}
	defer tx.Rollback(ctx)

	q := r.q.WithTx(tx)

	event, err := q.CreateEvent(ctx, sqlcgen.CreateEventParams{
		Topic:       string(topic),
		Title:       title,
		MatchText:   matchText,
		FirstSeenAt: toTimestamptz(publishedAt),
	})
	if err != nil {
		return domain.Event{}, err
	}

	if err := q.AssignPostToEvent(ctx, sqlcgen.AssignPostToEventParams{EventID: event.ID, ID: postUID}); err != nil {
		return domain.Event{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Event{}, err
	}

	return toDomainEvent(event), nil
}

// AttachPost присоединяет post к существующему event и пересчитывает его
// статистику одной транзакцией (см. CreateEventWithPost).
func (r *Repository) AttachPost(ctx context.Context, eventID, postID, matchText string, publishedAt time.Time) (domain.Event, error) {
	eventUID, err := parseUUID(eventID)
	if err != nil {
		return domain.Event{}, fmt.Errorf("invalid event id %q: %w", eventID, err)
	}
	postUID, err := parseUUID(postID)
	if err != nil {
		return domain.Event{}, fmt.Errorf("invalid post id %q: %w", postID, err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Event{}, err
	}
	defer tx.Rollback(ctx)

	q := r.q.WithTx(tx)

	if err := q.AssignPostToEvent(ctx, sqlcgen.AssignPostToEventParams{EventID: eventUID, ID: postUID}); err != nil {
		return domain.Event{}, err
	}

	event, err := q.RecomputeEventStats(ctx, sqlcgen.RecomputeEventStatsParams{
		ID:         eventUID,
		LastSeenAt: toTimestamptz(publishedAt),
		MatchText:  matchText,
	})
	if err != nil {
		return domain.Event{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Event{}, err
	}

	return toDomainEvent(event), nil
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

func toDomainEvent(e sqlcgen.Event) domain.Event {
	return domain.Event{
		ID:          uuidString(e.ID),
		Topic:       domain.Topic(e.Topic),
		Title:       e.Title,
		MatchText:   e.MatchText,
		Importance:  int(e.Importance),
		SourceCount: int(e.SourceCount),
		FirstSeenAt: e.FirstSeenAt.Time,
		LastSeenAt:  e.LastSeenAt.Time,
		CreatedAt:   e.CreatedAt.Time,
		UpdatedAt:   e.UpdatedAt.Time,
	}
}
