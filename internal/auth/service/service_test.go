package service_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/auth/domain"
	"github.com/Shipovmax/Lumora/internal/auth/service"
)

// fakeRepository — in-memory реализация domain.Repository для unit-тестов сервиса.
type fakeRepository struct {
	mu            sync.Mutex
	usersByID     map[string]domain.User
	usersByEmail  map[string]string // email -> id
	refreshTokens map[string]domain.RefreshToken
	nextID        int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		usersByID:     map[string]domain.User{},
		usersByEmail:  map[string]string{},
		refreshTokens: map[string]domain.RefreshToken{},
	}
}

func (f *fakeRepository) genID() string {
	f.nextID++
	return fmt.Sprintf("id-%d", f.nextID)
}

func (f *fakeRepository) CreateUser(_ context.Context, email, passwordHash string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.usersByEmail[email]; ok {
		return domain.User{}, domain.ErrEmailTaken
	}

	u := domain.User{
		ID:           f.genID(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	f.usersByID[u.ID] = u
	f.usersByEmail[email] = u.ID
	return u, nil
}

func (f *fakeRepository) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id, ok := f.usersByEmail[email]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return f.usersByID[id], nil
}

func (f *fakeRepository) GetUserByID(_ context.Context, id string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.usersByID[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeRepository) CreateRefreshToken(_ context.Context, userID, tokenHash string, expiresAt time.Time) (domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	rt := domain.RefreshToken{
		ID:        f.genID(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	f.refreshTokens[tokenHash] = rt
	return rt, nil
}

func (f *fakeRepository) GetRefreshTokenByHash(_ context.Context, tokenHash string) (domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	rt, ok := f.refreshTokens[tokenHash]
	if !ok {
		return domain.RefreshToken{}, domain.ErrRefreshTokenInvalid
	}
	return rt, nil
}

func (f *fakeRepository) RevokeRefreshToken(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for hash, rt := range f.refreshTokens {
		if rt.ID == id {
			now := time.Now()
			rt.RevokedAt = &now
			f.refreshTokens[hash] = rt
			return nil
		}
	}
	return nil
}

// fakeTokenIssuer выпускает access-токены как обычную строку "access-for-<userID>"
// без реальной криптографии — этого достаточно, чтобы проверить бизнес-логику сервиса.
type fakeTokenIssuer struct{}

func (fakeTokenIssuer) IssueAccessToken(userID string) (string, time.Time, error) {
	return "access-for-" + userID, time.Now().Add(15 * time.Minute), nil
}

func (fakeTokenIssuer) ParseAccessToken(token string) (string, error) {
	const prefix = "access-for-"
	if len(token) <= len(prefix) || token[:len(prefix)] != prefix {
		return "", errors.New("invalid token")
	}
	return token[len(prefix):], nil
}

func newTestService() *service.Service {
	return service.New(newFakeRepository(), fakeTokenIssuer{}, 720*time.Hour)
}

func TestRegister(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	result, err := svc.Register(ctx, "user@example.com", "password123")
	require.NoError(t, err)
	require.Equal(t, "user@example.com", result.User.Email)
	require.NotEmpty(t, result.AccessToken)
	require.NotEmpty(t, result.RefreshToken)

	_, err = svc.Register(ctx, "user@example.com", "password123")
	require.ErrorIs(t, err, domain.ErrEmailTaken)

	_, err = svc.Register(ctx, "not-an-email", "password123")
	require.ErrorIs(t, err, domain.ErrInvalidEmail)

	_, err = svc.Register(ctx, "another@example.com", "short")
	require.ErrorIs(t, err, domain.ErrWeakPassword)
}

func TestLogin(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.Register(ctx, "user@example.com", "password123")
	require.NoError(t, err)

	result, err := svc.Login(ctx, "user@example.com", "password123")
	require.NoError(t, err)
	require.Equal(t, "user@example.com", result.User.Email)

	_, err = svc.Login(ctx, "user@example.com", "wrong-password")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)

	_, err = svc.Login(ctx, "unknown@example.com", "password123")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestRefreshRotatesToken(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	reg, err := svc.Register(ctx, "user@example.com", "password123")
	require.NoError(t, err)

	refreshed, err := svc.Refresh(ctx, reg.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, reg.RefreshToken, refreshed.RefreshToken, "refresh token must rotate")

	// Старый refresh-токен отозван и больше не может быть использован повторно.
	_, err = svc.Refresh(ctx, reg.RefreshToken)
	require.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)
}

func TestLogoutIsIdempotent(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	reg, err := svc.Register(ctx, "user@example.com", "password123")
	require.NoError(t, err)

	require.NoError(t, svc.Logout(ctx, reg.RefreshToken))
	// Повторный logout тем же (уже отозванным) токеном не должен возвращать ошибку.
	require.NoError(t, svc.Logout(ctx, reg.RefreshToken))

	_, err = svc.Refresh(ctx, reg.RefreshToken)
	require.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)
}

func TestMe(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	reg, err := svc.Register(ctx, "user@example.com", "password123")
	require.NoError(t, err)

	user, err := svc.Me(ctx, reg.User.ID)
	require.NoError(t, err)
	require.Equal(t, "user@example.com", user.Email)

	_, err = svc.Me(ctx, "unknown-id")
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}
