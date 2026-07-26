package aiworker_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"

	aidomain "github.com/Shipovmax/Lumora/internal/ai/domain"
	"github.com/Shipovmax/Lumora/internal/ai/service"
	aiworker "github.com/Shipovmax/Lumora/internal/ai/transport/worker"
	pipelinedomain "github.com/Shipovmax/Lumora/internal/pipeline/domain"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
	usercontextdomain "github.com/Shipovmax/Lumora/internal/usercontext/domain"
)

type fakeEventRepository struct {
	events map[string]pipelinedomain.Event
}

func (f *fakeEventRepository) GetEventByID(_ context.Context, id string) (pipelinedomain.Event, error) {
	e, ok := f.events[id]
	if !ok {
		return pipelinedomain.Event{}, pipelinedomain.ErrEventNotFound
	}
	return e, nil
}

type fakeUserContextRepository struct{}

func (fakeUserContextRepository) GetContext(_ context.Context, userID string) (usercontextdomain.Context, error) {
	return usercontextdomain.Context{UserID: userID}, nil
}

type fakeProvider struct {
	result aidomain.ProviderResult
	err    error
}

func (f *fakeProvider) Explain(context.Context, aidomain.EventInput, string) (aidomain.ProviderResult, error) {
	return f.result, f.err
}

type fakeExplanationRepository struct{}

func (fakeExplanationRepository) SaveExplanation(_ context.Context, exp aidomain.Explanation) (aidomain.Explanation, error) {
	exp.ID = "explanation-1"
	return exp, nil
}

func (fakeExplanationRepository) GetExplanation(context.Context, string, string) (aidomain.Explanation, error) {
	return aidomain.Explanation{}, aidomain.ErrExplanationNotFound
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandleGenerateRejectsInvalidPayload(t *testing.T) {
	h := aiworker.NewHandler(nil, noopLogger())

	err := h.HandleGenerate(context.Background(), asynq.NewTask(queue.TypeAIGenerate, []byte("not json")))
	require.Error(t, err)
}

func TestHandleGenerateSuccess(t *testing.T) {
	events := &fakeEventRepository{events: map[string]pipelinedomain.Event{
		"event-1": {ID: "event-1", Title: "T", MatchText: "content"},
	}}
	provider := &fakeProvider{result: aidomain.ProviderResult{WhatHappened: "X", Model: "claude-opus-5"}}
	svc := service.New(fakeExplanationRepository{}, events, fakeUserContextRepository{}, provider)
	h := aiworker.NewHandler(svc, noopLogger())

	payload, err := json.Marshal(queue.AIGeneratePayload{EventID: "event-1", UserID: "user-1"})
	require.NoError(t, err)

	err = h.HandleGenerate(context.Background(), asynq.NewTask(queue.TypeAIGenerate, payload))
	require.NoError(t, err)
}

func TestHandleGeneratePropagatesServiceError(t *testing.T) {
	events := &fakeEventRepository{events: map[string]pipelinedomain.Event{}}
	svc := service.New(fakeExplanationRepository{}, events, fakeUserContextRepository{}, &fakeProvider{err: errors.New("provider down")})
	h := aiworker.NewHandler(svc, noopLogger())

	payload, err := json.Marshal(queue.AIGeneratePayload{EventID: "missing", UserID: "user-1"})
	require.NoError(t, err)

	err = h.HandleGenerate(context.Background(), asynq.NewTask(queue.TypeAIGenerate, payload))
	require.ErrorIs(t, err, pipelinedomain.ErrEventNotFound)
}
