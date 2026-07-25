// Package repository — адаптер домена auth к PostgreSQL поверх кода,
// сгенерированного sqlc (см. sqlc/gen). Реализует domain.Repository.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shipovmax/Lumora/internal/auth/domain"
	sqlcgen "github.com/Shipovmax/Lumora/internal/auth/repository/sqlc/gen"
)

const uniqueViolationCode = "23505"

type Repository struct {
	q *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(pool)}
}

var _ domain.Repository = (*Repository)(nil)

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (domain.User, error) {
	u, err := r.q.CreateUser(ctx, sqlcgen.CreateUserParams{Email: email, PasswordHash: passwordHash})
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, domain.ErrEmailTaken
		}
		return domain.User{}, err
	}
	return toDomainUser(u), nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}
	return toDomainUser(u), nil
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return domain.User{}, domain.ErrUserNotFound
	}

	u, err := r.q.GetUserByID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		return domain.User{}, err
	}
	return toDomainUser(u), nil
}

func (r *Repository) CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (domain.RefreshToken, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return domain.RefreshToken{}, domain.ErrUserNotFound
	}

	rt, err := r.q.CreateRefreshToken(ctx, sqlcgen.CreateRefreshTokenParams{
		UserID:    uid,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return domain.RefreshToken{}, err
	}
	return toDomainRefreshToken(rt), nil
}

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	rt, err := r.q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RefreshToken{}, domain.ErrRefreshTokenInvalid
		}
		return domain.RefreshToken{}, err
	}
	return toDomainRefreshToken(rt), nil
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, id string) error {
	uid, err := parseUUID(id)
	if err != nil {
		return domain.ErrRefreshTokenInvalid
	}
	return r.q.RevokeRefreshToken(ctx, uid)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
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

func toDomainUser(u sqlcgen.User) domain.User {
	return domain.User{
		ID:           uuidString(u.ID),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

func toDomainRefreshToken(rt sqlcgen.RefreshToken) domain.RefreshToken {
	var revokedAt *time.Time
	if rt.RevokedAt.Valid {
		t := rt.RevokedAt.Time
		revokedAt = &t
	}

	return domain.RefreshToken{
		ID:        uuidString(rt.ID),
		UserID:    uuidString(rt.UserID),
		TokenHash: rt.TokenHash,
		ExpiresAt: rt.ExpiresAt.Time,
		RevokedAt: revokedAt,
		CreatedAt: rt.CreatedAt.Time,
	}
}
