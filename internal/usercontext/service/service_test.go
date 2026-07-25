package service_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/usercontext/domain"
	"github.com/Shipovmax/Lumora/internal/usercontext/service"
)

// fakeRepository — in-memory реализация domain.Repository для unit-тестов сервиса.
type fakeRepository struct {
	mu       sync.Mutex
	contexts map[string]domain.Context
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{contexts: map[string]domain.Context{}}
}

func (f *fakeRepository) GetContext(_ context.Context, userID string) (domain.Context, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if c, ok := f.contexts[userID]; ok {
		return c, nil
	}

	c := domain.Context{UserID: userID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.contexts[userID] = c
	return c, nil
}

func (f *fakeRepository) UpdateContext(_ context.Context, userID, content string) (domain.Context, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	existing, ok := f.contexts[userID]
	createdAt := time.Now()
	if ok {
		createdAt = existing.CreatedAt
	}

	c := domain.Context{UserID: userID, Content: content, CreatedAt: createdAt, UpdatedAt: time.Now()}
	f.contexts[userID] = c
	return c, nil
}

func newTestService() *service.Service {
	return service.New(newFakeRepository())
}

func TestGetContextCreatesDefault(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	c, err := svc.GetContext(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, "user-1", c.UserID)
	require.Empty(t, c.Content)
}

func TestUpdateContextTrimsAndPersists(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	updated, err := svc.UpdateContext(ctx, "user-1", "  interested in deep tech  ")
	require.NoError(t, err)
	require.Equal(t, "interested in deep tech", updated.Content)

	fetched, err := svc.GetContext(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, "interested in deep tech", fetched.Content)
}

func TestUpdateContextRejectsTooLongContent(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.UpdateContext(ctx, "user-1", strings.Repeat("a", 4001))
	require.ErrorIs(t, err, domain.ErrContextTooLong)
}
