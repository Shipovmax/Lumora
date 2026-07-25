// Package repository — адаптер домена user к PostgreSQL поверх кода,
// сгенерированного sqlc (см. sqlc/gen). Реализует domain.Repository.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shipovmax/Lumora/internal/user/domain"
	sqlcgen "github.com/Shipovmax/Lumora/internal/user/repository/sqlc/gen"
)

type Repository struct {
	q *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(pool)}
}

var _ domain.Repository = (*Repository)(nil)

func (r *Repository) GetProfile(ctx context.Context, userID string) (domain.Profile, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return domain.Profile{}, domain.ErrProfileNotFound
	}

	p, err := r.q.GetOrCreateProfile(ctx, uid)
	if err != nil {
		return domain.Profile{}, err
	}
	return toDomainProfile(p), nil
}

func (r *Repository) UpdateProfile(ctx context.Context, userID string, update domain.ProfileUpdate) (domain.Profile, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return domain.Profile{}, domain.ErrProfileNotFound
	}

	p, err := r.q.UpsertProfile(ctx, sqlcgen.UpsertProfileParams{
		UserID:     uid,
		Name:       update.Name,
		Country:    update.Country,
		Language:   update.Language,
		Profession: update.Profession,
		Interests:  update.Interests,
		Topics:     update.Topics,
	})
	if err != nil {
		return domain.Profile{}, err
	}
	return toDomainProfile(p), nil
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

func toDomainProfile(p sqlcgen.UserProfile) domain.Profile {
	return domain.Profile{
		UserID:     uuidString(p.UserID),
		Name:       p.Name,
		Country:    p.Country,
		Language:   p.Language,
		Profession: p.Profession,
		Interests:  p.Interests,
		Topics:     p.Topics,
		CreatedAt:  p.CreatedAt.Time,
		UpdatedAt:  p.UpdatedAt.Time,
	}
}
