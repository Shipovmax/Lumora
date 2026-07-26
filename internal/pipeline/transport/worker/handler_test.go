package pipelineworker_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/pipeline/domain"
	"github.com/Shipovmax/Lumora/internal/pipeline/service"
	pipelineworker "github.com/Shipovmax/Lumora/internal/pipeline/transport/worker"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
)

// fakeRepository — minimal in-memory domain.Repository, same shape as
// internal/pipeline/service/service_test.go's, redeclared here since it's a
// different test package.
type fakeRepository struct {
	posts  map[string]domain.PostRef
	events map[string]domain.Event
	nextID int
}

func newFakeRepository(posts []domain.PostRef) *fakeRepository {
	r := &fakeRepository{posts: map[string]domain.PostRef{}, events: map[string]domain.Event{}}
	for _, p := range posts {
		r.posts[p.ID] = p
	}
	return r
}

func (r *fakeRepository) genID() string {
	r.nextID++
	return fmt.Sprintf("event-%d", r.nextID)
}

func (r *fakeRepository) GetPosts(_ context.Context, postIDs []string) ([]domain.PostRef, error) {
	out := make([]domain.PostRef, 0, len(postIDs))
	for _, id := range postIDs {
		if p, ok := r.posts[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakeRepository) ListRecentEvents(_ context.Context, since time.Time) ([]domain.Event, error) {
	var out []domain.Event
	for _, e := range r.events {
		if !e.LastSeenAt.Before(since) {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *fakeRepository) GetEventByID(_ context.Context, id string) (domain.Event, error) {
	e, ok := r.events[id]
	if !ok {
		return domain.Event{}, domain.ErrEventNotFound
	}
	return e, nil
}

func (r *fakeRepository) CreateEventWithPost(_ context.Context, topic domain.Topic, title, matchText, postID string, publishedAt time.Time) (domain.Event, error) {
	e := domain.Event{
		ID: r.genID(), Topic: topic, Title: title, MatchText: matchText,
		SourceCount: 1, Importance: 20, FirstSeenAt: publishedAt, LastSeenAt: publishedAt,
	}
	r.events[e.ID] = e
	return e, nil
}

func (r *fakeRepository) AttachPost(_ context.Context, eventID, postID, matchText string, publishedAt time.Time) (domain.Event, error) {
	e := r.events[eventID]
	e.MatchText = matchText
	e.LastSeenAt = publishedAt
	r.events[eventID] = e
	return e, nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandleProcessRejectsInvalidPayload(t *testing.T) {
	h := pipelineworker.NewHandler(nil, noopLogger())

	err := h.HandleProcess(context.Background(), asynq.NewTask(queue.TypePipelineProcess, []byte("not json")))
	require.Error(t, err)
}

func TestHandleProcessCreatesEventFromPosts(t *testing.T) {
	posts := []domain.PostRef{
		{ID: "post-1", SourceID: "src-1", Title: "OpenAI ships new model", Content: "Big AI news today", PublishedAt: time.Now()},
	}
	svc := service.New(newFakeRepository(posts))
	h := pipelineworker.NewHandler(svc, noopLogger())

	payload, err := json.Marshal(queue.PipelineProcessPayload{PostIDs: []string{"post-1"}})
	require.NoError(t, err)

	err = h.HandleProcess(context.Background(), asynq.NewTask(queue.TypePipelineProcess, payload))
	require.NoError(t, err)
}

func TestHandleProcessIsANoOpWhenPostsAreUnknown(t *testing.T) {
	// Empty repo: GetPosts on an unknown post ID returns nothing, so
	// Process has zero posts to cluster — not an error, just a no-op.
	svc := service.New(newFakeRepository(nil))
	h := pipelineworker.NewHandler(svc, noopLogger())

	payload, err := json.Marshal(queue.PipelineProcessPayload{PostIDs: []string{"missing"}})
	require.NoError(t, err)

	err = h.HandleProcess(context.Background(), asynq.NewTask(queue.TypePipelineProcess, payload))
	require.NoError(t, err)
}

type erroringRepository struct {
	*fakeRepository
	err error
}

func (r *erroringRepository) GetPosts(context.Context, []string) ([]domain.PostRef, error) {
	return nil, r.err
}

func TestHandleProcessPropagatesServiceError(t *testing.T) {
	svc := service.New(&erroringRepository{fakeRepository: newFakeRepository(nil), err: errors.New("db unavailable")})
	h := pipelineworker.NewHandler(svc, noopLogger())

	payload, err := json.Marshal(queue.PipelineProcessPayload{PostIDs: []string{"post-1"}})
	require.NoError(t, err)

	err = h.HandleProcess(context.Background(), asynq.NewTask(queue.TypePipelineProcess, payload))
	require.Error(t, err)
}
