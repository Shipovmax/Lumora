package service_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/source/domain"
	"github.com/Shipovmax/Lumora/internal/source/service"
)

// fakeRepository — in-memory реализация domain.Repository для unit-тестов сервиса.
type fakeRepository struct {
	mu      sync.Mutex
	sources map[string]domain.Source
	nextID  int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{sources: map[string]domain.Source{}}
}

func (f *fakeRepository) genID() string {
	f.nextID++
	return fmt.Sprintf("id-%d", f.nextID)
}

func (f *fakeRepository) CreateSource(_ context.Context, userID string, typ domain.Type, name, url string) (domain.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s := domain.Source{
		ID:        f.genID(),
		UserID:    userID,
		Type:      typ,
		Name:      name,
		URL:       url,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	f.sources[s.ID] = s
	return s, nil
}

func (f *fakeRepository) ListSources(_ context.Context, userID string) ([]domain.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []domain.Source
	for _, s := range f.sources {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeRepository) GetSource(_ context.Context, userID, id string) (domain.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, ok := f.sources[id]
	if !ok || s.UserID != userID {
		return domain.Source{}, domain.ErrSourceNotFound
	}
	return s, nil
}

func (f *fakeRepository) SetEnabled(_ context.Context, userID, id string, enabled bool) (domain.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, ok := f.sources[id]
	if !ok || s.UserID != userID {
		return domain.Source{}, domain.ErrSourceNotFound
	}
	s.Enabled = enabled
	f.sources[id] = s
	return s, nil
}

func (f *fakeRepository) DeleteSource(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, ok := f.sources[id]
	if !ok || s.UserID != userID {
		return domain.ErrSourceNotFound
	}
	delete(f.sources, id)
	return nil
}

func newTestService() *service.Service {
	return service.New(newFakeRepository())
}

func TestAddSourceValidation(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.AddSource(ctx, "user-1", domain.Type("carrier-pigeon"), "Feed", "https://example.com/rss")
	require.ErrorIs(t, err, domain.ErrInvalidType)

	_, err = svc.AddSource(ctx, "user-1", domain.TypeRSS, "  ", "https://example.com/rss")
	require.ErrorIs(t, err, domain.ErrNameRequired)

	_, err = svc.AddSource(ctx, "user-1", domain.TypeRSS, "Feed", "  ")
	require.ErrorIs(t, err, domain.ErrURLRequired)

	s, err := svc.AddSource(ctx, "user-1", domain.TypeRSS, "Feed", "https://example.com/rss")
	require.NoError(t, err)
	require.True(t, s.Enabled)
}

func TestListSourcesScopedToUser(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.AddSource(ctx, "user-1", domain.TypeRSS, "Feed 1", "https://example.com/1.xml")
	require.NoError(t, err)
	_, err = svc.AddSource(ctx, "user-2", domain.TypeYouTube, "Channel", "https://youtube.com/feeds/videos.xml?channel_id=x")
	require.NoError(t, err)

	list, err := svc.ListSources(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "Feed 1", list[0].Name)
}

func TestSetEnabledAndDelete(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	s, err := svc.AddSource(ctx, "user-1", domain.TypeTelegram, "Channel", "https://t.me/s/channel")
	require.NoError(t, err)

	disabled, err := svc.SetEnabled(ctx, "user-1", s.ID, false)
	require.NoError(t, err)
	require.False(t, disabled.Enabled)

	_, err = svc.SetEnabled(ctx, "other-user", s.ID, true)
	require.ErrorIs(t, err, domain.ErrSourceNotFound)

	require.NoError(t, svc.DeleteSource(ctx, "user-1", s.ID))

	err = svc.DeleteSource(ctx, "user-1", s.ID)
	require.ErrorIs(t, err, domain.ErrSourceNotFound)
}
