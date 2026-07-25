// Package service реализует бизнес-логику домена auth: регистрацию, вход,
// обновление и отзыв токенов. Зависит только от портов, объявленных в domain.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/Shipovmax/Lumora/internal/auth/domain"
)

const minPasswordLength = 8

type Service struct {
	repo            domain.Repository
	tokens          domain.TokenIssuer
	refreshTokenTTL time.Duration
}

func New(repo domain.Repository, tokens domain.TokenIssuer, refreshTokenTTL time.Duration) *Service {
	return &Service{repo: repo, tokens: tokens, refreshTokenTTL: refreshTokenTTL}
}

// AuthResult — результат операций, выдающих пару токенов.
type AuthResult struct {
	User                 domain.User
	AccessToken          string
	AccessTokenExpiresAt time.Time
	RefreshToken         string
}

func (s *Service) Register(ctx context.Context, email, password string) (AuthResult, error) {
	if err := validateCredentials(email, password); err != nil {
		return AuthResult{}, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, email, passwordHash)
	if err != nil {
		return AuthResult{}, err
	}

	return s.issueTokenPair(ctx, user)
}

func (s *Service) Login(ctx context.Context, email, password string) (AuthResult, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return AuthResult{}, domain.ErrInvalidCredentials
		}
		return AuthResult{}, err
	}

	if !comparePassword(user.PasswordHash, password) {
		return AuthResult{}, domain.ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, user)
}

// Refresh обменивает валидный refresh-токен на новую пару токенов, отзывая старый
// refresh-токен (ротация — предотвращает повторное использование украденного токена).
func (s *Service) Refresh(ctx context.Context, refreshToken string) (AuthResult, error) {
	stored, err := s.repo.GetRefreshTokenByHash(ctx, hashToken(refreshToken))
	if err != nil {
		return AuthResult{}, err
	}

	if !stored.Valid(time.Now()) {
		return AuthResult{}, domain.ErrRefreshTokenInvalid
	}

	if err := s.repo.RevokeRefreshToken(ctx, stored.ID); err != nil {
		return AuthResult{}, err
	}

	user, err := s.repo.GetUserByID(ctx, stored.UserID)
	if err != nil {
		return AuthResult{}, err
	}

	return s.issueTokenPair(ctx, user)
}

// Logout отзывает refresh-токен. Идемпотентна: неизвестный/уже отозванный токен не ошибка.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	stored, err := s.repo.GetRefreshTokenByHash(ctx, hashToken(refreshToken))
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenInvalid) {
			return nil
		}
		return err
	}

	return s.repo.RevokeRefreshToken(ctx, stored.ID)
}

func (s *Service) Me(ctx context.Context, userID string) (domain.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

func (s *Service) issueTokenPair(ctx context.Context, user domain.User) (AuthResult, error) {
	accessToken, expiresAt, err := s.tokens.IssueAccessToken(user.ID)
	if err != nil {
		return AuthResult{}, fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, err := newOpaqueToken()
	if err != nil {
		return AuthResult{}, err
	}

	if _, err := s.repo.CreateRefreshToken(ctx, user.ID, hashToken(refreshToken), time.Now().Add(s.refreshTokenTTL)); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		User:                 user,
		AccessToken:          accessToken,
		AccessTokenExpiresAt: expiresAt,
		RefreshToken:         refreshToken,
	}, nil
}

func validateCredentials(email, password string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return domain.ErrInvalidEmail
	}
	if len(password) < minPasswordLength {
		return domain.ErrWeakPassword
	}
	return nil
}
