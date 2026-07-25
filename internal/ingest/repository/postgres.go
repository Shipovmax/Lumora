// Package repository — адаптер домена ingest к PostgreSQL поверх кода,
// сгенерированного sqlc (см. sqlc/gen). Реализует domain.Repository.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shipovmax/Lumora/internal/ingest/domain"
	sqlcgen "github.com/Shipovmax/Lumora/internal/ingest/repository/sqlc/gen"
)

type Repository struct {
	q *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(pool)}
}

var _ domain.Repository = (*Repository)(nil)

// SaveNewPosts вставляет публикации по одной: ON CONFLICT DO NOTHING в sqlc-запросе
// возвращает pgx.ErrNoRows для дублей (source_id+external_id уже существует) — такие
// пропускаются молча, это ожидаемая дедупликация, а не ошибка.
func (r *Repository) SaveNewPosts(ctx context.Context, posts []domain.Post) ([]domain.Post, error) {
	saved := make([]domain.Post, 0, len(posts))

	for _, p := range posts {
		sourceID, err := parseUUID(p.SourceID)
		if err != nil {
			continue
		}

		row, err := r.q.InsertPostIgnoreDuplicate(ctx, sqlcgen.InsertPostIgnoreDuplicateParams{
			SourceID:    sourceID,
			ExternalID:  p.ExternalID,
			Title:       p.Title,
			Url:         p.URL,
			Content:     p.Content,
			PublishedAt: toTimestamptz(p.PublishedAt),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return saved, err
		}
		saved = append(saved, toDomainPost(row))
	}

	return saved, nil
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

func toDomainPost(p sqlcgen.Post) domain.Post {
	return domain.Post{
		ID:          uuidString(p.ID),
		SourceID:    uuidString(p.SourceID),
		ExternalID:  p.ExternalID,
		Title:       p.Title,
		URL:         p.Url,
		Content:     p.Content,
		PublishedAt: p.PublishedAt.Time,
		CreatedAt:   p.CreatedAt.Time,
	}
}
