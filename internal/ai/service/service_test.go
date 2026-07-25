package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	aidomain "github.com/Shipovmax/Lumora/internal/ai/domain"
	"github.com/Shipovmax/Lumora/internal/ai/service"
	pipelinedomain "github.com/Shipovmax/Lumora/internal/pipeline/domain"
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

type fakeUserContextRepository struct {
	contexts map[string]usercontextdomain.Context
}

func (f *fakeUserContextRepository) GetContext(_ context.Context, userID string) (usercontextdomain.Context, error) {
	c, ok := f.contexts[userID]
	if !ok {
		return usercontextdomain.Context{}, errors.New("no context")
	}
	return c, nil
}

type fakeProvider struct {
	result aidomain.ProviderResult
	err    error

	lastEvent   aidomain.EventInput
	lastContext string
}

func (f *fakeProvider) Explain(_ context.Context, event aidomain.EventInput, userContext string) (aidomain.ProviderResult, error) {
	f.lastEvent = event
	f.lastContext = userContext
	return f.result, f.err
}

type fakeExplanationRepository struct {
	saved aidomain.Explanation
}

func (f *fakeExplanationRepository) SaveExplanation(_ context.Context, exp aidomain.Explanation) (aidomain.Explanation, error) {
	exp.ID = "explanation-1"
	f.saved = exp
	return exp, nil
}

func (f *fakeExplanationRepository) GetExplanation(_ context.Context, eventID, userID string) (aidomain.Explanation, error) {
	if f.saved.EventID == eventID && f.saved.UserID == userID {
		return f.saved, nil
	}
	return aidomain.Explanation{}, aidomain.ErrExplanationNotFound
}

func TestGenerateExplanationPassesEventAndContextToProvider(t *testing.T) {
	ctx := context.Background()

	events := &fakeEventRepository{events: map[string]pipelinedomain.Event{
		"event-1": {ID: "event-1", Topic: pipelinedomain.TopicAI, Title: "OpenAI ships new model", MatchText: "OpenAI ships new model today"},
	}}
	userContexts := &fakeUserContextRepository{contexts: map[string]usercontextdomain.Context{
		"user-1": {UserID: "user-1", Content: "Interested in AI research"},
	}}
	provider := &fakeProvider{result: aidomain.ProviderResult{
		WhatHappened:       "OpenAI released a new model",
		WhyItHappened:      "Competitive pressure",
		WhatChanged:        "New capabilities available",
		WhatItMeansForUser: "Relevant to your AI research interest",
		Model:              "claude-opus-5",
	}}
	explanations := &fakeExplanationRepository{}

	svc := service.New(explanations, events, userContexts, provider)

	result, err := svc.GenerateExplanation(ctx, "event-1", "user-1")
	require.NoError(t, err)
	require.Equal(t, "event-1", result.EventID)
	require.Equal(t, "user-1", result.UserID)
	require.Equal(t, "OpenAI released a new model", result.WhatHappened)
	require.Equal(t, "claude-opus-5", result.Model)
	require.NotEmpty(t, result.ID)

	require.Equal(t, "OpenAI ships new model", provider.lastEvent.Title)
	require.Equal(t, "ai", provider.lastEvent.Topic)
	require.Equal(t, "Interested in AI research", provider.lastContext)
}

func TestGenerateExplanationUnknownEvent(t *testing.T) {
	ctx := context.Background()

	svc := service.New(
		&fakeExplanationRepository{},
		&fakeEventRepository{events: map[string]pipelinedomain.Event{}},
		&fakeUserContextRepository{},
		&fakeProvider{},
	)

	_, err := svc.GenerateExplanation(ctx, "missing", "user-1")
	require.ErrorIs(t, err, pipelinedomain.ErrEventNotFound)
}

func TestGenerateExplanationProviderError(t *testing.T) {
	ctx := context.Background()

	events := &fakeEventRepository{events: map[string]pipelinedomain.Event{
		"event-1": {ID: "event-1", Title: "T"},
	}}
	userContexts := &fakeUserContextRepository{contexts: map[string]usercontextdomain.Context{
		"user-1": {UserID: "user-1"},
	}}
	provider := &fakeProvider{err: errors.New("boom")}

	svc := service.New(&fakeExplanationRepository{}, events, userContexts, provider)

	_, err := svc.GenerateExplanation(ctx, "event-1", "user-1")
	require.Error(t, err)
}
