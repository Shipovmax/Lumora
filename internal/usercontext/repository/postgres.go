// Package repository — адаптер домена usercontext к PostgreSQL поверх кода,
// сгенерированного sqlc (см. sqlc/gen). Реализует domain.Repository.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shipovmax/Lumora/internal/usercontext/domain"
	sqlcgen "github.com/Shipovmax/Lumora/internal/usercontext/repository/sqlc/gen"
)

type Repository struct {
	q *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(pool)}
}

var _ domain.Repository = (*Repository)(nil)

func (r *Repository) GetContext(ctx context.Context, userID string) (domain.Context, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return domain.Context{}, domain.ErrContextNotFound
	}

	c, err := r.q.GetOrCreateContext(ctx, uid)
	if err != nil {
		return domain.Context{}, err
	}
	return toDomainContext(c), nil
}

func (r *Repository) UpdateContext(ctx context.Context, userID, content string) (domain.Context, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return domain.Context{}, domain.ErrContextNotFound
	}

	c, err := r.q.UpsertContext(ctx, sqlcgen.UpsertContextParams{UserID: uid, Content: content})
	if err != nil {
		return domain.Context{}, err
	}
	return toDomainContext(c), nil
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

func toDomainContext(c sqlcgen.UserContext) domain.Context {
	return domain.Context{
		UserID:    uuidString(c.UserID),
		Content:   c.Content,
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}
}
