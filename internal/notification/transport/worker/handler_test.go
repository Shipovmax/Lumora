package notificationworker_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/notification/domain"
	"github.com/Shipovmax/Lumora/internal/notification/service"
	notificationworker "github.com/Shipovmax/Lumora/internal/notification/transport/worker"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
)

type fakeRepository struct {
	tokens map[string][]domain.DeviceToken
}

func (f *fakeRepository) RegisterDevice(context.Context, string, string, string) (domain.DeviceToken, error) {
	return domain.DeviceToken{}, nil
}

func (f *fakeRepository) ListDeviceTokens(_ context.Context, userID string) ([]domain.DeviceToken, error) {
	return f.tokens[userID], nil
}

func (f *fakeRepository) RemoveDeviceToken(context.Context, string) error { return nil }

type fakeSender struct {
	sent []string
	err  error
}

func (f *fakeSender) Send(_ context.Context, token string, _ domain.PushMessage) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, token)
	return nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newPushTask(t *testing.T, userID, title, body string) *asynq.Task {
	t.Helper()
	b, err := json.Marshal(queue.NotificationPushPayload{UserID: userID, Title: title, Body: body})
	require.NoError(t, err)
	return asynq.NewTask(queue.TypeNotificationPush, b)
}

func TestHandlePushRejectsInvalidPayload(t *testing.T) {
	h := notificationworker.NewHandler(nil, noopLogger())

	err := h.HandlePush(context.Background(), asynq.NewTask(queue.TypeNotificationPush, []byte("not json")))
	require.Error(t, err)
}

func TestHandlePushReturnsNilWhenUserHasNoDevices(t *testing.T) {
	svc := service.New(&fakeRepository{}, &fakeSender{}, noopLogger())
	h := notificationworker.NewHandler(svc, noopLogger())

	err := h.HandlePush(context.Background(), newPushTask(t, "user-1", "T", "B"))
	require.NoError(t, err, "ErrNoDeviceTokens must not fail the task")
}

func TestHandlePushSendsToAllDevices(t *testing.T) {
	repo := &fakeRepository{tokens: map[string][]domain.DeviceToken{
		"user-1": {{Token: "token-1"}, {Token: "token-2"}},
	}}
	sender := &fakeSender{}
	svc := service.New(repo, sender, noopLogger())
	h := notificationworker.NewHandler(svc, noopLogger())

	err := h.HandlePush(context.Background(), newPushTask(t, "user-1", "Title", "Body"))
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"token-1", "token-2"}, sender.sent)
}

func TestHandlePushPropagatesUnexpectedError(t *testing.T) {
	repo := &fakeRepository{tokens: map[string][]domain.DeviceToken{
		"user-1": {{Token: "token-1"}},
	}}
	// A repository failure (not a Sender error) should surface as a real
	// task failure, unlike ErrNoDeviceTokens.
	svc := service.New(&erroringRepository{repo}, &fakeSender{}, noopLogger())
	h := notificationworker.NewHandler(svc, noopLogger())

	err := h.HandlePush(context.Background(), newPushTask(t, "user-1", "T", "B"))
	require.Error(t, err)
}

type erroringRepository struct {
	*fakeRepository
}

func (r *erroringRepository) ListDeviceTokens(context.Context, string) ([]domain.DeviceToken, error) {
	return nil, errors.New("db unavailable")
}
