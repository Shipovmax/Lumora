package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/notification/domain"
	"github.com/Shipovmax/Lumora/internal/notification/service"
)

type fakeRepository struct {
	tokens    map[string][]domain.DeviceToken
	removed   []string
	listErr   error
	removeErr error
}

func (f *fakeRepository) RegisterDevice(_ context.Context, userID, platform, token string) (domain.DeviceToken, error) {
	dt := domain.DeviceToken{ID: "device-1", UserID: userID, Platform: platform, Token: token}
	if f.tokens == nil {
		f.tokens = map[string][]domain.DeviceToken{}
	}
	f.tokens[userID] = append(f.tokens[userID], dt)
	return dt, nil
}

func (f *fakeRepository) ListDeviceTokens(_ context.Context, userID string) ([]domain.DeviceToken, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.tokens[userID], nil
}

func (f *fakeRepository) RemoveDeviceToken(_ context.Context, token string) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	f.removed = append(f.removed, token)
	return nil
}

type fakeSender struct {
	sent    []string
	failFor map[string]error
}

func (f *fakeSender) Send(_ context.Context, token string, _ domain.PushMessage) error {
	if err, ok := f.failFor[token]; ok {
		return err
	}
	f.sent = append(f.sent, token)
	return nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRegisterDeviceRejectsEmptyToken(t *testing.T) {
	svc := service.New(&fakeRepository{}, &fakeSender{}, noopLogger())

	_, err := svc.RegisterDevice(context.Background(), "user-1", domain.PlatformAndroid, "")
	require.ErrorIs(t, err, domain.ErrTokenRequired)
}

func TestRegisterDeviceRejectsInvalidPlatform(t *testing.T) {
	svc := service.New(&fakeRepository{}, &fakeSender{}, noopLogger())

	_, err := svc.RegisterDevice(context.Background(), "user-1", "windows-phone", "token-1")
	require.ErrorIs(t, err, domain.ErrInvalidPlatform)
}

func TestRegisterDeviceStoresToken(t *testing.T) {
	repo := &fakeRepository{}
	svc := service.New(repo, &fakeSender{}, noopLogger())

	dt, err := svc.RegisterDevice(context.Background(), "user-1", domain.PlatformIOS, "token-1")
	require.NoError(t, err)
	require.Equal(t, "token-1", dt.Token)
	require.Len(t, repo.tokens["user-1"], 1)
}

func TestRemoveDeviceTokenRejectsTokenNotOwnedByUser(t *testing.T) {
	repo := &fakeRepository{tokens: map[string][]domain.DeviceToken{
		"user-1": {{Token: "token-1", Platform: domain.PlatformAndroid}},
	}}
	svc := service.New(repo, &fakeSender{}, noopLogger())

	err := svc.RemoveDeviceToken(context.Background(), "user-2", "token-1")
	require.ErrorIs(t, err, domain.ErrInvalidToken)
	require.Empty(t, repo.removed)
}

func TestRemoveDeviceTokenRemovesOwnedToken(t *testing.T) {
	repo := &fakeRepository{tokens: map[string][]domain.DeviceToken{
		"user-1": {{Token: "token-1", Platform: domain.PlatformAndroid}},
	}}
	svc := service.New(repo, &fakeSender{}, noopLogger())

	err := svc.RemoveDeviceToken(context.Background(), "user-1", "token-1")
	require.NoError(t, err)
	require.Equal(t, []string{"token-1"}, repo.removed)
}

func TestNotifyEventReturnsErrNoDeviceTokensWhenEmpty(t *testing.T) {
	svc := service.New(&fakeRepository{}, &fakeSender{}, noopLogger())

	err := svc.NotifyEvent(context.Background(), "user-1", domain.PushMessage{Title: "T"})
	require.ErrorIs(t, err, domain.ErrNoDeviceTokens)
}

func TestNotifyEventSendsToAllDevices(t *testing.T) {
	repo := &fakeRepository{tokens: map[string][]domain.DeviceToken{
		"user-1": {
			{Token: "token-1", Platform: domain.PlatformAndroid},
			{Token: "token-2", Platform: domain.PlatformIOS},
		},
	}}
	sender := &fakeSender{}
	svc := service.New(repo, sender, noopLogger())

	err := svc.NotifyEvent(context.Background(), "user-1", domain.PushMessage{Title: "T"})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"token-1", "token-2"}, sender.sent)
}

func TestNotifyEventRemovesInvalidTokenAndKeepsSendingToOthers(t *testing.T) {
	repo := &fakeRepository{tokens: map[string][]domain.DeviceToken{
		"user-1": {
			{Token: "stale-token", Platform: domain.PlatformAndroid},
			{Token: "good-token", Platform: domain.PlatformIOS},
		},
	}}
	sender := &fakeSender{failFor: map[string]error{"stale-token": domain.ErrInvalidToken}}
	svc := service.New(repo, sender, noopLogger())

	err := svc.NotifyEvent(context.Background(), "user-1", domain.PushMessage{Title: "T"})
	require.NoError(t, err)
	require.Equal(t, []string{"stale-token"}, repo.removed)
	require.Equal(t, []string{"good-token"}, sender.sent)
}

func TestNotifyEventDoesNotFailOnTransientSendError(t *testing.T) {
	repo := &fakeRepository{tokens: map[string][]domain.DeviceToken{
		"user-1": {{Token: "token-1", Platform: domain.PlatformAndroid}},
	}}
	sender := &fakeSender{failFor: map[string]error{"token-1": errors.New("fcm unavailable")}}
	svc := service.New(repo, sender, noopLogger())

	err := svc.NotifyEvent(context.Background(), "user-1", domain.PushMessage{Title: "T"})
	require.NoError(t, err)
	require.Empty(t, repo.removed)
}
