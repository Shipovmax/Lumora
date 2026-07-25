package service_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	aidomain "github.com/Shipovmax/Lumora/internal/ai/domain"
	"github.com/Shipovmax/Lumora/internal/briefing/domain"
	"github.com/Shipovmax/Lumora/internal/briefing/service"
)

type fakeRepository struct {
	candidates       []domain.CandidateEvent
	createdUserID    string
	createdType      domain.Type
	createdEventIDs  []string
	createBriefingID string
}

func (f *fakeRepository) ListCandidateEvents(_ context.Context, _ string, _ time.Time, limit int) ([]domain.CandidateEvent, error) {
	if len(f.candidates) > limit {
		return f.candidates[:limit], nil
	}
	return f.candidates, nil
}

func (f *fakeRepository) CreateBriefing(_ context.Context, userID string, typ domain.Type, eventIDs []string) (string, time.Time, error) {
	f.createdUserID = userID
	f.createdType = typ
	f.createdEventIDs = eventIDs
	f.createBriefingID = "briefing-1"
	return f.createBriefingID, time.Now(), nil
}

func (f *fakeRepository) ListActiveUserIDs(_ context.Context) ([]string, error) {
	return nil, nil
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

type fakeGenerator struct {
	calls int
	err   error
}

func (f *fakeGenerator) GenerateExplanation(_ context.Context, eventID, userID string) (aidomain.Explanation, error) {
	f.calls++
	if f.err != nil {
		return aidomain.Explanation{}, f.err
	}
	return aidomain.Explanation{
		EventID:      eventID,
		UserID:       userID,
		WhatHappened: fmt.Sprintf("generated for %s", eventID),
	}, nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestBuildOrdersEventsByImportanceAndReusesExistingExplanations(t *testing.T) {
	ctx := context.Background()

	repo := &fakeRepository{candidates: []domain.CandidateEvent{
		{ID: "event-1", Topic: "ai", Title: "First", Importance: 80},
		{ID: "event-2", Topic: "world", Title: "Second", Importance: 40},
	}}
	explanations := &fakeExplanationRepository{explanations: map[string]aidomain.Explanation{
		"event-1:user-1": {EventID: "event-1", UserID: "user-1", WhatHappened: "already generated"},
	}}
	generator := &fakeGenerator{}

	svc := service.New(repo, explanations, generator, noopLogger())

	briefing, err := svc.Build(ctx, "user-1", domain.TypeMorning)
	require.NoError(t, err)
	require.Equal(t, domain.TypeMorning, briefing.Type)
	require.Len(t, briefing.Events, 2)
	require.Equal(t, "event-1", briefing.Events[0].EventID)
	require.Equal(t, "already generated", briefing.Events[0].WhatHappened)
	require.Equal(t, 1, briefing.Events[0].Rank)
	require.Equal(t, "event-2", briefing.Events[1].EventID)
	require.Equal(t, "generated for event-2", briefing.Events[1].WhatHappened)
	require.Equal(t, 2, briefing.Events[1].Rank)

	require.Equal(t, 1, generator.calls, "only event-2 needed generation")
	require.Equal(t, []string{"event-1", "event-2"}, repo.createdEventIDs)
}

func TestBuildReturnsErrNoRelevantEventsWhenNoCandidates(t *testing.T) {
	ctx := context.Background()

	svc := service.New(&fakeRepository{}, &fakeExplanationRepository{}, &fakeGenerator{}, noopLogger())

	_, err := svc.Build(ctx, "user-1", domain.TypeEvening)
	require.ErrorIs(t, err, domain.ErrNoRelevantEvents)
}

func TestBuildSkipsEventsWhereGenerationFails(t *testing.T) {
	ctx := context.Background()

	repo := &fakeRepository{candidates: []domain.CandidateEvent{
		{ID: "event-1", Topic: "ai", Title: "First", Importance: 80},
	}}
	generator := &fakeGenerator{err: errors.New("provider down")}

	svc := service.New(repo, &fakeExplanationRepository{}, generator, noopLogger())

	_, err := svc.Build(ctx, "user-1", domain.TypeMorning)
	require.ErrorIs(t, err, domain.ErrNoRelevantEvents, "all candidates failed generation, nothing to send")
}

func TestBuildRejectsInvalidType(t *testing.T) {
	ctx := context.Background()

	svc := service.New(&fakeRepository{}, &fakeExplanationRepository{}, &fakeGenerator{}, noopLogger())

	_, err := svc.Build(ctx, "user-1", domain.Type("afternoon"))
	require.ErrorIs(t, err, domain.ErrInvalidType)
}
