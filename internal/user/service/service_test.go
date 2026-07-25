package service_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/user/domain"
	"github.com/Shipovmax/Lumora/internal/user/service"
)

// fakeRepository — in-memory реализация domain.Repository для unit-тестов сервиса.
type fakeRepository struct {
	mu       sync.Mutex
	profiles map[string]domain.Profile
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{profiles: map[string]domain.Profile{}}
}

func (f *fakeRepository) GetProfile(_ context.Context, userID string) (domain.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if p, ok := f.profiles[userID]; ok {
		return p, nil
	}

	p := domain.Profile{UserID: userID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.profiles[userID] = p
	return p, nil
}

func (f *fakeRepository) UpdateProfile(_ context.Context, userID string, update domain.ProfileUpdate) (domain.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	existing, ok := f.profiles[userID]
	createdAt := time.Now()
	if ok {
		createdAt = existing.CreatedAt
	}

	p := domain.Profile{
		UserID:     userID,
		Name:       update.Name,
		Country:    update.Country,
		Language:   update.Language,
		Profession: update.Profession,
		Interests:  update.Interests,
		Topics:     update.Topics,
		CreatedAt:  createdAt,
		UpdatedAt:  time.Now(),
	}
	f.profiles[userID] = p
	return p, nil
}

func newTestService() *service.Service {
	return service.New(newFakeRepository())
}

func TestGetProfileCreatesDefault(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	profile, err := svc.GetProfile(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, "user-1", profile.UserID)
	require.Empty(t, profile.Name)
}

func TestUpdateProfileTrimsAndDropsEmptyListEntries(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	updated, err := svc.UpdateProfile(ctx, "user-1", domain.ProfileUpdate{
		Name:       "  Ada Lovelace  ",
		Country:    " UK ",
		Language:   " en ",
		Profession: " Mathematician ",
		Interests:  []string{" computing ", "", "  "},
		Topics:     []string{"ai", " "},
	})
	require.NoError(t, err)
	require.Equal(t, "Ada Lovelace", updated.Name)
	require.Equal(t, "UK", updated.Country)
	require.Equal(t, []string{"computing"}, updated.Interests)
	require.Equal(t, []string{"ai"}, updated.Topics)

	fetched, err := svc.GetProfile(ctx, "user-1")
	require.NoError(t, err)
	require.Equal(t, "Ada Lovelace", fetched.Name)
}

func TestUpdateProfileRejectsTooLongName(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	_, err := svc.UpdateProfile(ctx, "user-1", domain.ProfileUpdate{Name: strings.Repeat("a", 201)})
	require.ErrorIs(t, err, domain.ErrNameTooLong)
}
