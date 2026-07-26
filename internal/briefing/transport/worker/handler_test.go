package briefingworker_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"

	aidomain "github.com/Shipovmax/Lumora/internal/ai/domain"
	"github.com/Shipovmax/Lumora/internal/briefing/domain"
	"github.com/Shipovmax/Lumora/internal/briefing/service"
	briefingworker "github.com/Shipovmax/Lumora/internal/briefing/transport/worker"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
)

// fakeRepository — same shape as internal/briefing/service/service_test.go's,
// redeclared here since it's a different test package.
type fakeRepository struct {
	candidates      []domain.CandidateEvent
	createBriefing  string
	activeUserIDs   []string
	createdEventIDs []string
}

func (f *fakeRepository) ListCandidateEvents(_ context.Context, _ string, _ time.Time, limit int) ([]domain.CandidateEvent, error) {
	if len(f.candidates) > limit {
		return f.candidates[:limit], nil
	}
	return f.candidates, nil
}

func (f *fakeRepository) CreateBriefing(_ context.Context, _ string, _ domain.Type, eventIDs []string) (string, time.Time, error) {
	f.createdEventIDs = eventIDs
	return "briefing-1", time.Now(), nil
}

func (f *fakeRepository) ListActiveUserIDs(_ context.Context) ([]string, error) {
	return f.activeUserIDs, nil
}

type fakeExplanationRepository struct {
	explanations map[string]aidomain.Explanation
}

func (f *fakeExplanationRepository) GetExplanation(_ context.Context, eventID, userID string) (aidomain.Explanation, error) {
	e, ok := f.explanations[eventID+":"+userID]
	if !ok {
		return aidomain.Explanation{}, aidomain.ErrExplanationNotFound
	}
	return e, nil
}

type fakeGenerator struct{}

func (fakeGenerator) GenerateExplanation(_ context.Context, eventID, userID string) (aidomain.Explanation, error) {
	return aidomain.Explanation{EventID: eventID, UserID: userID, WhatHappened: "generated explanation"}, nil
}

type fakeEnqueuer struct {
	tasks []*asynq.Task
	err   error
}

func (f *fakeEnqueuer) Enqueue(task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.tasks = append(f.tasks, task)
	return &asynq.TaskInfo{}, nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newBuildTask(t *testing.T, userID, typ string) *asynq.Task {
	t.Helper()
	b, err := json.Marshal(queue.BriefingBuildPayload{UserID: userID, Type: typ})
	require.NoError(t, err)
	return asynq.NewTask(queue.TypeBriefingBuild, b)
}

func TestHandleBuildRejectsInvalidPayload(t *testing.T) {
	h := briefingworker.NewHandler(nil, &fakeEnqueuer{}, noopLogger())

	err := h.HandleBuild(context.Background(), asynq.NewTask(queue.TypeBriefingBuild, []byte("not json")))
	require.Error(t, err)
}

func TestHandleBuildReturnsNilWhenNoRelevantEvents(t *testing.T) {
	svc := service.New(&fakeRepository{}, &fakeExplanationRepository{}, fakeGenerator{}, noopLogger())
	h := briefingworker.NewHandler(svc, &fakeEnqueuer{}, noopLogger())

	err := h.HandleBuild(context.Background(), newBuildTask(t, "user-1", "morning"))
	require.NoError(t, err, "ErrNoRelevantEvents must not fail the task")
}

func TestHandleBuildEnqueuesPushOnlyForImportantEvents(t *testing.T) {
	repo := &fakeRepository{candidates: []domain.CandidateEvent{
		{ID: "event-important", Title: "Big news", Importance: 80},
		{ID: "event-minor", Title: "Small news", Importance: 20},
	}}
	svc := service.New(repo, &fakeExplanationRepository{}, fakeGenerator{}, noopLogger())
	enqueuer := &fakeEnqueuer{}
	h := briefingworker.NewHandler(svc, enqueuer, noopLogger())

	err := h.HandleBuild(context.Background(), newBuildTask(t, "user-1", "morning"))
	require.NoError(t, err)

	require.Len(t, enqueuer.tasks, 1, "only the event at/above the importance threshold should trigger a push")
	require.Equal(t, queue.TypeNotificationPush, enqueuer.tasks[0].Type())

	var payload queue.NotificationPushPayload
	require.NoError(t, json.Unmarshal(enqueuer.tasks[0].Payload(), &payload))
	require.Equal(t, "user-1", payload.UserID)
	require.Equal(t, "Big news", payload.Title)
}

func TestHandleBuildSucceedsEvenIfPushEnqueueFails(t *testing.T) {
	repo := &fakeRepository{candidates: []domain.CandidateEvent{
		{ID: "event-important", Title: "Big news", Importance: 100},
	}}
	svc := service.New(repo, &fakeExplanationRepository{}, fakeGenerator{}, noopLogger())
	enqueuer := &fakeEnqueuer{err: errors.New("redis down")}
	h := briefingworker.NewHandler(svc, enqueuer, noopLogger())

	err := h.HandleBuild(context.Background(), newBuildTask(t, "user-1", "morning"))
	require.NoError(t, err, "briefing is already saved, push can be retried/enqueued later")
}

func newDispatchTask(t *testing.T, typ string) *asynq.Task {
	t.Helper()
	b, err := json.Marshal(queue.BriefingDispatchPayload{Type: typ})
	require.NoError(t, err)
	return asynq.NewTask(queue.TypeBriefingDispatch, b)
}

func TestDispatchHandlerRejectsInvalidPayload(t *testing.T) {
	h := briefingworker.NewDispatchHandler(&fakeRepository{}, &fakeEnqueuer{}, noopLogger())

	err := h.HandleDispatch(context.Background(), asynq.NewTask(queue.TypeBriefingDispatch, []byte("not json")))
	require.Error(t, err)
}

func TestDispatchHandlerFansOutToAllActiveUsers(t *testing.T) {
	repo := &fakeRepository{activeUserIDs: []string{"user-1", "user-2"}}
	enqueuer := &fakeEnqueuer{}
	h := briefingworker.NewDispatchHandler(repo, enqueuer, noopLogger())

	err := h.HandleDispatch(context.Background(), newDispatchTask(t, "morning"))
	require.NoError(t, err)
	require.Len(t, enqueuer.tasks, 2)

	for _, task := range enqueuer.tasks {
		require.Equal(t, queue.TypeBriefingBuild, task.Type())
		var payload queue.BriefingBuildPayload
		require.NoError(t, json.Unmarshal(task.Payload(), &payload))
		require.Equal(t, "morning", payload.Type)
	}
}

func TestDispatchHandlerSucceedsEvenIfEnqueueFailsForOneUser(t *testing.T) {
	repo := &fakeRepository{activeUserIDs: []string{"user-1"}}
	enqueuer := &fakeEnqueuer{err: errors.New("redis down")}
	h := briefingworker.NewDispatchHandler(repo, enqueuer, noopLogger())

	err := h.HandleDispatch(context.Background(), newDispatchTask(t, "morning"))
	require.NoError(t, err)
}
