package domain

import (
	"context"
	"time"
)

// Repository — порт доступа к данным домена auth. Реализуется в
// internal/auth/repository поверх Postgres/sqlc.
type Repository interface {
	CreateUser(ctx context.Context, email, passwordHash string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)

	CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string) error
}

// TokenIssuer — порт выпуска/валидации access-токенов. Реализуется
// internal/platform/jwtauth.Issuer.
type TokenIssuer interface {
	IssueAccessToken(userID string) (token string, expiresAt time.Time, err error)
	ParseAccessToken(token string) (userID string, err error)
}
