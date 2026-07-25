// Package repository — адаптер домена notification к PostgreSQL поверх кода,
// сгенерированного sqlc (см. sqlc/gen). Реализует domain.Repository.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shipovmax/Lumora/internal/notification/domain"
	sqlcgen "github.com/Shipovmax/Lumora/internal/notification/repository/sqlc/gen"
)

type Repository struct {
	q *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(pool)}
}

var _ domain.Repository = (*Repository)(nil)

func (r *Repository) RegisterDevice(ctx context.Context, userID, platform, token string) (domain.DeviceToken, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return domain.DeviceToken{}, err
	}

	d, err := r.q.RegisterDevice(ctx, sqlcgen.RegisterDeviceParams{UserID: uid, Platform: platform, Token: token})
	if err != nil {
		return domain.DeviceToken{}, err
	}
	return toDomainDeviceToken(d), nil
}

func (r *Repository) ListDeviceTokens(ctx context.Context, userID string) ([]domain.DeviceToken, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	rows, err := r.q.ListDeviceTokens(ctx, uid)
	if err != nil {
		return nil, err
	}

	tokens := make([]domain.DeviceToken, 0, len(rows))
	for _, row := range rows {
		tokens = append(tokens, toDomainDeviceToken(row))
	}
	return tokens, nil
}

func (r *Repository) RemoveDeviceToken(ctx context.Context, token string) error {
	return r.q.RemoveDeviceToken(ctx, token)
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

func toDomainDeviceToken(d sqlcgen.DeviceToken) domain.DeviceToken {
	return domain.DeviceToken{
		ID:        uuidString(d.ID),
		UserID:    uuidString(d.UserID),
		Platform:  d.Platform,
		Token:     d.Token,
		CreatedAt: d.CreatedAt.Time,
		UpdatedAt: d.UpdatedAt.Time,
	}
}
